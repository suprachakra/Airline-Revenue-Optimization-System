# BookingClass.py
"""
Booking Class Model
-------------------
Standardized model for fare classes with embedded regulatory constraints.
Ensures consistency across pricing and reservation systems.
"""

class BookingClass:
    def __init__(self, code: str, description: str, restrictions: dict):
        self.code = code
        self.description = description
        self.restrictions = restrictions  # E.g., baggage allowance, refund policies
        self.validate()

    def validate(self):
        # Example validation: booking class code should be 2-3 uppercase letters/numbers.
        if not self.code.isalnum() or not self.code.isupper() or not (2 <= len(self.code) <= 3):
            raise ValueError(f"Invalid booking class code: {self.code}")
        # Additional validation for restrictions can be added here.
