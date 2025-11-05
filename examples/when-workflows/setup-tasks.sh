#!/bin/bash
# Setup when tasks for SPARQL service workflows

set -e

FETCHER_PATH="/home/opunix/fetcher/fetcher"
WORKFLOWS_DIR="/home/opunix/sparqlservice/examples/workflows"
WHEN_CMD="when"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Setting up when tasks for SPARQL workflows ===${NC}\n"

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v $WHEN_CMD &> /dev/null; then
    echo -e "${RED}Error: 'when' command not found${NC}"
    echo "Please install when: cd /home/opunix/when && go build -o /usr/local/bin/when cmd/when/main.go"
    exit 1
fi

if [ ! -f "$FETCHER_PATH" ]; then
    echo -e "${RED}Error: fetcher not found at $FETCHER_PATH${NC}"
    echo "Please build fetcher: cd /home/opunix/fetcher && go build -o fetcher cmd/fetcher/main.go"
    exit 1
fi

if [ ! -d "$WORKFLOWS_DIR" ]; then
    echo -e "${RED}Error: Workflows directory not found at $WORKFLOWS_DIR${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites OK${NC}\n"

# Environment variables check
echo "Checking environment variables..."
if [ -z "$POOLPARTY_URL" ]; then
    echo -e "${YELLOW}Warning: POOLPARTY_URL not set${NC}"
fi
if [ -z "$POOLPARTY_USERNAME" ]; then
    echo -e "${YELLOW}Warning: POOLPARTY_USERNAME not set${NC}"
fi
if [ -z "$POOLPARTY_PASSWORD" ]; then
    echo -e "${YELLOW}Warning: POOLPARTY_PASSWORD not set${NC}"
fi
echo ""

# Create tasks
echo -e "${GREEN}Creating when tasks...${NC}\n"

# Task 1: Concept Schemes (daily)
echo "1. Creating sparql-concept-schemes task..."
$WHEN_CMD create sparql-concept-schemes \
  "$FETCHER_PATH semantic $WORKFLOWS_DIR/01-concept-schemes.json" \
  --name "Daily Concept Schemes Query" \
  --schedule "every 24h" \
  --timeout 120 \
  --retry 3 || echo -e "${YELLOW}Task may already exist${NC}"

# Task 2: All Users (every 4 hours)
echo "2. Creating sparql-all-users task..."
$WHEN_CMD create sparql-all-users \
  "$FETCHER_PATH semantic $WORKFLOWS_DIR/02-all-users.json" \
  --name "4-Hourly Users Query" \
  --schedule "every 4h" \
  --timeout 60 \
  --retry 3 || echo -e "${YELLOW}Task may already exist${NC}"

# Task 3: Device Number Info (daily)
echo "3. Creating sparql-device-info task..."
$WHEN_CMD create sparql-device-info \
  "$FETCHER_PATH semantic $WORKFLOWS_DIR/03-device-number-info.json" \
  --name "Daily Device Info Query" \
  --schedule "every 24h" \
  --timeout 120 \
  --retry 3 || echo -e "${YELLOW}Task may already exist${NC}"

# Task 4: Empolis JSONs (weekly)
echo "4. Creating sparql-empolis task..."
$WHEN_CMD create sparql-empolis \
  "$FETCHER_PATH semantic $WORKFLOWS_DIR/05-empolis-jsons.json" \
  --name "Weekly Empolis JSONs Query" \
  --schedule "every 168h" \
  --timeout 180 \
  --retry 3 || echo -e "${YELLOW}Task may already exist${NC}"

echo -e "\n${GREEN}=== Task setup complete ===${NC}\n"

# List created tasks
echo "Created tasks:"
$WHEN_CMD list | grep "sparql-" || echo "No sparql tasks found"

echo -e "\n${GREEN}Usage:${NC}"
echo "  when list                      # List all tasks"
echo "  when run-now sparql-all-users  # Run task manually"
echo "  when logs sparql-all-users     # View task logs"
echo "  when status                    # Check running tasks"
echo ""
echo -e "${YELLOW}Note: Ensure when-daemon is running and sparqlservice is available on port 8091${NC}"
