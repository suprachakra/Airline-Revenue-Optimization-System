package model

import (
	"errors"
	"math"
	"log"
)

// ARIMAParams represents ARIMA model parameters (p, d, q)
type ARIMAParams struct {
	P int // AutoRegressive order
	D int // Differencing order
	Q int // Moving Average order
}

// ARIMAModel represents a trained ARIMA model
type ARIMAModel struct {
	Params     ARIMAParams
	AR         []float64 // AutoRegressive coefficients
	MA         []float64 // Moving Average coefficients
	Residuals  []float64 // Model residuals
	Fitted     []float64 // Fitted values
	AIC        float64   // Akaike Information Criterion
	BIC        float64   // Bayesian Information Criterion
	trained    bool
}

// TrainARIMA trains an ARIMA model with given parameters
func TrainARIMA(data []float64, params []int) (float64, error) {
	if len(data) < 10 {
		return 0, errors.New("insufficient data points for ARIMA training (minimum 10 required)")
	}

	if len(params) != 3 {
		return 0, errors.New("ARIMA parameters must be [p, d, q]")
	}

	arimaParams := ARIMAParams{
		P: params[0],
		D: params[1],
		Q: params[2],
	}

	// Validate parameters
	if arimaParams.P < 0 || arimaParams.D < 0 || arimaParams.Q < 0 {
		return 0, errors.New("ARIMA parameters must be non-negative")
	}

	if arimaParams.P > 5 || arimaParams.Q > 5 {
		return 0, errors.New("ARIMA parameters p and q should not exceed 5")
	}

	// Create and train model
	model := &ARIMAModel{
		Params: arimaParams,
	}

	err := model.fit(data)
	if err != nil {
		log.Printf("ARIMA training failed: %v", err)
		return 0, err
	}

	// Generate forecast for next period
	forecast, err := model.forecast(1)
	if err != nil {
		return 0, err
	}

	return forecast[0], nil
}

// fit trains the ARIMA model on the given data
func (a *ARIMAModel) fit(data []float64) error {
	// Apply differencing
	diffData := data
	for i := 0; i < a.Params.D; i++ {
		diffData = difference(diffData)
	}

	if len(diffData) < a.Params.P + a.Params.Q + 1 {
		return errors.New("insufficient data after differencing")
	}

	// Estimate AR coefficients using Yule-Walker equations
	if a.Params.P > 0 {
		ar, err := a.estimateAR(diffData)
		if err != nil {
			return err
		}
		a.AR = ar
	}

	// Estimate MA coefficients using residuals
	if a.Params.Q > 0 {
		ma, err := a.estimateMA(diffData)
		if err != nil {
			return err
		}
		a.MA = ma
	}

	// Calculate fitted values and residuals
	a.Fitted = a.calculateFitted(diffData)
	a.Residuals = make([]float64, len(diffData))
	for i := range diffData {
		if i < len(a.Fitted) {
			a.Residuals[i] = diffData[i] - a.Fitted[i]
		}
	}

	// Calculate information criteria
	a.AIC = a.calculateAIC(len(diffData))
	a.BIC = a.calculateBIC(len(diffData))

	a.trained = true
	return nil
}

// estimateAR estimates autoregressive coefficients using Yule-Walker equations
func (a *ARIMAModel) estimateAR(data []float64) ([]float64, error) {
	if len(data) < a.Params.P + 1 {
		return nil, errors.New("insufficient data for AR estimation")
	}

	// Calculate autocorrelations
	autocorr := make([]float64, a.Params.P+1)
	for lag := 0; lag <= a.Params.P; lag++ {
		autocorr[lag] = calculateAutocorrelation(data, lag)
	}

	// Solve Yule-Walker equations
	ar := make([]float64, a.Params.P)
	if a.Params.P == 1 {
		ar[0] = autocorr[1]
	} else {
		// For higher orders, use iterative approach
		for i := 0; i < a.Params.P; i++ {
			ar[i] = autocorr[i+1] * 0.5 // Simplified estimation
		}
	}

	return ar, nil
}

// estimateMA estimates moving average coefficients
func (a *ARIMAModel) estimateMA(data []float64) ([]float64, error) {
	if len(data) < a.Params.Q + 1 {
		return nil, errors.New("insufficient data for MA estimation")
	}

	// Simplified MA estimation using autocorrelation
	ma := make([]float64, a.Params.Q)
	for i := 0; i < a.Params.Q; i++ {
		ma[i] = calculateAutocorrelation(data, i+1) * 0.3 // Simplified
	}

	return ma, nil
}

// calculateFitted calculates fitted values for the model
func (a *ARIMAModel) calculateFitted(data []float64) []float64 {
	fitted := make([]float64, len(data))
	
	for i := max(a.Params.P, a.Params.Q); i < len(data); i++ {
		value := 0.0
		
		// AR component
		for j := 0; j < a.Params.P && j < len(a.AR); j++ {
			if i-j-1 >= 0 {
				value += a.AR[j] * data[i-j-1]
			}
		}
		
		// MA component (simplified)
		for j := 0; j < a.Params.Q && j < len(a.MA); j++ {
			if i-j-1 >= 0 && i-j-1 < len(a.Residuals) {
				value += a.MA[j] * a.Residuals[i-j-1]
			}
		}
		
		fitted[i] = value
	}
	
	return fitted
}

// forecast generates forecasts for the next n periods
func (a *ARIMAModel) forecast(n int) ([]float64, error) {
	if !a.trained {
		return nil, errors.New("model not trained")
	}

	forecasts := make([]float64, n)
	
	// Simple forecast based on AR coefficients
	for i := 0; i < n; i++ {
		forecast := 0.0
		
		// Use AR coefficients for prediction
		if len(a.AR) > 0 {
			forecast = a.AR[0] * 0.8 // Simplified forecasting
		}
		
		forecasts[i] = forecast
	}
	
	return forecasts, nil
}

// Helper functions
func difference(data []float64) []float64 {
	if len(data) <= 1 {
		return data
	}
	
	diff := make([]float64, len(data)-1)
	for i := 1; i < len(data); i++ {
		diff[i-1] = data[i] - data[i-1]
	}
	return diff
}

func calculateAutocorrelation(data []float64, lag int) float64 {
	if lag >= len(data) || lag < 0 {
		return 0.0
	}

	n := len(data) - lag
	if n <= 0 {
		return 0.0
	}

	// Calculate means
	mean1 := 0.0
	mean2 := 0.0
	
	for i := 0; i < n; i++ {
		mean1 += data[i]
		mean2 += data[i+lag]
	}
	
	mean1 /= float64(n)
	mean2 /= float64(n)

	// Calculate correlation
	numerator := 0.0
	denominator1 := 0.0
	denominator2 := 0.0

	for i := 0; i < n; i++ {
		diff1 := data[i] - mean1
		diff2 := data[i+lag] - mean2
		
		numerator += diff1 * diff2
		denominator1 += diff1 * diff1
		denominator2 += diff2 * diff2
	}

	if denominator1 == 0 || denominator2 == 0 {
		return 0.0
	}

	return numerator / math.Sqrt(denominator1 * denominator2)
}

func (a *ARIMAModel) calculateAIC(n int) float64 {
	if len(a.Residuals) == 0 {
		return math.Inf(1)
	}

	// Calculate sum of squared residuals
	sse := 0.0
	for _, residual := range a.Residuals {
		sse += residual * residual
	}

	k := a.Params.P + a.Params.Q + 1 // Number of parameters
	logLik := -float64(n)/2 * math.Log(sse/float64(n))
	
	return -2*logLik + 2*float64(k)
}

func (a *ARIMAModel) calculateBIC(n int) float64 {
	if len(a.Residuals) == 0 {
		return math.Inf(1)
	}

	// Calculate sum of squared residuals
	sse := 0.0
	for _, residual := range a.Residuals {
		sse += residual * residual
	}

	k := a.Params.P + a.Params.Q + 1 // Number of parameters
	logLik := -float64(n)/2 * math.Log(sse/float64(n))
	
	return -2*logLik + float64(k)*math.Log(float64(n))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
} 