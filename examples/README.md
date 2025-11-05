# SPARQL Service Examples

This directory contains example workflows for the SPARQL Service using Schema.org SearchAction JSON-LD format.

## Workflow Examples

All examples use the `SearchAction` type from Schema.org to represent SPARQL queries semantically.

### Template-Based Queries

These workflows reference SPARQL template files from the `sparql/` directory:

1. **01-concept-schemes.json** - Query all concept schemes
   - Template: `get_concept_schemes.sparql`
   - Parameters: `language=de-DE`

2. **02-all-users.json** - Query all users
   - Template: `get_all_users.sparql`

3. **03-device-number-info.json** - Query device number information
   - Template: `get_device_number_info.sparql`
   - Parameters: `language=de-DE`

4. **04-top-concepts.json** - Query top concepts of a concept scheme
   - Template: `get_top_concepts_of_concept_scheme.sparql`
   - Parameters: `concept_scheme=https://data.zeiss.com/IQS/3578`

5. **05-empolis-jsons.json** - Query Empolis JSON data
   - Template: `get_empolis_jsons.sparql`
   - Parameters: `concept_scheme=https://data.zeiss.com/IQS/4`

6. **06-schema-st4.json** - Query ST4 schema data
   - Template: `get_schema_st4.sparql`
   - Parameters: `concept_scheme=https://data.zeiss.com/IQS/172`

7. **07-user-certificate.json** - Query specific user certificate
   - Template: `get_user_certificate_iqs.sparql`
   - Parameters: `user_id=86774`

### Inline Query Example

8. **08-inline-query-example.json** - Execute inline SPARQL without template
   - Uses `queryInput` field for inline SPARQL query
   - Demonstrates SPARQL-results+json format

## Usage

### Execute a workflow via curl:

```bash
# Set environment variables
export POOLPARTY_URL="https://poolparty.example.com"
export POOLPARTY_USERNAME="admin"
export POOLPARTY_PASSWORD="secret"

# Replace environment variables in workflow
envsubst < workflows/01-concept-schemes.json > /tmp/workflow.json

# Execute via SPARQL service
curl -X POST http://localhost:8091/v1/api/semantic/action \
  -H "Content-Type: application/json" \
  -d @/tmp/workflow.json
```

### Execute using fetcher:

```bash
# Load workflow into fetcher
./fetcher load workflows/01-concept-schemes.json

# Execute workflow
./fetcher execute iqs-concept-schemes
```

## SearchAction Structure

All workflows follow this Schema.org SearchAction structure:

```json
{
  "@context": "https://schema.org",
  "@type": "SearchAction",
  "identifier": "unique-workflow-id",
  "name": "Human-readable name",
  "description": "Description of what this query does",
  "query": {
    "@type": "SearchAction",
    "contentUrl": "template-file.sparql",
    "queryInput": "inline SPARQL query (alternative to contentUrl)",
    "additionalProperty": {
      "param1": "value1",
      "param2": "value2"
    }
  },
  "target": {
    "@type": "DataCatalog",
    "identifier": "ProjectID",
    "url": "https://poolparty.example.com",
    "encodingFormat": "application/rdf+xml",
    "additionalProperty": {
      "username": "admin",
      "password": "secret"
    }
  }
}
```

## Response Format

The service returns a SearchAction with results:

```json
{
  "@context": "https://schema.org",
  "@type": "SearchAction",
  "identifier": "workflow-id",
  "actionStatus": "CompletedActionStatus",
  "result": "<RDF/XML or other format results>"
}
```

## Supported Encoding Formats

- `application/rdf+xml` (default)
- `application/sparql-results+json`
- `application/sparql-results+xml`
- `text/turtle`
- `application/n-triples`

## Environment Variables

The service uses these environment variables:

- `POOLPARTY_URL` - Base URL of PoolParty instance
- `POOLPARTY_USERNAME` - Authentication username
- `POOLPARTY_PASSWORD` - Authentication password
- `SPARQL_TEMPLATE_DIR` - Directory containing SPARQL templates (default: `./sparql`)
- `PORT` - Service port (default: `8091`)
