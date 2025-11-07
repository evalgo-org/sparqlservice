package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

// REST endpoint request types

type QueryRequest struct {
	Query      string                 `json:"query"`
	Endpoint   string                 `json:"endpoint"`
	ProjectID  string                 `json:"projectId,omitempty"`
	Username   string                 `json:"username,omitempty"`
	Password   string                 `json:"password,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Format     string                 `json:"format,omitempty"`
}

// registerRESTEndpoints adds REST endpoints that convert to semantic actions
func registerRESTEndpoints(apiGroup *echo.Group, apiKeyMiddleware echo.MiddlewareFunc) {
	// POST /v1/api/queries - Execute SPARQL query
	apiGroup.POST("/queries", executeSPARQLQueryREST, apiKeyMiddleware)
}

// executeSPARQLQueryREST handles REST POST /v1/api/queries
// Converts to SearchAction and delegates to semantic handler
func executeSPARQLQueryREST(c echo.Context) error {
	var req QueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid request: %v", err)})
	}

	// Validate required fields
	if req.Query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query is required"})
	}
	if req.Endpoint == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "endpoint is required"})
	}

	// Default format
	if req.Format == "" {
		req.Format = "application/rdf+xml"
	}

	// Build query object
	query := map[string]interface{}{
		"@type": "SearchQuery",
		"text":  req.Query,
	}
	if len(req.Parameters) > 0 {
		query["additionalProperty"] = req.Parameters
	}

	// Build target/endpoint object
	target := map[string]interface{}{
		"@type":          "EntryPoint",
		"url":            req.Endpoint,
		"encodingFormat": req.Format,
	}
	if req.ProjectID != "" {
		target["identifier"] = req.ProjectID
	}
	if req.Username != "" {
		target["username"] = req.Username
	}
	if req.Password != "" {
		target["password"] = req.Password
	}

	// Convert to JSON-LD SearchAction
	action := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "SearchAction",
		"query":    query,
		"target":   target,
	}

	return callSemanticHandler(c, action)
}

// callSemanticHandler converts action to JSON and calls the semantic action handler
func callSemanticHandler(c echo.Context, action map[string]interface{}) error {
	// Marshal action to JSON
	actionJSON, err := json.Marshal(action)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to marshal action: %v", err)})
	}

	// Create new request with JSON-LD body
	newReq := c.Request().Clone(c.Request().Context())
	newReq.Body = io.NopCloser(bytes.NewReader(actionJSON))
	newReq.Header.Set("Content-Type", "application/json")

	// Create new context with modified request
	newCtx := c.Echo().NewContext(newReq, c.Response())
	newCtx.SetPath(c.Path())
	newCtx.SetParamNames(c.ParamNames()...)
	newCtx.SetParamValues(c.ParamValues()...)

	// Call the existing semantic action handler
	return handleSemanticAction(newCtx)
}
