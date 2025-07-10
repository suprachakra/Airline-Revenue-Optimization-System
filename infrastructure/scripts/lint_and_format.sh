#!/bin/bash
# lint_and_format.sh - Runs code linters and formatters.
echo "Running linters..."
eslint . --ext .js,.jsx,.ts,.tsx
echo "Formatting code..."
prettier --write .
