# SPARQL Service

> Semantic SPARQL query service using Schema.org SearchAction API

A Go-based HTTP service that provides a semantic API for executing SPARQL queries against PoolParty triple stores using Schema.org vocabulary.

## Features

- **Schema.org SearchAction API** - Semantic representation of SPARQL queries
- **EVE Library Integration** - Uses `eve.evalgo.org/semantic` types and `eve.evalgo.org/db` PoolParty client
- **Template Support** - Parameterized SPARQL queries with Go text/template
- **Inline Queries** - Direct SPARQL execution without templates
- **Multiple Formats** - RDF/XML, SPARQL JSON, Turtle, N-Triples
- **When Orchestration** - Integration with `when` task scheduler

## Architecture

```
when (scheduler)
  → fetcher semantic (HTTP client)
    → sparqlservice:8091 (SearchAction API)
      → EVE PoolPartyClient
        → PoolParty SPARQL endpoint
```

## Installation

### Prerequisites

- Go 1.21+
- EVE library (`eve.evalgo.org`)
- PoolParty instance with SPARQL endpoint

### Build

```bash
cd /home/opunix/sparqlservice
go build -o sparqlservice ./cmd
```

## Usage

### Start the Service

```bash
# Default port 8091
./sparqlservice

# Custom port
PORT=9000 ./sparqlservice

# With custom SPARQL template directory
SPARQL_TEMPLATE_DIR=/path/to/templates ./sparqlservice
```

### Health Check

```bash
curl http://localhost:8091/health
```

### Execute SPARQL Query

#### Using Template

```bash
curl -X POST http://localhost:8091/v1/api/semantic/action \
  -H "Content-Type: application/json" \
  -d @examples/workflows/01-concept-schemes.json
```

#### Using Inline Query

```bash
curl -X POST http://localhost:8091/v1/api/semantic/action \
  -H "Content-Type: application/json" \
  -d @examples/workflows/08-inline-query-example.json
```

### Using fetcher

```bash
# Execute workflow via fetcher
./fetcher semantic examples/workflows/01-concept-schemes.json

# With output formatting
./fetcher semantic examples/workflows/01-concept-schemes.json --format jsonld
```

## API

### POST /v1/api/semantic/action

Executes a SPARQL query using Schema.org SearchAction.

**Request Body** (JSON-LD):
```json
{
  "@context": "https://schema.org",
  "@type": "SearchAction",
  "identifier": "query-id",
  "name": "Query Name",
  "description": "Description",
  "query": {
    "@type": "SearchAction",
    "contentUrl": "template.sparql",
    "queryInput": "SPARQL query string (alternative)",
    "additionalProperty": {
      "param1": "value1"
    }
  },
  "target": {
    "@type": "DataCatalog",
    "identifier": "ProjectID",
    "url": "https://poolparty.example.com",
    "encodingFormat": "application/rdf+xml",
    "additionalProperty": {
      "username": "user",
      "password": "pass"
    }
  }
}
```

**Response** (JSON-LD):
```json
{
  "@context": "https://schema.org",
  "@type": "SearchAction",
  "identifier": "query-id",
  "actionStatus": "CompletedActionStatus",
  "result": "<RDF/XML or other format>"
}
```

## Examples

### Workflow Examples (`examples/workflows/`)

- `01-concept-schemes.json` - Query all concept schemes
- `02-all-users.json` - Query all users
- `03-device-number-info.json` - Query device information
- `04-top-concepts.json` - Query top concepts of scheme
- `05-empolis-jsons.json` - Query Empolis JSON data
- `06-schema-st4.json` - Query ST4 schema data
- `07-user-certificate.json` - Query user certificate
- `08-inline-query-example.json` - Inline SPARQL query

### When Orchestration (`examples/when-workflows/`)

Scheduled SPARQL query execution using `when` task scheduler.

#### Setup Tasks

```bash
cd examples/when-workflows
./setup-tasks.sh
```

#### Manual Task Creation

```bash
when create sparql-users \
  "/home/opunix/fetcher/fetcher semantic /home/opunix/sparqlservice/examples/workflows/02-all-users.json" \
  --name "Hourly Users Query" \
  --schedule "every 4h" \
  --timeout 60
```

#### Manage Tasks

```bash
when list                    # List all tasks
when run-now sparql-users    # Run manually
when logs sparql-users       # View logs
when status                  # Check running tasks
```

## SPARQL Templates

Templates are stored in `/home/opunix/sparqlservice/sparql/`:

- `get_all_users.sparql`
- `get_concept_schemes.sparql`
- `get_device_number_info.sparql`
- `get_top_concepts_of_concept_scheme.sparql`
- `get_empolis_jsons.sparql`
- `get_schema_st4.sparql`
- `get_user_certificate_iqs.sparql`
- And 5 more...

Templates support Go text/template syntax with parameters passed via `additionalProperty`.

## Environment Variables

- `PORT` - HTTP server port (default: 8091)
- `SPARQL_TEMPLATE_DIR` - Template directory (default: ./sparql)
- `POOLPARTY_URL` - Default PoolParty base URL
- `POOLPARTY_USERNAME` - Default username
- `POOLPARTY_PASSWORD` - Default password

## Project Structure

```
sparqlservice/
├── cmd/
│   ├── main.go              # Server entry point
│   └── semantic_api.go      # SearchAction handler
├── sparql/                  # SPARQL template files (12 files)
├── examples/
│   ├── workflows/           # SearchAction workflows (8 files)
│   │   └── README.md
│   └── when-workflows/      # When ScheduledAction workflows
│       ├── README.md
│       └── setup-tasks.sh   # Task setup script
├── go.mod
├── go.sum
├── sparqlservice           # Built binary
└── README.md
```

## Integration with EVE

This service uses EVE library v0.0.17+:

- **Semantic Types**: `eve.evalgo.org/semantic`
  - `SearchAction` - SPARQL query representation
  - `SearchQuery` - Query details (template or inline)
  - `SPARQLEndpoint` - PoolParty project/endpoint

- **Database Client**: `eve.evalgo.org/db`
  - `PoolPartyClient` - SPARQL execution
  - `RunSparQLFromFile` - Template-based queries

## Registration in claude-memory

The service is registered in the claude-memory database:

```sql
SELECT project_id, project_name, description
FROM project_metadata
WHERE project_id = 'sparqlservice';
```

## Related Projects

- **basexservice** - BaseX XML database service (port 8090)
- **fetcher** - HTTP client with semantic support
- **when** - Task scheduler and orchestration
- **eve** - Core library with semantic types and DB clients

## License

Part of the evalgo.org ecosystem.
