package fauxmux

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"slices"
	"time"
)

var ErrInvalidResponseFormat = fmt.Errorf("invalid response format")

type ResponseFormat string

const (
	JSON  ResponseFormat = "json"
	Bytes ResponseFormat = "bytes"
)

type ErrorResponse struct {
	StatusCode     int
	Response       interface{}
	ResponseFormat ResponseFormat
}

type ErrorResponseConfig struct {
	Frequency float64
	Responses []ErrorResponse
}

func (e ErrorResponseConfig) Validate() error {
	if e.Frequency < 0 {
		return fmt.Errorf("frequency cannot be negative")
	}

	if e.Frequency > 1 {
		return fmt.Errorf("frequency cannot be greater than 1")
	}

	if len(e.Responses) == 0 {
		return fmt.Errorf("responses cannot be empty")
	}

	for _, response := range e.Responses {
		if response.StatusCode < 100 || response.StatusCode > 599 {
			return fmt.Errorf("invalid status code")
		}

		if response.Response == nil {
			return fmt.Errorf("response cannot be nil")
		}

		if !slices.Contains([]ResponseFormat{JSON, Bytes}, response.ResponseFormat) {
			return ErrInvalidResponseFormat
		}
	}

	return nil
}

type ListResponseConfig struct {
	MinItems int
	MaxItems int
}

func (l ListResponseConfig) Validate() error {
	if l.MinItems < 0 {
		return fmt.Errorf("min items cannot be negative")
	}

	if l.MaxItems < 0 {
		return fmt.Errorf("max items cannot be negative")
	}

	if l.MaxItems < l.MinItems {
		return fmt.Errorf("max items cannot be less than min items")
	}

	return nil
}

type EndpointConfig struct {
	Method              string
	Path                string
	MinLatency          time.Duration
	MaxLatency          time.Duration
	FakeDataFunc        FakeDataFunc
	ResponseFormat      ResponseFormat
	ListResponseConfig  *ListResponseConfig
	ErrorResponseConfig *ErrorResponseConfig
}

func (e EndpointConfig) Validate() error {
	if e.Method == "" {
		return fmt.Errorf("method cannot be empty")
	}

	if e.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if e.MinLatency < 0 {
		return fmt.Errorf("min latency cannot be negative")
	}

	if e.MaxLatency < 0 {
		return fmt.Errorf("max latency cannot be negative")
	}

	if e.MaxLatency < e.MinLatency {
		return fmt.Errorf("max latency cannot be less than min latency")
	}

	if !slices.Contains([]ResponseFormat{JSON}, e.ResponseFormat) {
		return fmt.Errorf("invalid response format")
	}

	if e.ListResponseConfig != nil {
		if err := e.ListResponseConfig.Validate(); err != nil {
			return err
		}
	}

	if e.ErrorResponseConfig != nil {
		if err := e.ErrorResponseConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func getFakeDataFunc(endpointCfg EndpointConfig) FakeDataFunc {
	if endpointCfg.FakeDataFunc != nil {
		return endpointCfg.FakeDataFunc
	}
	return config.FakeDataFunc
}

func shouldTriggerError(errorCfg *ErrorResponseConfig) bool {
	if errorCfg == nil {
		return false
	}
	return rand.Float64() < errorCfg.Frequency
}

func handleErrorResponse(w http.ResponseWriter, endpointCfg EndpointConfig) {
	if endpointCfg.ErrorResponseConfig == nil {
		http.Error(w, "Internal Server Error: empty error config", http.StatusInternalServerError)
		return
	}

	errorCfg := endpointCfg.ErrorResponseConfig
	randErrorResponse := errorCfg.Responses[rand.Intn(len(errorCfg.Responses))]

	w.WriteHeader(randErrorResponse.StatusCode)

	switch endpointCfg.ResponseFormat {
	case Bytes:
		w.Write(randErrorResponse.Response.([]byte))
	case JSON:
		writeJSON(w, randErrorResponse.Response)
	default:
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func getResponseData[T any](endpointCfg EndpointConfig) (*T, error) {
	var response T
	err := getFakeDataFunc(endpointCfg)(&response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func getListResponseData[T any](endpointCfg EndpointConfig) ([]T, error) {
	responseLen := rand.Intn(endpointCfg.ListResponseConfig.MaxItems-endpointCfg.ListResponseConfig.MinItems) + endpointCfg.ListResponseConfig.MinItems
	response := make([]T, 0, responseLen)

	fakeDataFunc := getFakeDataFunc(endpointCfg)
	for i := 0; i < responseLen; i++ {
		var item T
		err := fakeDataFunc(&item)
		if err != nil {
			return nil, err
		}
		response = append(response, item)
	}

	return response, nil
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
