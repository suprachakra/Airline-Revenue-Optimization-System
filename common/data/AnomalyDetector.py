# AnomalyDetector.py
"""
Anomaly Detector
----------------
Uses statistical methods to detect outliers in data.
Returns indices of data points that deviate significantly from the mean.
"""

import numpy as np

def detect_outliers(data, threshold=3.0):
    mean = np.mean(data)
    std = np.std(data)
    z_scores = [(x - mean) / std for x in data]
    return [i for i, z in enumerate(z_scores) if abs(z) > threshold]
