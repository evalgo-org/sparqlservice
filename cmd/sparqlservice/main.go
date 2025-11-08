package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"eve.evalgo.org/web"

	"eve.evalgo.org/common"
	evehttp "eve.evalgo.org/http"
	"eve.evalgo.org/registry"
	"eve.evalgo.org/statemanager"
	"eve.evalgo.org/tracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Initialize logger
	logger := common.ServiceLogger("sparqlservice", "1.0.0")

	e := echo.New()

	// Register EVE corporate identity assets
	web.RegisterAssets(e)

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize tracing (gracefully disabled if unavailable)
	if tracer := tracing.Init(tracing.InitConfig{
		ServiceID:        "sparqlservice",
		DisableIfMissing: true,
	}); tracer != nil {
		e.Use(tracer.Middleware())
	}

	// EVE health check
	e.GET("/health", evehttp.HealthCheckHandler("sparqlservice", "1.0.0"))

	// Documentation endpoint
	e.GET("/v1/api/docs", evehttp.DocumentationHandler(evehttp.ServiceDocConfig{
		ServiceID:    "sparqlservice",
		ServiceName:  "SPARQL Query Service",
		Description:  "SPARQL query execution and RDF data management",
		Version:      "v1",
		Port:         8091,
		Capabilities: []string{"graph-database", "sparql", "rdf", "semantic-query", "state-tracking"},
		Endpoints: []evehttp.EndpointDoc{
			{
				Method:      "POST",
				Path:        "/v1/api/semantic/action",
				Description: "Execute SPARQL queries via semantic actions (primary interface)",
			},
			{
				Method:      "POST",
				Path:        "/v1/api/queries",
				Description: "Execute SPARQL query (REST convenience - converts to SearchAction)",
			},
			{
				Method:      "GET",
				Path:        "/health",
				Description: "Health check endpoint",
			},
		},
	}))

	// Initialize state manager
	sm := statemanager.New(statemanager.Config{
		ServiceName:   "sparqlservice",
		MaxOperations: 100,
	})

	// Register state endpoints
	apiGroup := e.Group("/v1/api")
	sm.RegisterRoutes(apiGroup)

	// API Key middleware
	apiKey := os.Getenv("SPARQL_API_KEY")
	apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)

	// Semantic API endpoint (primary interface)
	apiGroup.POST("/semantic/action", handleSemanticAction, apiKeyMiddleware)

	// REST endpoints (convenience adapters that convert to semantic actions)
	registerRESTEndpoints(apiGroup, apiKeyMiddleware)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	// Auto-register with registry service if REGISTRYSERVICE_API_URL is set
	portInt, _ := strconv.Atoi(port)
	if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
		ServiceID:    "sparqlservice",
		ServiceName:  "SPARQL Query Service",
		Description:  "SPARQL query execution and RDF data management",
		Port:         portInt,
		Directory:    "/home/opunix/sparqlservice",
		Binary:       "sparqlservice",
		Version:      "v1",
		Capabilities: []string{"graph-database", "sparql", "rdf", "semantic-query", "state-tracking"},
		APIVersions: []registry.APIVersion{
			{
				Version:       "v1",
				URL:           fmt.Sprintf("http://localhost:%d/v1", portInt),
				Documentation: fmt.Sprintf("http://localhost:%d/v1/api/docs", portInt),
				IsDefault:     true,
				Status:        "stable",
				ReleaseDate:   "2024-01-01",
				Capabilities:  []string{"graph-database", "sparql", "rdf", "semantic-query"},
			},
		},
	}); err != nil {
		logger.WithError(err).Error("Failed to register with registry")
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting sparqlservice on port %s", port)
		if err := e.Start(":" + port); err != nil {
			logger.WithError(err).Error("Server error")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Unregister from registry
	if err := registry.AutoUnregister("sparqlservice"); err != nil {
		logger.WithError(err).Error("Failed to unregister from registry")
	}

	// Shutdown server
	if err := e.Close(); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}

	logger.Info("Server stopped")
}
