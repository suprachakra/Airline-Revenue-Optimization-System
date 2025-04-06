# Config.py
"""
Configuration Loader
--------------------
Loads and validates configuration settings from environment variables and YAML files.
Prevents misconfiguration by enforcing schema checks.
"""

import os
import yaml

class Config:
    def __init__(self, config_path: str):
        self.config = self.load_config(config_path)
        self.validate_config()

    def load_config(self, path: str) -> dict:
        with open(path, 'r') as f:
            return yaml.safe_load(f)

    def validate_config(self):
        # Example validation: Ensure required keys exist.
        required_keys = ["environments", "security", "compliance"]
        for key in required_keys:
            if key not in self.config:
                raise ValueError(f"Missing required configuration key: {key}")

    def get(self, key: str, default=None):
        return self.config.get(key, default)
