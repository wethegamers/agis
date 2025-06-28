#!/bin/bash
set -e

echo "Running agis-bot integration tests..."

# Example: check if the bot's health endpoint is up (adjust as needed)
curl -f http://localhost:9090/healthz || {
  echo "Health check failed!";
  exit 1;
}

echo "All integration tests passed!"
