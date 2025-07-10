#!/bin/bash
# run_tests.sh - Executes the complete test suite.
echo "Running tests across all services..."
./vendor/bin/phpunit --configuration phpunit.xml
if [ $? -ne 0 ]; then
    echo "Tests failed. Exiting."
    exit 1
fi
echo "All tests passed successfully."
