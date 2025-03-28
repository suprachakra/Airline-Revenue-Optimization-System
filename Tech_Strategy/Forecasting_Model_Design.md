# Forecasting Model Design

## Overview
The forecasting module leverages a combination of ARIMA and LSTM models to provide accurate, real-time demand forecasts. With 83 models continuously retrained using current data, this module underpins our dynamic pricing and network planning decisions.

## Model Catalog

| **Model ID** | **Type**    | **Data Sources**                     | **Retrain Frequency** | **Fallback Strategy**                      |
|--------------|-------------|--------------------------------------|-----------------------|--------------------------------------------|
| FCM-01       | Cargo LSTM  | 18M waybills, sensor data            | Hourly                | 7-day moving average if new data is missing|
| FCM-02       | Crew ARIMA  | 450K crew logs                       | Daily                 | Use previous month pattern if retraining fails |
| FCM-03       | Passenger ARIMA | Historical booking data          | Daily                 | Fall back to static model if anomaly detected |

## Drift Detection & Continuous Training

### Drift Detection
```python
# File: services/forecasting_service/src/drift.py
def check_drift(predictions, actuals):
    ks_stat, _ = ks_2samp(predictions, actuals)
    if ks_stat > 0.3:
        trigger_retrain()  # Automatically trigger retraining via SageMaker
```
- **Description:** Compares current predictions with actual outcomes using a KS-test. If drift exceeds a threshold (e.g., 0.3), an automated retraining job is initiated.

### Continuous Training
```bash
# File: infrastructure/ci-cd/retrain.sh
aws sagemaker create-training-job \
  --model-name qatar_cargo_v4 \
  --input-data-config file://s3://qatar-forecast/2024/ \
  --output-data-config S3OutputPath=s3://qatar-forecast/output/
```
- **Description:** This script automates model retraining using AWS SageMaker, ensuring that models remain up to date.

## Monitoring & Fallback
- **Validation Checkpoints:**  
  Each model's performance is monitored (e.g., using MAPE) with automated alerts if accuracy drops below 90%.
- **Fallback:**  
  Cached forecasts are served during retraining, ensuring continuous operation.

*This comprehensive approach guarantees that our forecasting models maintain high accuracy and reliability even in the event of data inconsistencies or retraining delays.*
```
