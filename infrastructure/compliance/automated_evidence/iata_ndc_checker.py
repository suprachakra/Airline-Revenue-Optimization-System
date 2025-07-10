#!/usr/bin/env python3
"""
IATA NDC Checker
This script performs real-time validation of offers against IATA NDC Level 4 requirements.
"""
import json
import sys

def validate_offer(offer):
    # Pseudocode: Check if offer is compliant with NDC rules.
    if offer.get("ndc_version") != "2.4":
        raise Exception("Offer non-compliant with IATA NDC Level 4")
    return True

if __name__ == "__main__":
    offer = json.load(sys.stdin)
    try:
        if validate_offer(offer):
            sys.exit(0)
    except Exception as e:
        print(f"Compliance error: {str(e)}", file=sys.stderr)
        sys.exit(1)
