# RetryPolicies.py
"""
Exponential Backoff and Retry Strategies
------------------------------------------
Implements exponential backoff for retrying operations in the event of transient failures.
"""

import time
import math

def exponential_backoff(retry_count, base_delay=100):
    """
    Calculate delay (in milliseconds) using exponential backoff.
    """
    return base_delay * math.pow(2, retry_count)

def retry_operation(operation, max_attempts=3, base_delay=100):
    """
    Retry an operation using exponential backoff.
    Raises an Exception if maximum attempts are reached.
    """
    attempt = 0
    while attempt < max_attempts:
        try:
            return operation()
        except Exception:
            delay = exponential_backoff(attempt, base_delay)
            time.sleep(delay / 1000.0)
            attempt += 1
    raise Exception("Maximum retry attempts reached")
