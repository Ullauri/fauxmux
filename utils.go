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
	JSON ResponseFormat = "json"
)

type ListResponseConfig struct {
	MinItems int
	MaxItems int
}

func (l ListResponseConfig) Validate() error {
	if l.MinItems != 0 || l.MaxItems != 0 {
		if l.MinItems < 0 {
			return fmt.Errorf("min items cannot be negative")
		}

		if l.MaxItems < 0 {
			return fmt.Errorf("max items cannot be negative")
		}

		if l.MaxItems < l.MinItems {
			return fmt.Errorf("max items cannot be less than min items")
		}
	}

	return nil
}

type EndpointConfig struct {
	Method             string
	Path               string
	MinLatency         time.Duration
	MaxLatency         time.Duration
	FakeDataFunc       FakeDataFunc
	ResponseFormat     string
	ListResponseConfig ListResponseConfig
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

	if err := e.ListResponseConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func getFakeDataFunc(endpointCfg EndpointConfig) FakeDataFunc {
	if endpointCfg.FakeDataFunc != nil {
		return endpointCfg.FakeDataFunc
	}
	return config.FakeDataFunc
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
