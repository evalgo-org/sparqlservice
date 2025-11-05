package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"eve.evalgo.org/db"
	"eve.evalgo.org/semantic"
	"github.com/labstack/echo/v4"
)

// handleSemanticAction handles Schema.org JSON-LD SearchAction for SPARQL queries
func handleSemanticAction(c echo.Context) error {
	// Read request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to read request body: %v", err))
	}

	// Parse using EVE library
	action, err := semantic.ParseSPARQLAction(body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to parse SearchAction: %v", err))
	}

	// Execute the SPARQL query
	return executeSearchAction(c, action)
}

// executeSearchAction executes a SPARQL query (SearchAction)
func executeSearchAction(c echo.Context, action *semantic.SearchAction) error {
	// Extract SPARQL endpoint credentials
	baseURL, username, password, projectID, err := semantic.ExtractSPARQLCredentials(action.Target)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to extract endpoint credentials: %v", err))
	}

	// Extract query template or inline query
	templatePath, inlineQuery, params, err := semantic.ExtractQueryTemplate(action.Query)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to extract query: %v", err))
	}

	// Get template directory from environment or use default
	templateDir := os.Getenv("SPARQL_TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "./sparql"
	}

	// Determine content type (encoding format)
	contentType := action.Target.EncodingFormat
	if contentType == "" {
		contentType = "application/rdf+xml" // Default
	}

	var result []byte

	if inlineQuery != "" {
		// Execute inline query directly
		client := db.NewPoolPartyClient(baseURL, username, password, templateDir)
		result, err = client.ExecuteSPARQL(projectID, inlineQuery, contentType)
		if err != nil {
			action.ActionStatus = "FailedActionStatus"
			action.Error = &semantic.PropertyValue{
				Type:  "PropertyValue",
				Name:  "error",
				Value: err.Error(),
			}
			return c.JSON(http.StatusInternalServerError, action)
		}
	} else {
		// Execute query from template
		// Make template path absolute if it's relative
		if !filepath.IsAbs(templatePath) {
			templatePath = filepath.Join(templateDir, templatePath)
		}

		// Get just the filename for the template loader
		templateFile := filepath.Base(templatePath)

		result, err = db.RunSparQLFromFile(baseURL, username, password, projectID,
			filepath.Dir(templatePath), templateFile, contentType, params)
		if err != nil {
			action.ActionStatus = "FailedActionStatus"
			action.Error = &semantic.PropertyValue{
				Type:  "PropertyValue",
				Name:  "error",
				Value: err.Error(),
			}
			return c.JSON(http.StatusInternalServerError, action)
		}
	}

	// Check if result should be written to file
	var outputFile string
	if action.Result != nil {
		// Result could be a map or struct, try to extract contentUrl
		if resultMap, ok := action.Result.(map[string]interface{}); ok {
			if contentUrl, ok := resultMap["contentUrl"].(string); ok {
				outputFile = contentUrl
				fmt.Printf("DEBUG: Found contentUrl in action.Result: %s\n", contentUrl)
			}
		}
	}

	// Check for outputType in target properties (default: "inline")
	outputType := "inline"
	if action.Target.Properties != nil {
		if ot, ok := action.Target.Properties["outputType"].(string); ok {
			outputType = ot
			fmt.Printf("DEBUG: Found outputType in target.Properties: %s\n", outputType)
		}
	}
	fmt.Printf("DEBUG: outputFile='%s', outputType='%s'\n", outputFile, outputType)

	// Write to file if outputFile is specified or outputType is "file"
	if outputFile != "" || outputType == "file" {
		// If no outputFile specified but outputType is "file", generate a default path
		if outputFile == "" {
			outputFile = fmt.Sprintf("/tmp/%s-result.xml", action.Identifier)
		}

		// Ensure parent directory exists
		dir := filepath.Dir(outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			action.ActionStatus = "FailedActionStatus"
			action.Error = &semantic.PropertyValue{
				Type:  "PropertyValue",
				Name:  "error",
				Value: fmt.Sprintf("Failed to create output directory: %v", err),
			}
			return c.JSON(http.StatusInternalServerError, action)
		}

		// Write result to file
		if err := os.WriteFile(outputFile, result, 0644); err != nil {
			action.ActionStatus = "FailedActionStatus"
			action.Error = &semantic.PropertyValue{
				Type:  "PropertyValue",
				Name:  "error",
				Value: fmt.Sprintf("Failed to write result to file: %v", err),
			}
			return c.JSON(http.StatusInternalServerError, action)
		}

		// Create result dataset with file reference
		action.Result = &semantic.XMLDocument{
			Type:           "Dataset",
			Identifier:     fmt.Sprintf("%s-result", action.Identifier),
			EncodingFormat: contentType,
			ContentUrl:     outputFile,
		}
		action.ActionStatus = "CompletedActionStatus"

		// Return action with result pointing to file
		response := map[string]interface{}{
			"@context":     "https://schema.org",
			"@type":        "SearchAction",
			"identifier":   action.Identifier,
			"actionStatus": "CompletedActionStatus",
			"result": map[string]interface{}{
				"@type":          "Dataset",
				"contentUrl":     outputFile,
				"encodingFormat": contentType,
			},
		}

		return c.JSON(http.StatusOK, response)
	}

	// Default: return inline result
	action.Result = &semantic.XMLDocument{
		Type:           "Dataset",
		Identifier:     fmt.Sprintf("%s-result", action.Identifier),
		EncodingFormat: contentType,
	}
	action.ActionStatus = "CompletedActionStatus"

	// Return action with result embedded (inline)
	response := map[string]interface{}{
		"@context":     "https://schema.org",
		"@type":        "SearchAction",
		"identifier":   action.Identifier,
		"actionStatus": "CompletedActionStatus",
		"result":       string(result),
	}

	return c.JSON(http.StatusOK, response)
}
