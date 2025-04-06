#!/bin/bash
# init_local_env.sh - Initializes the local development environment.
echo "Setting up virtualenv and installing dependencies..."
python3 -m venv env
source env/bin/activate
pip install -r requirements.txt
echo "Local environment setup complete."
