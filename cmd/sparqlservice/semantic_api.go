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

	// Parse as SemanticAction
	action, err := semantic.ParseSemanticAction(body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to parse action: %v", err))
	}

	// Dispatch to registered handler using the ActionRegistry
	// No switch statement needed - handlers are registered at startup
	return semantic.Handle(c, action)
}

// executeSearchAction executes a SPARQL query (SearchAction)
func executeSearchAction(c echo.Context, action *semantic.SemanticAction) error {
	// Extract query and endpoint using helpers
	query, err := semantic.GetSearchQueryFromAction(action)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to extract query", err)
	}

	endpoint, err := semantic.GetSPARQLEndpointFromAction(action)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to extract SPARQL endpoint", err)
	}

	// Extract SPARQL endpoint credentials
	baseURL, username, password, projectID, err := semantic.ExtractSPARQLCredentials(endpoint)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to extract endpoint credentials", err)
	}

	// Extract query template or inline query
	templatePath, inlineQuery, params, err := semantic.ExtractQueryTemplate(query)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to extract query", err)
	}

	// Get template directory from environment or use default
	templateDir := os.Getenv("SPARQL_TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "./sparql"
	}

	// Determine content type (encoding format)
	contentType := endpoint.EncodingFormat
	if contentType == "" {
		contentType = "application/rdf+xml" // Default
	}

	var result []byte

	if inlineQuery != "" {
		// Execute inline query directly
		client := db.NewPoolPartyClient(baseURL, username, password, templateDir)
		result, err = client.ExecuteSPARQL(projectID, inlineQuery, contentType)
		if err != nil {
			return semantic.ReturnActionError(c, action, "Failed to execute inline query", err)
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
			return semantic.ReturnActionError(c, action, "Failed to execute query from template", err)
		}
	}

	// Check if result should be written to file
	var outputFile string
	if result, ok := action.Properties["result"]; ok && result != nil {
		// Result could be a map or struct, try to extract contentUrl
		if resultMap, ok := result.(map[string]interface{}); ok {
			if contentUrl, ok := resultMap["contentUrl"].(string); ok {
				outputFile = contentUrl
				fmt.Printf("DEBUG: Found contentUrl in action.Properties[result]: %s\n", contentUrl)
			}
		}
	}

	// Check for outputType in target properties (default: "inline")
	outputType := "inline"
	if endpoint.Properties != nil {
		if ot, ok := endpoint.Properties["outputType"].(string); ok {
			outputType = ot
			fmt.Printf("DEBUG: Found outputType in endpoint.Properties: %s\n", outputType)
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
			return semantic.ReturnActionError(c, action, "Failed to create output directory", err)
		}

		// Write result to file
		if err := os.WriteFile(outputFile, result, 0644); err != nil {
			return semantic.ReturnActionError(c, action, "Failed to write result to file", err)
		}

		// Create result dataset with file reference
		action.Properties["result"] = &semantic.XMLDocument{
			Type:           "Dataset",
			Identifier:     fmt.Sprintf("%s-result", action.Identifier),
			EncodingFormat: contentType,
			ContentUrl:     outputFile,
		}
		semantic.SetSuccessOnAction(action)

		return c.JSON(http.StatusOK, action)
	}

	// Default: return inline result
	action.Properties["result"] = string(result)
	semantic.SetSuccessOnAction(action)

	return c.JSON(http.StatusOK, action)
}
