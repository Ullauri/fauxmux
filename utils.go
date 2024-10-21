package fauxmux

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"
)

var ErrInvalidResponseFormat = fmt.Errorf("invalid response format")

type ResponseFormat string

const (
	JSON ResponseFormat = "json"
)

type EndpointConfig struct {
	Method         string
	Path           string
	MinLatency     time.Duration
	MaxLatency     time.Duration
	FakeDataFunc   FakeDataFunc
	ResponseFormat string
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

	if !slices.Contains([]string{string(JSON)}, e.ResponseFormat) {
		return fmt.Errorf("invalid response format")
	}

	return nil
}

func getResponseData[T any](endpointCfg EndpointConfig) (*T, error) {
	var response T

	if endpointCfg.FakeDataFunc != nil {
		err := endpointCfg.FakeDataFunc(&response)
		if err != nil {
			return nil, fmt.Errorf("internal Server Error: %v", err)
		}
	} else {
		err := config.FakeDataFunc(&response)
		if err != nil {
			return nil, fmt.Errorf("internal Server Error: %v", err)
		}
	}

	return &response, nil
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
