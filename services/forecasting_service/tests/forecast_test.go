package forecasting_test

import (
	"testing"
	"iaros/forecasting_service"
	"github.com/stretchr/testify/assert"
)

func TestForecastModelAccuracy(t *testing.T) {
	sampleData := []float64{100, 105, 110, 115, 120}
	forecast, err := forecasting.ForecastModel(sampleData, "ARIMA")
	assert.NoError(t, err, "ARIMA model should not return an error")
	expected := 125.0  // Expected value for testing purposes.
	assert.InDelta(t, expected, forecast, 10.0, "Forecast should be within acceptable error margin")
}
