# Logger.py
"""
Centralized Logging Utility
---------------------------
Provides structured logging across IAROS with alert triggers.
Integrates with SIEM systems for automated escalation.
"""

import logging

# Configure root logger
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s - %(message)s",
)

def get_logger(name: str) -> logging.Logger:
    return logging.getLogger(name)
