# ChaosMonkey.py
"""
Chaos Monkey
------------
Simulates various failure modes (e.g., network partition, high latency, service timeouts)
to test the resilience of IAROS. Used during chaos testing in development.
"""

import random
import logging

logger = logging.getLogger(__name__)

def inject_failure():
    failure_modes = [
        "network_partition",
        "high_latency",
        "service_timeout",
        "third_party_api_failure"
    ]
    selected = random.choice(failure_modes)
    logger.info(f"Injecting failure: {selected}")
    simulate_failure(selected)

def simulate_failure(mode):
    # Placeholder: Implement actual failure simulation logic as per environment.
    if mode == "network_partition":
        logger.warning("Simulating network partition")
    elif mode == "high_latency":
        logger.warning("Simulating high latency")
    elif mode == "service_timeout":
        logger.warning("Simulating service timeout")
    elif mode == "third_party_api_failure":
        logger.warning("Simulating third-party API failure")
