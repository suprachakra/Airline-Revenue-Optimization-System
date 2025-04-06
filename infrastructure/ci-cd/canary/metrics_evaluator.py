#!/usr/bin/env python3
"""
metrics_evaluator.py

Automatically evaluates key metrics during canary deployments to trigger rollbacks if necessary.
"""

import json
import sys

def evaluate_metrics(metrics_file):
    with open(metrics_file, 'r') as f:
        metrics = json.load(f)
    
    # Example: Check if error rate is below threshold.
    if metrics["error_rate"] > 0.05:
        print("Error rate too high; triggering rollback.")
        sys.exit(1)
    else:
        print("Metrics within acceptable thresholds.")

if __name__ == "__main__":
    metrics_file = sys.argv[1]
    evaluate_metrics(metrics_file)
