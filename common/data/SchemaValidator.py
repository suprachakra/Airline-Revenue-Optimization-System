# SchemaValidator.py
"""
Schema Validator for Flight Data
--------------------------------
Enforces data schema using regular expressions and range checks.
Ensures runtime validation of incoming data against defined schemas.
"""

import re

class DataQualityError(Exception):
    pass

class FlightDataValidator:
    schema = {
        "flight_id": {"type": "string", "regex": "^[A-Z]{2}\\d{4}$"},
        "capacity": {"type": "integer", "min": 0, "max": 850}
    }

    def validate(self, flight_data):
        for field, rules in self.schema.items():
            value = flight_data.get(field)
            if value is None:
                raise DataQualityError(f"Missing field: {field}")
            if rules["type"] == "string":
                if not re.match(rules["regex"], value):
                    raise DataQualityError(f"Invalid format for {field}: {value}")
            elif rules["type"] == "integer":
                if not (rules["min"] <= value <= rules["max"]):
                    raise DataQualityError(f"{field} out of range: {value}")
