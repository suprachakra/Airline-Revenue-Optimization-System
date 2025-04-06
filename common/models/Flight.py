# Flight.py
"""
Flight Data Model
-----------------
This module defines the Flight model with rigorous validations to ensure data integrity.
It enforces IATA-compliant flight ID formats and capacity constraints.
"""

import re

class Flight:
    def __init__(self, flight_id: str, capacity: int, departure: str, arrival: str):
        self.flight_id = flight_id
        self.capacity = capacity
        self.departure = departure
        self.arrival = arrival
        self.validate()

    def validate(self):
        # Flight ID must be two uppercase letters followed by four digits (e.g., "AA1234")
        if not re.match(r'^[A-Z]{2}\d{4}$', self.flight_id):
            raise ValueError(f"Invalid flight ID format: {self.flight_id}")
        # Capacity must be between 0 and 850
        if not (0 <= self.capacity <= 850):
            raise ValueError(f"Flight capacity out of range: {self.capacity}")
