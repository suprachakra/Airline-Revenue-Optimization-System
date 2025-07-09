package model

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

// LSTMCell represents a single LSTM cell
type LSTMCell struct {
	WeightInputForgettable [][]float64
	WeightHiddenForgettable [][]float64
	BiasForgettable        []float64
	WeightInputInput       [][]float64
	WeightHiddenInput      [][]float64
	BiasInput              []float64
	WeightInputOutput      [][]float64
	WeightHiddenOutput     [][]float64
	BiasOutput             []float64
	WeightInputCandidate   [][]float64
	WeightHiddenCandidate  [][]float64
	BiasCandidate          []float64
	HiddenSize             int
	InputSize              int
}

// LSTMModel represents an LSTM neural network for time series forecasting
type LSTMModel struct {
	Cells         []LSTMCell
	OutputWeights [][]float64
	OutputBias    []float64
	NumLayers     int
	HiddenSize    int
	InputSize     int
	OutputSize    int
	LearningRate  float64
	Epochs        int
	trained       bool
}

// LSTMState represents the state of an LSTM cell
type LSTMState struct {
	Hidden []float64
	Cell   []float64
}

// BuildLSTM creates and trains an LSTM model for time series forecasting
func BuildLSTM(data []float64) (float64, error) {
	if len(data) < 20 {
		return 0, errors.New("insufficient data points for LSTM training (minimum 20 required)")
	}

	// Create model with default parameters
	model := &LSTMModel{
		NumLayers:    2,
		HiddenSize:   64,
		InputSize:    1,
		OutputSize:   1,
		LearningRate: 0.001,
		Epochs:       100,
	}

	// Initialize model
	err := model.initialize()
	if err != nil {
		return 0, err
	}

	// Prepare training data
	trainData, err := model.prepareData(data)
	if err != nil {
		return 0, err
	}

	// Train the model
	err = model.train(trainData)
	if err != nil {
		return 0, err
	}

	// Generate forecast
	forecast, err := model.predict(data[len(data)-10:]) // Use last 10 points for prediction
	if err != nil {
		return 0, err
	}

	return forecast, nil
}

// initialize initializes the LSTM model parameters
func (l *LSTMModel) initialize() error {
	rand.Seed(time.Now().UnixNano())

	// Initialize LSTM cells
	l.Cells = make([]LSTMCell, l.NumLayers)
	for i := 0; i < l.NumLayers; i++ {
		inputSize := l.InputSize
		if i > 0 {
			inputSize = l.HiddenSize
		}

		cell := LSTMCell{
			HiddenSize: l.HiddenSize,
			InputSize:  inputSize,
		}

		// Initialize weights and biases
		cell.WeightInputForgettable = l.initializeMatrix(l.HiddenSize, inputSize)
		cell.WeightHiddenForgettable = l.initializeMatrix(l.HiddenSize, l.HiddenSize)
		cell.BiasForgettable = l.initializeVector(l.HiddenSize)

		cell.WeightInputInput = l.initializeMatrix(l.HiddenSize, inputSize)
		cell.WeightHiddenInput = l.initializeMatrix(l.HiddenSize, l.HiddenSize)
		cell.BiasInput = l.initializeVector(l.HiddenSize)

		cell.WeightInputOutput = l.initializeMatrix(l.HiddenSize, inputSize)
		cell.WeightHiddenOutput = l.initializeMatrix(l.HiddenSize, l.HiddenSize)
		cell.BiasOutput = l.initializeVector(l.HiddenSize)

		cell.WeightInputCandidate = l.initializeMatrix(l.HiddenSize, inputSize)
		cell.WeightHiddenCandidate = l.initializeMatrix(l.HiddenSize, l.HiddenSize)
		cell.BiasCandidate = l.initializeVector(l.HiddenSize)

		l.Cells[i] = cell
	}

	// Initialize output layer
	l.OutputWeights = l.initializeMatrix(l.OutputSize, l.HiddenSize)
	l.OutputBias = l.initializeVector(l.OutputSize)

	return nil
}

// initializeMatrix creates a matrix with random values
func (l *LSTMModel) initializeMatrix(rows, cols int) [][]float64 {
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
		for j := range matrix[i] {
			matrix[i][j] = (rand.Float64() - 0.5) * 0.1 // Small random values
		}
	}
	return matrix
}

// initializeVector creates a vector with random values
func (l *LSTMModel) initializeVector(size int) []float64 {
	vector := make([]float64, size)
	for i := range vector {
		vector[i] = (rand.Float64() - 0.5) * 0.1
	}
	return vector
}

// prepareData prepares the data for training
func (l *LSTMModel) prepareData(data []float64) ([][]float64, error) {
	if len(data) < 10 {
		return nil, errors.New("insufficient data for sequence preparation")
	}

	// Create sequences of length 10 for training
	sequenceLength := 10
	sequences := make([][]float64, 0)

	for i := 0; i <= len(data)-sequenceLength-1; i++ {
		sequence := make([]float64, sequenceLength+1) // +1 for target
		copy(sequence, data[i:i+sequenceLength+1])
		sequences = append(sequences, sequence)
	}

	return sequences, nil
}

// train trains the LSTM model
func (l *LSTMModel) train(trainData [][]float64) error {
	if len(trainData) == 0 {
		return errors.New("no training data provided")
	}

	// Simplified training loop
	for epoch := 0; epoch < l.Epochs; epoch++ {
		totalLoss := 0.0

		for _, sequence := range trainData {
			// Forward pass
			prediction, states := l.forward(sequence[:len(sequence)-1])
			target := sequence[len(sequence)-1]

			// Calculate loss (MSE)
			loss := (prediction - target) * (prediction - target)
			totalLoss += loss

			// Backward pass (simplified)
			l.backward(prediction, target, states)
		}

		// Early stopping if loss is small enough
		avgLoss := totalLoss / float64(len(trainData))
		if avgLoss < 0.001 {
			break
		}
	}

	l.trained = true
	return nil
}

// forward performs forward pass through the LSTM
func (l *LSTMModel) forward(sequence []float64) (float64, [][]LSTMState) {
	states := make([][]LSTMState, l.NumLayers)
	
	// Initialize states
	for i := 0; i < l.NumLayers; i++ {
		states[i] = make([]LSTMState, len(sequence))
		for j := range states[i] {
			states[i][j] = LSTMState{
				Hidden: make([]float64, l.HiddenSize),
				Cell:   make([]float64, l.HiddenSize),
			}
		}
	}

	// Process sequence
	for t := 0; t < len(sequence); t++ {
		input := []float64{sequence[t]}

		// Forward through layers
		for layerIdx := 0; layerIdx < l.NumLayers; layerIdx++ {
			if t > 0 {
				states[layerIdx][t] = l.forwardCell(
					&l.Cells[layerIdx],
					input,
					states[layerIdx][t-1],
				)
			} else {
				states[layerIdx][t] = l.forwardCell(
					&l.Cells[layerIdx],
					input,
					LSTMState{
						Hidden: make([]float64, l.HiddenSize),
						Cell:   make([]float64, l.HiddenSize),
					},
				)
			}
			
			// Output of this layer becomes input to next layer
			input = states[layerIdx][t].Hidden
		}
	}

	// Get final output
	lastHidden := states[l.NumLayers-1][len(sequence)-1].Hidden
	output := l.computeOutput(lastHidden)

	return output, states
}

// forwardCell performs forward pass through a single LSTM cell
func (l *LSTMModel) forwardCell(cell *LSTMCell, input []float64, prevState LSTMState) LSTMState {
	// Forget gate
	forgetGate := l.sigmoid(l.matrixVectorMultiply(cell.WeightInputForgettable, input),
		l.matrixVectorMultiply(cell.WeightHiddenForgettable, prevState.Hidden),
		cell.BiasForgettable)

	// Input gate
	inputGate := l.sigmoid(l.matrixVectorMultiply(cell.WeightInputInput, input),
		l.matrixVectorMultiply(cell.WeightHiddenInput, prevState.Hidden),
		cell.BiasInput)

	// Candidate values
	candidateValues := l.tanh(l.matrixVectorMultiply(cell.WeightInputCandidate, input),
		l.matrixVectorMultiply(cell.WeightHiddenCandidate, prevState.Hidden),
		cell.BiasCandidate)

	// Update cell state
	newCellState := make([]float64, len(prevState.Cell))
	for i := range newCellState {
		newCellState[i] = forgetGate[i]*prevState.Cell[i] + inputGate[i]*candidateValues[i]
	}

	// Output gate
	outputGate := l.sigmoid(l.matrixVectorMultiply(cell.WeightInputOutput, input),
		l.matrixVectorMultiply(cell.WeightHiddenOutput, prevState.Hidden),
		cell.BiasOutput)

	// New hidden state
	newHiddenState := make([]float64, len(outputGate))
	for i := range newHiddenState {
		newHiddenState[i] = outputGate[i] * math.Tanh(newCellState[i])
	}

	return LSTMState{
		Hidden: newHiddenState,
		Cell:   newCellState,
	}
}

// backward performs simplified backward pass
func (l *LSTMModel) backward(prediction, target float64, states [][]LSTMState) {
	// Simplified gradient descent
	error := prediction - target
	
	// Update output weights (simplified)
	for i := range l.OutputWeights {
		for j := range l.OutputWeights[i] {
			if len(states) > 0 && len(states[l.NumLayers-1]) > 0 {
				l.OutputWeights[i][j] -= l.LearningRate * error * states[l.NumLayers-1][len(states[l.NumLayers-1])-1].Hidden[j]
			}
		}
	}

	// Update output bias
	for i := range l.OutputBias {
		l.OutputBias[i] -= l.LearningRate * error
	}
}

// predict generates a prediction for the next time step
func (l *LSTMModel) predict(sequence []float64) (float64, error) {
	if !l.trained {
		return 0, errors.New("model not trained")
	}

	if len(sequence) == 0 {
		return 0, errors.New("empty sequence provided")
	}

	prediction, _ := l.forward(sequence)
	return prediction, nil
}

// computeOutput computes the final output from hidden state
func (l *LSTMModel) computeOutput(hidden []float64) float64 {
	output := 0.0
	for i := 0; i < len(l.OutputWeights[0]); i++ {
		output += l.OutputWeights[0][i] * hidden[i]
	}
	output += l.OutputBias[0]
	return output
}

// Helper functions for matrix operations
func (l *LSTMModel) matrixVectorMultiply(matrix [][]float64, vector []float64) []float64 {
	if len(matrix) == 0 || len(matrix[0]) != len(vector) {
		return make([]float64, len(matrix))
	}

	result := make([]float64, len(matrix))
	for i := range result {
		for j := range vector {
			result[i] += matrix[i][j] * vector[j]
		}
	}
	return result
}

func (l *LSTMModel) sigmoid(a, b, bias []float64) []float64 {
	result := make([]float64, len(a))
	for i := range result {
		sum := a[i] + b[i] + bias[i]
		result[i] = 1.0 / (1.0 + math.Exp(-sum))
	}
	return result
}

func (l *LSTMModel) tanh(a, b, bias []float64) []float64 {
	result := make([]float64, len(a))
	for i := range result {
		sum := a[i] + b[i] + bias[i]
		result[i] = math.Tanh(sum)
	}
	return result
} 