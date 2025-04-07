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
	"os"
	"strings"
	"time"

	"github.com/slicken/history"
)

type AI_Data struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume float64 `json:"volume"`
	Body   float64 `json:"body"`
	Hl     float64 `json:"hl"`
	WickUp float64 `json:"wickUp"`
	WickDn float64 `json:"wickDn"`
	Sma8   float64 `json:"sma8"`
	Sma20  float64 `json:"sma20"`
	Sma200 float64 `json:"sma200"`
}

// saveAI_Data saves market data to JSON file for the given symbol
func saveAI_Data(hist *history.History, symbol string) error {
	bars := hist.GetBars(symbol)
	if len(bars) == 0 {
		return fmt.Errorf("no data for symbol %s", symbol)
	}

	// Initialize dataset slice
	dataset := make([]AI_Data, len(bars))

	// Fill dataset
	for i, bar := range bars {
		dataset[i] = AI_Data{
			Time:   bar.T().Unix(),
			Open:   bar.O(),
			Close:  bar.C(),
			High:   bar.H(),
			Low:    bar.L(),
			Volume: bar.V(),
			Body:   bar.Body(),
			Hl:     bar.Range(),
			WickUp: bar.WickUp(),
			WickDn: bar.WickDn(),
		}
		if i >= 8 {
			dataset[i].Sma8 = bars[i-8 : i].SMA(history.C)
		}
		if i >= 20 {
			dataset[i].Sma20 = bars[i-20 : i].SMA(history.C)
		}
		if i >= 200 {
			dataset[i].Sma200 = bars[i-200 : i].SMA(history.C)
		}
	}

	// Create filename
	filename := strings.ReplaceAll(symbol, "/", "_") + ".json"

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	log.Println("saved data for", filename)
	return nil
}

// prepareAI_Data returns 200 bars structured for AI prediction
func prepareAI_Data(bars history.Bars) ([]AI_Data, error) {
	// If there are fewer than 200 bars, return an error
	if len(bars) < 200 {
		return nil, fmt.Errorf("not enough bars, need at least 200, but got %d", len(bars))
	}

	// Create a slice to hold the AI-structured data
	dataset := make([]AI_Data, 200)

	// Process the last 200 bars to fill the dataset
	for i := 0; i < 200; i++ {
		bar := bars[i]

		// Initialize the dataset structure for this bar
		data := AI_Data{
			Time:   bar.T().Unix(),
			Open:   bar.O(),
			Close:  bar.C(),
			High:   bar.H(),
			Low:    bar.L(),
			Volume: bar.V(),
			Body:   bar.Body(),
			Hl:     bar.Range(),
			WickUp: bar.WickUp(),
			WickDn: bar.WickDn(),
		}

		// Calculate SMA values for 8, 20, and 200 periods
		if i >= 8 {
			data.Sma8 = bars[i-8 : i].SMA(history.C) // SMA for 8 periods
		}
		if i >= 20 {
			data.Sma20 = bars[i-20 : i].SMA(history.C) // SMA for 20 periods
		}
		if i >= 200 {
			data.Sma200 = bars[i-200 : i].SMA(history.C) // SMA for 200 periods
		}

		// Add the data to the dataset slice
		dataset[i] = data
	}

	return dataset, nil
}

// Request prediction from Python server
// Takes the symbol and the full dataset (e.g., 200 bars) as input
func reqPrediction(symbol string, dataset []AI_Data) (float64, error) {
	// --- Configuration ---
	const windowSize = 20 // This MUST match the WINDOW_SIZE in train.py and server.py
	const apiBaseURL = "http://localhost:5000"
	// ---

	// 1. Ensure there are enough data points in the input dataset
	if len(dataset) < windowSize {
		return 0, fmt.Errorf("not enough data points provided, expected at least %d, but got %d", windowSize, len(dataset))
	}

	// 2. Get the last 'windowSize' entries (latest bars)
	// Ensure the 'dataset' is sorted chronologically (oldest to newest)
	lastWindowBars := dataset[len(dataset)-windowSize:]
	// log.Printf("Requesting prediction for %s using last %d bars.", symbol, windowSize)

	// 3. Prepare the request data payload (slice of slices)
	//    The inner slice order MUST exactly match the FEATURES list in train.py
	//    Python FEATURES = ['open', 'close', 'high', 'low', 'volume', 'body', 'hl', 'wickUp', 'wickDn', 'sma8', 'sma20', 'sma200']
	requestData := make([][]float64, windowSize)
	for i, bar := range lastWindowBars {
		requestData[i] = []float64{
			bar.Open,   // open
			bar.Close,  // close
			bar.High,   // high
			bar.Low,    // low
			bar.Volume, // volume
			bar.Body,   // body
			bar.Hl,     // hl (Range)
			bar.WickUp, // wickUp
			bar.WickDn, // wickDn
			bar.Sma8,   // sma8
			bar.Sma20,  // sma20
			bar.Sma200, // sma200
		}
		// Optional: Log the first bar's data being sent for debugging
		// if i == 0 {
		//  log.Printf("First bar data being sent: %+v", requestData[i])
		// }
	}

	// 4. Convert the request data to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return 0, fmt.Errorf("error marshalling dataset for %s: %v", symbol, err)
	}

	// 5. Construct the API endpoint URL
	predictURL := fmt.Sprintf("%s/predict/%s", apiBaseURL, symbol)
	// log.Printf("Sending POST request to: %s", predictURL)

	// 6. Make the POST request
	resp, err := http.Post(predictURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("error sending POST request for %s: %v", symbol, err)
	}
	defer resp.Body.Close()

	// 7. Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body for %s: %v", symbol, err)
	}

	// Log raw response for debugging if needed
	// log.Printf("Raw response from server for %s: %s", symbol, string(body))

	// 8. Check HTTP Status Code
	if resp.StatusCode != http.StatusOK {
		// Attempt to get error message from JSON body even if status is bad
		var errorResult map[string]interface{}
		errMsg := fmt.Sprintf("received non-OK status code %d", resp.StatusCode)
		if json.Unmarshal(body, &errorResult) == nil { // If unmarshal works
			if apiError, exists := errorResult["error"]; exists {
				errMsg = fmt.Sprintf("API error (status %d): %v", resp.StatusCode, apiError)
			}
		} else {
			// If body isn't JSON, include the raw body in error (up to a limit)
			rawBodyStr := string(body)
			if len(rawBodyStr) > 200 { // Limit length
				rawBodyStr = rawBodyStr[:200] + "..."
			}
			errMsg = fmt.Sprintf("received non-OK status code %d. Response body: %s", resp.StatusCode, rawBodyStr)
		}
		return 0, fmt.Errorf(errMsg)
	}

	// 9. Attempt to unmarshal the successful response
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		// This might happen if the server returns plain text on success (unlikely with Flask jsonify)
		return 0, fmt.Errorf("error unmarshalling successful response for %s: %v. Body: %s", symbol, err, string(body))
	}

	// 10. Check if the response contains an error message (even with 200 OK, Flask might return an error structure)
	if errorMessage, exists := result["error"]; exists {
		return 0, fmt.Errorf("error message received from Python server for %s: %v", symbol, errorMessage)
	}

	// 11. Check if the response contains the prediction value and assert its type
	if predictionVal, exists := result["prediction"]; exists {
		// Type assert to float64, as Flask jsonify typically converts Python floats to JSON numbers
		if predictionFloat, ok := predictionVal.(float64); ok {
			// log.Printf("Successfully received prediction for %s: %f", symbol, predictionFloat)
			return predictionFloat, nil
		} else {
			// This case might occur if the prediction is returned as an integer or string
			return 0, fmt.Errorf("prediction field found for %s but is not a valid float64 number (type: %T, value: %v)", symbol, predictionVal, predictionVal)
		}
	}

	// 12. If prediction key is missing
	return 0, fmt.Errorf("prediction field not found in the successful response for %s. Body: %s", symbol, string(body))
}

// ----------------------------------------------------------------------------------------------
// P R E D I C T O R   S T R A T E G Y
// ----------------------------------------------------------------------------------------------

type Prediction struct {
	Symbol string
	Time   time.Time
	Price  float64
}

// Predictor test strategy
type Predictor struct {
	predictions []Prediction
	num         int
	win         int
	loss        int
	history.BaseStrategy
}

func NewPredictor() *Predictor {
	return &Predictor{
		predictions: make([]Prediction, 0),
	}
}

// Event Predictor
func (s *Predictor) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	if len(bars) < 201 {
		return history.Event{}, false
	}

	// prepare data for the prediction model
	ai_data, err := prepareAI_Data(bars[1:])
	if err != nil {
		log.Println("Error converting to dataset:", err)
		return history.Event{}, false
	}

	// request prediction from the model (returns predicted price)
	predictedPrice, err := reqPrediction(symbol, ai_data)
	if err != nil {
		log.Println(err)
		return history.Event{}, false
	}

	// add prediction to the list
	s.predictions = append(s.predictions, Prediction{symbol, bars[0].Time, predictedPrice})

	// Check previous prediction accuracy if we have one
	if len(s.predictions) > 1 {
		s.num++
		lastPred := s.predictions[len(s.predictions)-2]
		actualPrice := bars[0].C()
		if (lastPred.Price > bars[1].C() && actualPrice > bars[1].C()) ||
			(lastPred.Price < bars[1].C() && actualPrice < bars[1].C()) {
			s.win++
			log.Printf("%s CORRECT Predicted %f, Actual %f. Win rate: %.2f%%\n", symbol, predictedPrice, bars[0].C(), float64(s.win)/float64(s.num)*100)
		} else {
			s.loss++
			log.Printf("%s WRONG Predicted %f, Actual %f. Win rate: %.2f%%\n", symbol, predictedPrice, bars[0].C(), float64(s.win)/float64(s.num)*100)
		}
	}

	event := history.Event{
		Symbol: symbol,
		Type:   history.OTHER,
		Price:  predictedPrice,
		Time:   bars[0].Time,
	}

	return event, true
}

// Name returns the strategy name
func (s *Predictor) Name() string {
	return "Predictor"
}
