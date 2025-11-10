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

	// Try to parse as SemanticScheduledAction first (has query, target, etc. fields)
	var scheduledAction semantic.SemanticScheduledAction
	if err := semantic.FromJSONLD(body, &scheduledAction); err == nil {
		// Check if this actually has scheduled action fields
		fmt.Printf("DEBUG: Parsed scheduled action - Query: %v, Target: %v\n", scheduledAction.Query != nil, scheduledAction.Target != nil)
		if scheduledAction.Query != nil || scheduledAction.Target != nil {
			// Dispatch with SemanticScheduledAction
			fmt.Printf("DEBUG: Dispatching as SemanticScheduledAction\n")
			return semantic.Handle(c, &scheduledAction)
		}
	}

	// Fall back to regular SemanticAction
	action, err := semantic.ParseSemanticAction(body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to parse action: %v", err))
	}

	// Dispatch to registered handler using the ActionRegistry
	// No switch statement needed - handlers are registered at startup
	return semantic.Handle(c, action)
}

// executeSearchAction executes a SPARQL query (SearchAction)
// Accepts either *semantic.SemanticAction or *semantic.SemanticScheduledAction
func executeSearchAction(c echo.Context, actionInterface interface{}) error {
	fmt.Printf("DEBUG executeSearchAction: called with type %T\n", actionInterface)
	// Extract query and endpoint using helpers
	query, err := semantic.GetSearchQueryFromAction(actionInterface)
	if err != nil {
		// Get the underlying SemanticAction for error reporting
		var action *semantic.SemanticAction
		if sa, ok := actionInterface.(*semantic.SemanticScheduledAction); ok {
			action = &sa.SemanticAction
		} else {
			action = actionInterface.(*semantic.SemanticAction)
		}
		return semantic.ReturnActionError(c, action, "Failed to extract query", err)
	}

	endpoint, err := semantic.GetSPARQLEndpointFromAction(actionInterface)
	if err != nil {
		// Get the underlying SemanticAction for error reporting
		var action *semantic.SemanticAction
		if sa, ok := actionInterface.(*semantic.SemanticScheduledAction); ok {
			action = &sa.SemanticAction
		} else {
			action = actionInterface.(*semantic.SemanticAction)
		}
		return semantic.ReturnActionError(c, action, "Failed to extract SPARQL endpoint", err)
	}

	// Get the underlying SemanticAction for the rest of the function
	var action *semantic.SemanticAction
	if sa, ok := actionInterface.(*semantic.SemanticScheduledAction); ok {
		action = &sa.SemanticAction
	} else {
		action = actionInterface.(*semantic.SemanticAction)
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

		// Use semantic Result structure for file output
		action.Result = &semantic.SemanticResult{
			Type:   "DigitalDocument",
			Format: contentType,
			Value: map[string]interface{}{
				"contentUrl":     outputFile,
				"encodingFormat": contentType,
				"identifier":     fmt.Sprintf("%s-result", action.Identifier),
			},
		}
		semantic.SetSuccessOnAction(action)

		return c.JSON(http.StatusOK, action)
	}

	// Default: return inline result using semantic Dataset
	action.Result = &semantic.SemanticResult{
		Type:   "Dataset",
		Format: contentType,
		Output: string(result), // Raw SPARQL results
	}
	semantic.SetSuccessOnAction(action)

	return c.JSON(http.StatusOK, action)
}
