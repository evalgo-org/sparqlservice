# When Orchestration Workflows for SPARQL Service

This directory contains `when` orchestration workflows for scheduling SPARQL queries.

## Architecture

The integration follows this pattern:

```
when (scheduler)
  → fetcher semantic (HTTP client)
    → sparqlservice (SPARQL API)
      → PoolParty (SPARQL endpoint)
```

## Workflow Structure

Each workflow is a Schema.org `ScheduledAction` that wraps a `SearchAction`:

```json
{
  "@context": "https://schema.org",
  "@type": "ScheduledAction",
  "identifier": "unique-task-id",
  "name": "Human-readable name",
  "description": "Description of scheduled task",
  "object": {
    "@type": "SearchAction",
    "... SearchAction details ..."
  },
  "additionalProperty": {
    "schedule": "every 4h",
    "command": "/path/to/fetcher semantic workflow.json"
  }
}
```

## Available Workflows

### 01-scheduled-concept-schemes.json
- **Schedule**: Daily (every 24h)
- **Description**: Query all concept schemes from IQS
- **Command**: `fetcher semantic 01-concept-schemes.json`

### 02-scheduled-all-users.json
- **Schedule**: Every 4 hours
- **Description**: Query all users from IQS
- **Command**: `fetcher semantic 02-all-users.json`

## Usage with `when`

### 1. Create a scheduled task

```bash
# Start the when daemon if not running
when-daemon &

# Create task from ScheduledAction workflow
when create sparql-concept-schemes \
  "/home/opunix/fetcher/fetcher semantic /home/opunix/sparqlservice/examples/workflows/01-concept-schemes.json" \
  --name "Daily Concept Schemes Query" \
  --schedule "every 24h" \
  --timeout 120

# Create all-users task
when create sparql-all-users \
  "/home/opunix/fetcher/fetcher semantic /home/opunix/sparqlservice/examples/workflows/02-all-users.json" \
  --name "Hourly Users Query" \
  --schedule "every 4h" \
  --timeout 60
```

### 2. Manage scheduled tasks

```bash
# List all tasks
when list

# Run task manually (test before scheduling)
when run-now sparql-concept-schemes

# View execution logs
when logs sparql-concept-schemes
when logs sparql-concept-schemes --tail 10

# Check running tasks
when status

# Disable/enable task
when disable sparql-concept-schemes
when enable sparql-concept-schemes

# Delete task
when delete sparql-concept-schemes
```

## Environment Variables

Before creating tasks, set required environment variables:

```bash
export POOLPARTY_URL="https://poolparty.example.com"
export POOLPARTY_USERNAME="admin"
export POOLPARTY_PASSWORD="secret"
```

## Direct Execution (without when)

You can also execute workflows directly without scheduling:

```bash
# Execute once
/home/opunix/fetcher/fetcher semantic \
  /home/opunix/sparqlservice/examples/workflows/01-concept-schemes.json

# With output formatting
/home/opunix/fetcher/fetcher semantic \
  /home/opunix/sparqlservice/examples/workflows/01-concept-schemes.json \
  --format jsonld
```

## Service Dependencies

1. **sparqlservice**: Must be running on port 8091
   ```bash
   cd /home/opunix/sparqlservice
   PORT=8091 ./sparqlservice
   ```

2. **PoolParty**: SPARQL endpoint must be accessible
   - Default: configured via environment variables
   - Set in workflow JSON or environment

3. **fetcher**: Must be built and available
   ```bash
   cd /home/opunix/fetcher
   go build -o fetcher cmd/fetcher/main.go
   ```

4. **when**: Daemon must be running for scheduled execution
   ```bash
   when-daemon > /var/log/when-daemon.log 2>&1 &
   ```

## Integration with EVE Semantic Types

All workflows use EVE library's semantic types:
- `eve.evalgo.org/semantic.SearchAction` - SPARQL queries
- `eve.evalgo.org/semantic.SPARQLEndpoint` - PoolParty projects
- `eve.evalgo.org/db.PoolPartyClient` - SPARQL execution

This ensures type safety and semantic consistency across the entire stack.

## Troubleshooting

### Task fails immediately
- Check sparqlservice is running: `curl http://localhost:8091/health`
- Verify fetcher exists: `which fetcher` or check `/home/opunix/fetcher/fetcher`
- Test workflow directly: `fetcher semantic workflow.json`

### SPARQL endpoint errors
- Verify PoolParty URL in environment variables
- Check credentials are correct
- Test endpoint connectivity: `curl -u user:pass $POOLPARTY_URL/PoolParty/api/projects`

### Schedule not triggering
- Check when-daemon is running: `ps aux | grep when-daemon`
- View daemon logs: `tail -f /var/log/when-daemon.log`
- Check task status: `when status`
