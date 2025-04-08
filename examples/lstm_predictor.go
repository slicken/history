/*
Summary Table (Approximate Heuristics):

Timeframe	Minimum Bars (Years)	Common Range (Years)	Upper Range (Years)	Focus
1h			~1000-1500 (1.5-2 mo)	~1500-4000 (2-6 mo)		~5000+ (7+ mo)	Recency, short-term patterns, noise
1d			~500-750 (2-3 yrs)		~750-2500 (3-7 yrs)		~2500-3600+ (7-10+ yrs)	Medium-term trends, news reactions
1w			~150-200 (3-4 yrs)		~250-520 (5-10 yrs)		~520-780+ (10-15+ yrs)	Long-term trends, major cycles
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/slicken/history"
)

// saveAI_Data saves market data to JSON file for the given symbol
func saveAI_Data(hist *history.History, symbol string) error {
	hist.GetBars(symbol).WriteJSON(symbol)
	return nil
}

// OHLCV represents a single OHLCV bar for the prediction request.
// The Time field is removed because the Python server doesn't need it.
// The field names MUST match the keys expected by the Python server.
type OHLCV struct {
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume float64 `json:"volume"`
}

// PredictionRequest is the structure of the JSON payload sent to the
// Python prediction server.
type PredictionRequest struct {
	Symbol string  `json:"symbol"`
	OHLCV  []OHLCV `json:"ohlcv"`
}

// PredictionResponse is the structure of the JSON response received from
// the Python prediction server.
type PredictionResponse struct {
	Prediction float64 `json:"prediction"`
}

const (
	// predictionServerURL is the URL of the Python prediction server.
	predictionServerURL = "http://localhost:5000/predict"

	// windowSize is the number of OHLCV bars required for a prediction.
	// This MUST match the WINDOW_SIZE in your Python training script and
	// http_server.py.
	windowSize = 60
)

// reqPrediction fetches a prediction from the Python prediction server for the
// given symbol.
func reqPrediction(symbol string, bars history.Bars) (float64, error) {
	// --- Input Validation ---
	if len(bars) != windowSize {
		return 0, fmt.Errorf("bars must have length %d, but has length %d", windowSize, len(bars))
	}

	// Convert Bars to []OHLCV.  This is necessary to remove the Time field
	// and to ensure that the field names match what the Python server expects.
	ohlcvData := make([]OHLCV, windowSize)
	for i, bar := range bars {
		ohlcvData[i] = OHLCV{
			Open:   bar.Open,
			Close:  bar.Close,
			High:   bar.High,
			Low:    bar.Low,
			Volume: bar.Volume,
		}
	}

	// Create the request payload.
	reqBody := PredictionRequest{
		Symbol: symbol,
		OHLCV:  ohlcvData,
	}

	// Marshal the request body to JSON.
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("error marshalling JSON: %w", err)
	}

	// Create a new HTTP request.
	req, err := http.NewRequest("POST", predictionServerURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return 0, fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request to the Python prediction server.
	client := &http.Client{Timeout: 10 * time.Second} // Set a timeout!
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	// Check the response status code.
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(respBytes))
	}

	// Unmarshal the response body to JSON.
	var predictionResp PredictionResponse
	if err := json.Unmarshal(respBytes, &predictionResp); err != nil {
		return 0, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return predictionResp.Prediction, nil
}

// ----------------------------------------------------------------------------------------------
// P R E D I C T O R   S T R A T E G Y
// ----------------------------------------------------------------------------------------------

type Prediction struct {
	Symbol      string
	Time        time.Time // The *expected* time of the future bar
	Price       float64   // The predicted price
	AnchorPrice float64   // Closing price of the last bar used for prediction
}

// Predictor test strategy
type Predictor struct {
	predictions []Prediction
	num         int
	win         int
	loss        int
	WindowSize  int // Number of bars to use for prediction
}

func NewPredictor(windowSize int) *Predictor {
	return &Predictor{
		predictions: make([]Prediction, 0),
		WindowSize:  windowSize, // Set the window size
	}
}

// Event Predictor
func (s *Predictor) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	// Ensure we have enough bars
	if len(bars) < s.WindowSize { // We don't need +1 anymore
		return history.Event{}, false
	}

	// Get the last 'WindowSize' bars for prediction
	predictionBars := bars[len(bars)-s.WindowSize:]

	// Request prediction from the model (returns predicted price)
	predictedPrice, err := reqPrediction(symbol, predictionBars)
	if err != nil {
		log.Println("Prediction error:", err)
		return history.Event{}, false
	}

	// Time of the *last* bar in the window (used for determining prediction success)
	lastBarTime := predictionBars[len(predictionBars)-1].Time
	lastBarClose := predictionBars[len(predictionBars)-1].Close // Anchor Price

	// Time for the next bar (used for the Event)
	barsPeriod := bars.Period()
	nextBarTime := lastBarTime.Add(barsPeriod)

	// Add prediction to the list
	s.predictions = append(s.predictions, Prediction{
		Symbol:      symbol,
		Time:        nextBarTime, // Save the *future* bar time
		Price:       predictedPrice,
		AnchorPrice: lastBarClose, // Closing price of the last bar
	})

	// **Check for and Evaluate Previous Predictions**
	// Iterate through the *existing* predictions to see if we now have
	// the data to evaluate them.
	for i := 0; i < len(s.predictions)-1; i++ { // Don't check the *last* prediction
		pred := s.predictions[i]

		// Attempt to find the bar corresponding to the prediction time
		index, actualBar := bars.Find(pred.Time)
		if index != -1 {
			// We found the bar!  Evaluate the prediction
			s.num++
			actualPrice := actualBar.Close

			if (pred.Price > pred.AnchorPrice && actualPrice > pred.AnchorPrice) ||
				(pred.Price < pred.AnchorPrice && actualPrice < pred.AnchorPrice) {
				s.win++
				log.Printf("%s CORRECT Predicted %.2f, Actual %.2f. Win rate: %.2f%%\n",
					symbol, pred.Price, actualPrice, float64(s.win)/float64(s.num)*100)
			} else {
				s.loss++
				log.Printf("%s WRONG Predicted %.2f, Actual %.2f. Win rate: %.2f%%\n",
					symbol, pred.Price, actualPrice, float64(s.win)/float64(s.num)*100)
			}

			// Remove the prediction from the list, so we don't evaluate it again
			s.predictions = append(s.predictions[:i], s.predictions[i+1:]...)
			i-- // Adjust index because we removed an element
		}
	}

	// Create the event
	event := history.Event{
		Symbol: symbol,
		Type:   history.FORECAST,
		Name:   "Predict",
		Price:  predictedPrice,
		Time:   nextBarTime, // Time of the *future* bar
	}

	return event, true
}

// Name returns the strategy name
func (s *Predictor) Name() string {
	return "Predictor"
}
