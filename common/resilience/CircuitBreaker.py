# CircuitBreaker.py
"""
Adaptive Circuit Breaker
------------------------
Provides a unified circuit breaker implementation that automatically transitions between states.
It triggers fallback responses if error thresholds are exceeded.
"""

import time

class AdaptiveCircuitBreaker:
    def __init__(self, error_threshold=0.45, reset_timeout=30000):
        self.state = "CLOSED"
        self.error_count = 0
        self.request_count = 0
        self.error_threshold = error_threshold  # Ratio threshold (e.g., 45%)
        self.reset_timeout = reset_timeout  # in milliseconds
        self.next_attempt = 0

    def execute(self, operation):
        self.request_count += 1
        current_time = int(time.time() * 1000)
        if self.state == "OPEN" and (current_time < self.next_attempt):
            return self.fallback()
        try:
            result = operation()
            self._reset()
            return result
        except Exception as e:
            self.error_count += 1
            if self.request_count > 0 and (self.error_count / self.request_count > self.error_threshold):
                self.state = "OPEN"
                self.next_attempt = current_time + self.reset_timeout
            raise e

    def _reset(self):
        self.state = "CLOSED"
        self.error_count = 0
        self.request_count = 0

    def fallback(self):
        # Return a generic fallback response.
        return {"error": "Circuit breaker open, fallback response activated"}
