## Forecasting Model Design
The forecasting module leverages a suite of 83 models to predict demand across various airline operations. Utilizing both ARIMA for linear trend analysis and LSTM networks for complex non-linear patterns, the system continuously retrains models to maintain accuracy. This document details each model category, retraining protocols, drift detection mechanisms, and fallback strategies.

---

### 2.1 Model Catalog

Below are the detailed tables for example like Qatar Airways Forecasting Models. Each table includes additional columns for Retrain Frequency and Fallback Strategy; designed to capture the complete details for each model category, ensuring that every model’s operational parameters and contingency plans are clearly defined.

#### **Cargo Models (22 Models)**

| **Model ID** | **Model Name**                         | **Algorithm**                 | **Data Inputs**                                             | **Retrain Frequency** | **Fallback Strategy**                                    |
|--------------|----------------------------------------|-------------------------------|-------------------------------------------------------------|-----------------------|----------------------------------------------------------|
| CM-01        | Perishables Demand Forecasting         | Gradient Boosting Machines    | 18M historical waybills + temperature sensor data           | Hourly                | Use a 7-day moving average if live data is missing       |
| CM-02        | Pharma Capacity Optimization           | XGBoost                       | IATA CEIV Pharma certifications + cold chain logs           | Daily                 | Revert to previous day's forecast on training failure    |
| CM-03        | Live Animal Transport Planning         | Random Forest                 | Veterinary certificates + IATA LAR regulations              | Daily                 | Fall back to static model configuration                  |
| CM-04        | E-commerce Shipment Trends             | Prophet                       | AliExpress/Amazon seller API data                           | Hourly                | Use a 24-hour historical average if external data delays |
| CM-05        | Dangerous Goods Volume Prediction      | Isolation Forest              | UN hazardous material codes                                 | Daily                 | Default to last known forecast if new data is unavailable  |
| CM-06        | Mail Flow Analysis                     | ARIMA                         | Postal service API integration                              | Daily                 | Use static trend analysis when API calls fail            |
| CM-07        | Temperature-Sensitive Cargo            | LSTM                          | IoT sensor historical data                                  | Hourly                | Use 7-day moving average if sensor data is missing       |
| CM-08        | High-Value Goods Risk Assessment       | Anomaly Detection             | Insurance claim records                                     | Daily                 | Default to previous anomaly threshold values             |
| CM-09        | Automotive Parts Logistics             | Graph Neural Networks         | OEM production schedules                                    | Hourly                | Revert to 12-hour moving average forecast                |
| CM-10        | Fashion Seasonality                    | Clustering                    | Zara/H&M inventory APIs                                     | Daily                 | Use seasonal averages if clustering fails               |
| CM-11        | Electronics Shipping Cycles            | Bayesian Networks             | CES launch dates                                            | Daily                 | Fall back to static pricing model                        |
| CM-12        | Construction Material Demand           | Regression                    | IMF infrastructure investment reports                       | Daily                 | Use previous month’s data if new data is insufficient      |
| CM-13        | Agricultural Export Waves              | Wavelet Analysis              | UN Comtrade data                                            | Daily                 | Default to a 7-day moving average forecast               |
| CM-14        | Seafood Export Patterns                | Survival Analysis             | Fishing quota databases                                     | Daily                 | Revert to last known survival rate model                 |
| CM-15        | Flower Transport Optimization          | Genetic Algorithms            | Dutch Flower Auction clock data                             | Hourly                | Use historical optimization parameters                 |
| CM-16        | Heavy Machinery Routing                | Constraint Programming        | Port clearance times                                        | Daily                 | Fall back to standard routing rules                      |
| CM-17        | Aerospace Parts Urgency                | Reinforcement Learning        | Airbus/Boeing maintenance alerts                            | Hourly                | Default to conservative rate based on historical alerts  |
| CM-18        | Military Cargo Prioritization          | Q-Learning                    | NATO movement orders                                        | Daily                 | Use pre-determined priority index if retraining fails     |
| CM-19        | Humanitarian Aid Logistics             | Agent-Based Modeling          | UN OCHA crisis reports                                      | Daily                 | Revert to historical averages for crisis response         |
| CM-20        | Event-Driven Cargo Spikes              | Change Point Detection        | FIFA/EXPO schedules                                         | Hourly                | Fall back to historical average adjustments              |
| CM-21        | Oil Equipment Logistics                | Time Series                   | Brent Crude futures                                         | Daily                 | Use last known good value from cache                     |
| CM-22        | Automotive Parts Logistics (2nd)       | Graph Neural Networks         | JIT manufacturing schedules                                 | Hourly                | Revert to a 6-hour moving average forecast               |

---

#### **Crew Models (19 Models)**

| **Model ID** | **Model Name**                     | **Algorithm**           | **Data Inputs**                                            | **Retrain Frequency** | **Fallback Strategy**                              |
|--------------|------------------------------------|-------------------------|------------------------------------------------------------|-----------------------|----------------------------------------------------|
| CR-01        | Pilot Fatigue Risk                 | LSTM Networks           | 450K crew rest logs + cockpit voice recordings             | Hourly                | Use 7-day moving average predictions if live data is missing |
| CR-02        | Cabin Crew Scheduling              | Genetic Algorithms      | Pairing history + FAA/CAA rules                             | Daily                 | Revert to historical pairing rules if optimization fails |
| CR-03        | Ground Staff Allocation            | Linear Programming      | Flight turnaround metrics                                   | Daily                 | Use default allocation percentages               |
| CR-04        | Training Capacity Planning         | Monte Carlo             | ICAO recency requirements                                   | Daily                 | Default to average training capacity if simulation fails    |
| CR-05        | Crew Visa Compliance               | Decision Trees          | Embassy processing times                                    | Daily                 | Use historical compliance rates                  |
| CR-06        | Layover Optimization               | Ant Colony              | Hotel availability APIs                                     | Hourly                | Fall back to predefined layover rules            |
| CR-07        | Standby Crew Prediction            | Survival Analysis       | Historical no-show patterns                                 | Daily                 | Use last month’s standby ratios                  |
| CR-08        | Seasonal Staff Forecasting         | SARIMA                  | 10-year seasonal indices                                    | Daily                 | Revert to seasonal baseline if new data unavailable|
| CR-09        | Route Qualification Mapping        | Clustering              | Aircraft type ratings database                              | Daily                 | Use standard mapping if clustering algorithm fails|
| CR-10        | Crew Cost Optimization             | Mixed Integer Programming| Union contract terms                                        | Daily                 | Use previous period’s cost data                    |
| CR-11        | Language Requirement               | NLP                     | Passenger complaint analysis                                | Daily                 | Default to average language proficiency score    |
| CR-12        | Disruption Recovery                | Markov Decision Process | ATC delay statistics                                        | Hourly                | Use historical delay recovery protocols          |
| CR-13        | Long-Term Hiring Needs             | Cohort Analysis         | Retirement eligibility database                             | Monthly               | Use last quarter’s hiring trends                 |
| CR-14        | Crew Productivity                  | DEA (Data Envelopment)  | On-time performance metrics                                 | Daily                 | Revert to previous productivity metrics          |
| CR-15        | Absenteeism Prediction             | Gradient Boosting       | HR health records                                           | Daily                 | Use static absenteeism rates                     |
| CR-16        | Crew Satisfaction                  | Sentiment Analysis      | Internal survey data                                        | Monthly               | Use last survey cycle data                       |
| CR-17        | Multi-Base Utilization             | k-Means                 | Hub data from Doha, Dubai, Istanbul                          | Daily                 | Use averaged hub utilization if clustering fails |
| CR-18        | Performance-Based Assignment       | Reinforcement Learning  | Customer feedback scores                                    | Hourly                | Default to static assignment rules             |
| CR-19        | Crew Health Monitoring             | Wearable IoT            | Fitbit/Apple Watch data                                     | Real-Time             | Use last known healthy metrics if sensor data is delayed |

---

#### **Fuel Models (15 Models)**

| **Model ID** | **Model Name**                      | **Algorithm**                | **Data Inputs**                                 | **Retrain Frequency** | **Fallback Strategy**                              |
|--------------|-------------------------------------|------------------------------|-------------------------------------------------|-----------------------|----------------------------------------------------|
| FM-01        | Fuel Tankering Optimization         | ARIMA + ML                   | 5-year weather patterns + jet stream data       | Daily                 | Use historical fuel consumption averages          |
| FM-02        | Altitude Efficiency                  | Q-Learning                   | Airbus Flight Ops Database                      | Daily                 | Default to standard altitude efficiency metrics    |
| FM-03        | Route-Specific Burn Rate             | Physics-Informed Neural Net  | Aircraft performance models                     | Hourly                | Revert to historical burn rates                   |
| FM-04        | Weather Impact Analysis              | CNN                          | ECMWF weather models                            | Hourly                | Use historical weather impact factors             |
| FM-05        | APU Usage Optimization               | Dynamic Programming          | Ground power availability                       | Daily                 | Use previous day's APU usage data                 |
| FM-06        | Engine Wash Cycles                    | Survival Analysis            | EGT margin trends                               | Daily                 | Default to standard engine wash schedule          |
| FM-07        | Fuel Quality Impact                  | Random Forest                | Fuel test reports                               | Daily                 | Use last known quality metrics                     |
| FM-08        | Taxi Time Optimization               | Monte Carlo                  | Airport congestion data                         | Hourly                | Use average taxi time from last 24 hours           |
| FM-09        | Carbon Emission Tracking             | LCA Models                   | ICAO CORSIA requirements                        | Daily                 | Fall back to baseline emission factors             |
| FM-10        | SAF Adoption Strategy                | System Dynamics              | Neste MY production forecasts                   | Monthly               | Use previous month’s adoption rate                 |
| FM-11        | Fleet Renewal Impact                 | Counterfactual ML            | A350-1000 performance data                      | Monthly               | Default to historical renewal impact               |
| FM-12        | Contrail Avoidance                   | Reinforcement Learning       | Satellite imagery analysis                      | Hourly                | Use precomputed contrail risk factors              |
| FM-13        | Wind Shear Response                  | LSTM                         | TDWR radar data                                 | Hourly                | Default to averaged wind shear data                |
| FM-14        | Fuel Hedging Strategy                | Stochastic Optimization      | NYMEX futures data                              | Daily                 | Use a hedging strategy based on last known futures   |
| FM-15        | Fuel Price Volatility                | GARCH                        | OPEC meeting minutes                            | Daily                 | Default to average price volatility index          |

---

#### **Passenger Models (27 Models)**

| **Model ID** | **Model Name**                      | **Algorithm**                   | **Data Inputs**                                   | **Retrain Frequency** | **Fallback Strategy**                              |
|--------------|-------------------------------------|---------------------------------|---------------------------------------------------|-----------------------|----------------------------------------------------|
| PM-01        | Booking Curve Prediction            | Prophet                         | 9-year booking history                            | Daily                 | Use a 7-day historical moving average              |
| PM-02        | Cancellation Forecasting            | XGBoost                         | Credit score data                                 | Daily                 | Default to last month’s cancellation rate          |
| PM-03        | Ancillary Propensity                | Collaborative Filtering         | 2.4M meal preferences                             | Hourly                | Use a static propensity score if data is delayed   |
| PM-04        | Loyalty Engagement                  | RFM Analysis                    | Qmiles accrual patterns                           | Daily                 | Fall back to average engagement metrics            |
| PM-05        | Premium Upgrade Likelihood          | Logistic Regression             | CRM purchase history                              | Daily                 | Default to baseline upgrade probability            |
| PM-06        | Connecting Traffic Flow             | Graph Theory                    | MIDT data                                         | Hourly                | Revert to historical connectivity ratios           |
| PM-07        | O&D Pair Popularity                 | Community Detection             | GDS search data                                   | Daily                 | Use the previous period's popularity data          |
| PM-08        | Seasonal Travel Patterns            | STL Decomposition               | 15-year seasonal data                             | Daily                 | Fall back to long-term average seasonal patterns    |
| PM-09        | Group Booking Analysis              | Power Law                       | Travel agency contracts                           | Daily                 | Use previous group booking metrics                 |
| PM-10        | Corporate Travel Volume             | Panel Regression                | Dun & Bradstreet data                             | Daily                 | Default to last quarter’s volume                   |
| PM-11        | Leisure/Business Mix                | SVM                             | Booking channel analysis                          | Daily                 | Revert to a balanced mix based on historical data  |
| PM-12        | Advance Booking Windows             | Kaplan-Meier                    | Payment timing logs                               | Daily                 | Use static booking window averages                 |
| PM-13        | Price Elasticity                    | Bayesian Hierarchical           | A/B test results                                  | Daily                 | Default to historical elasticity estimates         |
| PM-14        | Competitor Response                 | Game Theory                     | Sabre AirPrice IQ                                 | Hourly                | Use fallback competitive metrics                   |
| PM-15        | Mobile Conversion                   | Uplift Modeling                 | App session logs                                  | Hourly                | Default to previous conversion rate                |
| PM-16        | Social Media Impact                 | VAR                             | Brandwatch sentiment                              | Daily                 | Use a 7-day moving average of social sentiment       |
| PM-17        | Weather-Driven Demand               | Causal Impact                   | WMO datasets                                      | Hourly                | Fall back to historical weather impacts            |
| PM-18        | Visa Policy Impact                  | Difference-in-Differences       | IATA Timatic                                      | Daily                 | Default to average visa impact metrics             |
| PM-19        | Economic Indicator Model            | VAR                             | IMF World Outlook                                 | Monthly               | Use last month’s economic outlook data             |
| PM-20        | Currency Fluctuation                | Neural Prophet                  | OANDA FX rates                                    | Hourly                | Revert to previous day’s exchange rates            |
| PM-21        | Demographic Trends                  | Cohort Analysis                 | UN population projections                         | Monthly               | Default to long-term demographic averages          |
| PM-22        | New Route Demand                    | Bass Diffusion                  | Google Trends data                                | Daily                 | Use historical route demand patterns               |
| PM-23        | Codeshare Performance               | SHAP Values                     | OTP partner airline data                          | Hourly                | Use a fallback performance index from previous period|
| PM-24        | Alliance Optimization               | Network Analysis                | oneworld performance                              | Daily                 | Default to historical alliance metrics             |
| PM-25        | Interline Utilization               | Association Rules               | Multilateral agreements                           | Daily                 | Fall back to standard utilization rates            |
| PM-26        | Baggage Fee Pricing                 | Elastic Net                     | Checked bag weight data                           | Daily                 | Use historical baggage fee metrics                 |
| PM-27        | Cabin Crew Language                 | Transformer                     | Passenger nationality mix                         | Daily                 | Default to a baseline language proficiency model   |

---

*Note: The full list of 83 models is maintained in our internal repository, with each model’s parameters and performance metrics logged and reviewed daily.*

---

### 2.2 Model Implementation and Retraining

#### ARIMA Model Implementation
- **Usage:**  
  Captures seasonal trends and linear growth. Ideal for steady, predictable components of demand.
- **Example:**
```python
# File: services/forecasting_service/src/ARIMA_Model.py
import statsmodels.api as sm

def train_arima_model(data, order=(2,1,1)):
    model = sm.tsa.ARIMA(data, order=order)
    results = model.fit(disp=0)
    return results

# Fallback: If retraining fails, load the last saved model.
```
- **Fallback:**  
  If training fails, the system loads a pre-saved model from disk and flags an alert.

#### LSTM Model Implementation
- **Usage:**  
   Model complex, non‑linear patterns such as sudden demand spikes or crew fatigue. Adaptable to rapidly changing market dynamics.Forecasts non-linear patterns such as crew fatigue or cargo demand fluctuations.
- **Example:**
```python
# File: services/forecasting_service/src/LSTM_Model.py
import tensorflow as tf
from tensorflow.keras.models import Sequential
from tensorflow.keras.layers import LSTM, Dense, Dropout

def build_lstm_model(input_shape):
    model = Sequential()
    model.add(LSTM(50, return_sequences=True, input_shape=input_shape))
    model.add(Dropout(0.2))
    model.add(LSTM(50))
    model.add(Dropout(0.2))
    model.add(Dense(1))
    model.compile(optimizer='adam', loss='mean_squared_error')
    return model

# Fallback: If real-time retraining is delayed, use predictions from the previous training cycle.
```
- **Fallback:**  
  If real-time retraining isn’t possible, the system falls back to cached LSTM predictions and triggers an automated retraining job.

### Hybrid/Ensemble Models: 
  - *Purpose:* Combine forecasts from multiple techniques for robust, resilient predictions.
  - *Usage:* Enhance overall forecast accuracy by mitigating weaknesses of individual models.

#### Validation Metrics:
- **MAPE (Mean Absolute Percentage Error):**  
  - *Target:* <10% across all models.
- **KS Statistic (Kolmogorov-Smirnov):**  
  - *Target:* <0.3 to ensure no significant drift in predictions.

---

### 2.3 Drift Detection & Continuous Training

#### Drift Detection Mechanism
```python
# File: services/forecasting_service/src/drift.py
from scipy.stats import ks_2samp

def check_drift(predictions, actuals):
    ks_stat, p_value = ks_2samp(predictions, actuals)
    # If the KS statistic exceeds threshold, trigger a retraining process.
    if ks_stat > 0.3:
        trigger_retrain()  # This function interfaces with our automated retraining pipeline
    return ks_stat, p_value
```
- **Explanation:**  
  The KS-test is used to detect significant differences between the distribution of current predictions and actual values. A KS statistic above 0.3 triggers an automatic retraining job.

#### Continuous Retraining Process
- **Process:**  
  - Models are retrained on a scheduled basis using AWS SageMaker pipelines (hourly for high-frequency models; daily for others).
  - Retraining is automatically triggered by performance degradation detected via statistical tests.
- **Retraining Trigger:**  
  - Automated drift detection mechanisms (e.g., KS-test) compare current predictions with actual outcomes.

```bash
# File: infrastructure/ci-cd/retrain.sh
aws sagemaker create-training-job \
  --model-name qatar_cargo_v4 \
  --input-data-config file://s3://qatar-forecast/2024/ \
  --output-data-config S3OutputPath=s3://qatar-forecast/output/ \
  --region us-east-1
```
- **Explanation:**  
  This script automates the retraining of our forecasting models using AWS SageMaker, ensuring that new data is incorporated on a continuous basis.

---

### 2.4 Monitoring and Fallbacks

- **Performance Monitoring:**  
  Each model’s performance (e.g., MAPE, RMSE) is tracked in real time. If accuracy drops below 90%, an alert is triggered, and the system automatically serves cached predictions.
- **Fallback Procedures:**  
  If any model fails to update, the system defaults to using the previous day’s forecast data until the retraining process is successfully completed. If retraining fails or if drift is detected beyond thresholds, the system reverts to a 7‑day moving average forecast. Automated alerts are issued to notify the system of fallback activation for further analysis.

### 2.5 Documentation & Versioning
- **Parameter Logs:**
  Every model's configuration, performance metrics, and version history are stored in our internal repository.
- **Fallback Documentation:**
  Each model includes detailed fallback procedures that are automatically activated when data quality issues are detected.
- **Version Control:**
  Models and parameters are versioned using automated pipelines (e.g., via DVC and S3 versioned buckets) to ensure reproducibility and auditability.

### 2.6. Integration with Downstream Modules
- **Data Flow:** Forecast outputs are exposed via REST/GraphQL endpoints, integrating directly with the Pricing Engine and Network Optimization modules.
- **Automated Validation:** Continuous integration tests validate the consistency and accuracy of forecast data before deployment.
- **Monitoring:** Real-time performance monitoring (via Prometheus and Grafana) ensures that any degradation in forecast accuracy triggers immediate automated fallback actions

### 2.7. Continuous Improvement & Monitoring
- **Automated Alerts:** Performance metrics (e.g., MAPE, KS statistic) are continuously monitored, with automated alerts generated if targets are not met.
- **Scheduled Reviews:** Regular Inspect & Adapt sessions (as part of our SAFe Agile PI Planning) ensure that forecasting models and fallback strategies are updated based on operational feedback.
- **Data Quality Checks:** Automated ETL pipelines and anomaly detection mechanisms ensure high-quality, up-to-date data feeds.

### Summary
This forecasting module is designed to ensure consistent, accurate demand predictions through 83 robust models. With continuous retraining, drift detection, and comprehensive fallback strategies, the system maintains a high forecast accuracy, directly contributing to optimized dynamic pricing and network planning.
```
