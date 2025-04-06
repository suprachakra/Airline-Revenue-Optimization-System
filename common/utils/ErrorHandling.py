# ErrorHandling.py
"""
Error Handling Module
---------------------
Defines standard error classes and helper functions for uniform error responses across IAROS.
"""

class IAROSError(Exception):
    """Base class for all IAROS-related errors."""
    pass

class DataQualityError(IAROSError):
    """Raised when data validation fails."""
    pass

class ComplianceError(IAROSError):
    """Raised when a compliance violation is detected."""
    pass

def handle_error(e: Exception) -> dict:
    # Returns a standardized error response.
    return {
        "error": str(e),
        "type": e.__class__.__name__
    }
