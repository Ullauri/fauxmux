package fauxmux

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestErrorResponseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ErrorResponseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ErrorResponseConfig{
				Frequency: 0.5,
				Responses: []ErrorResponse{
					{StatusCode: 200, Response: "OK", ResponseFormat: JSON},
				},
			},
			wantErr: false,
		},
		{
			name: "negative frequency",
			config: ErrorResponseConfig{
				Frequency: -0.1,
			},
			wantErr: true,
		},
		{
			name: "frequency greater than 1",
			config: ErrorResponseConfig{
				Frequency: 1.1,
			},
			wantErr: true,
		},
		{
			name: "empty responses",
			config: ErrorResponseConfig{
				Frequency: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid status code",
			config: ErrorResponseConfig{
				Frequency: 0.5,
				Responses: []ErrorResponse{
					{StatusCode: 99, Response: "OK", ResponseFormat: JSON},
				},
			},
			wantErr: true,
		},
		{
			name: "nil response",
			config: ErrorResponseConfig{
				Frequency: 0.5,
				Responses: []ErrorResponse{
					{StatusCode: 200, Response: nil, ResponseFormat: JSON},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid response format",
			config: ErrorResponseConfig{
				Frequency: 0.5,
				Responses: []ErrorResponse{
					{StatusCode: 200, Response: "OK", ResponseFormat: "xml"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ErrorResponseConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListResponseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ListResponseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ListResponseConfig{
				MinItems: 1,
				MaxItems: 10,
			},
			wantErr: false,
		},
		{
			name: "negative min items",
			config: ListResponseConfig{
				MinItems: -1,
				MaxItems: 10,
			},
			wantErr: true,
		},
		{
			name: "negative max items",
			config: ListResponseConfig{
				MinItems: 1,
				MaxItems: -10,
			},
			wantErr: true,
		},
		{
			name: "max items less than min items",
			config: ListResponseConfig{
				MinItems: 10,
				MaxItems: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ListResponseConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEndpointConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  EndpointConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: EndpointConfig{
				Method:         "GET",
				Path:           "/test",
				MinLatency:     10 * time.Millisecond,
				MaxLatency:     100 * time.Millisecond,
				ResponseFormat: JSON,
			},
			wantErr: false,
		},
		{
			name: "empty method",
			config: EndpointConfig{
				Path:           "/test",
				MinLatency:     10 * time.Millisecond,
				MaxLatency:     100 * time.Millisecond,
				ResponseFormat: JSON,
			},
			wantErr: true,
		},
		{
			name: "empty path",
			config: EndpointConfig{
				Method:         "GET",
				MinLatency:     10 * time.Millisecond,
				MaxLatency:     100 * time.Millisecond,
				ResponseFormat: JSON,
			},
			wantErr: true,
		},
		{
			name: "negative min latency",
			config: EndpointConfig{
				Method:         "GET",
				Path:           "/test",
				MinLatency:     -10 * time.Millisecond,
				MaxLatency:     100 * time.Millisecond,
				ResponseFormat: JSON,
			},
			wantErr: true,
		},
		{
			name: "negative max latency",
			config: EndpointConfig{
				Method:         "GET",
				Path:           "/test",
				MinLatency:     10 * time.Millisecond,
				MaxLatency:     -100 * time.Millisecond,
				ResponseFormat: JSON,
			},
			wantErr: true,
		},
		{
			name: "max latency less than min latency",
			config: EndpointConfig{
				Method:         "GET",
				Path:           "/test",
				MinLatency:     100 * time.Millisecond,
				MaxLatency:     10 * time.Millisecond,
				ResponseFormat: JSON,
			},
			wantErr: true,
		},
		{
			name: "invalid response format",
			config: EndpointConfig{
				Method:         "GET",
				Path:           "/test",
				MinLatency:     10 * time.Millisecond,
				MaxLatency:     100 * time.Millisecond,
				ResponseFormat: "xml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("EndpointConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		endpointCfg EndpointConfig
		wantStatus  int
		wantBody    string
	}{
		{
			name: "valid error response",
			endpointCfg: EndpointConfig{
				ResponseFormat: JSON,
				ErrorResponseConfig: &ErrorResponseConfig{
					Responses: []ErrorResponse{
						{StatusCode: 500, Response: map[string]string{"error": "internal server error"}, ResponseFormat: JSON},
					},
				},
			},
			wantStatus: 500,
			wantBody:   `{"error":"internal server error"}`,
		},
		{
			name: "empty error config",
			endpointCfg: EndpointConfig{
				ResponseFormat: JSON,
			},
			wantStatus: 500,
			wantBody:   "Internal Server Error: empty error config\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleErrorResponse(w, tt.endpointCfg)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("handleErrorResponse() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}

			if tt.endpointCfg.ResponseFormat == JSON {
				var gotBody map[string]string
				json.Unmarshal(body, &gotBody)
				var wantBody map[string]string
				json.Unmarshal([]byte(tt.wantBody), &wantBody)
				if !reflect.DeepEqual(gotBody, wantBody) {
					t.Errorf("handleErrorResponse() body = %v, want %v", gotBody, wantBody)
				}
			} else {
				if string(body) != tt.wantBody {
					t.Errorf("handleErrorResponse() body = %v, want %v", string(body), tt.wantBody)
				}
			}
		})
	}
}
