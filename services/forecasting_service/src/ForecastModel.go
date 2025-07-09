package forecasting

import (
	"log"
	"errors"
	"iaros/forecasting_service/src/model"
)

// ForecastModel computes a forecast using the specified model type.
func ForecastModel(data []float64, modelType string) (float64, error) {
	var forecast float64
	var err error

	switch modelType {
	case "ARIMA":
		forecast, err = model.TrainARIMA(data, []int{2, 1, 1})
	case "LSTM":
		forecast, err = model.BuildLSTM(data)
	default:
		err = errors.New("unsupported model type")
	}

	if err != nil {
		log.Printf("Error in ForecastModel: %v", err)
		return 0.0, err
	}
	return forecast, nil
}
