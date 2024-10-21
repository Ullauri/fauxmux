package fauxmux

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Error struct {
	Message string `json:"message"`
}

func TestMain(m *testing.M) {
	Setup(Config{
		FakeDataFunc: func(v interface{}) error {
			user := v.(*User)
			user.ID = 1
			user.Name = "Doe"
			user.Email = "doe@testing.com"
			return nil
		},
	})
	exitCode := m.Run()
	os.Exit(exitCode)
}

// TestFauxMuxBasicGET tests basic GET endpoint response
func TestFauxMuxBasicGET(t *testing.T) {
	mux := NewMux()

	err := RegisterEndpoint[User](mux, EndpointConfig{
		Method:         "GET",
		Path:           "/users",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     1000 * time.Millisecond,
		ResponseFormat: "json",
	})
	if err != nil {
		t.Fatalf("failed to register endpoint: %v", err)
	}

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	mux.Mux().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("expected status code %d but got %d", http.StatusOK, status)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected content type application/json but got %s", contentType)
	}

	var user User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	if err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if user.ID == 0 || user.Name == "" || user.Email == "" {
		t.Fatalf("expected non-empty user but got %+v", user)
	}
}

// TestFauxMuxListGET tests GET endpoint with a list response
func TestFauxMuxListGET(t *testing.T) {
	mux := NewMux()

	err := RegisterEndpoint[User](mux, EndpointConfig{
		Method:         "GET",
		Path:           "/users",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     1000 * time.Millisecond,
		ResponseFormat: "json",
		ListResponseConfig: &ListResponseConfig{
			MinItems: 2,
			MaxItems: 5,
		},
	})
	if err != nil {
		t.Fatalf("failed to register endpoint: %v", err)
	}

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	mux.Mux().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("expected status code %d but got %d", http.StatusOK, status)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected content type application/json but got %s", contentType)
	}

	var users []User
	err = json.Unmarshal(w.Body.Bytes(), &users)
	if err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if len(users) < 2 || len(users) > 5 {
		t.Fatalf("expected between 2 and 5 users but got %d", len(users))
	}

	for _, user := range users {
		if user.ID == 0 || user.Name == "" || user.Email == "" {
			t.Fatalf("expected non-empty user but got %+v", user)
		}
	}
}

// TestFauxMuxCustomError tests custom error responses
func TestFauxMuxCustomError(t *testing.T) {
	mux := NewMux()

	err := RegisterEndpoint[User](mux, EndpointConfig{
		Method:         "GET",
		Path:           "/users/error",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     1000 * time.Millisecond,
		ResponseFormat: "json",
		ErrorResponseConfig: &ErrorResponseConfig{
			Frequency: 0.9,
			Responses: []ErrorResponse{
				{
					StatusCode:     500,
					Response:       "Internal Server Error",
					ResponseFormat: Bytes,
				},
				{
					StatusCode:     503,
					Response:       `{"error": "Service Unavailable"}`,
					ResponseFormat: JSON,
				},
				{
					StatusCode:     404,
					Response:       Error{Message: "Not Found"},
					ResponseFormat: JSON,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to register endpoint: %v", err)
	}

	req := httptest.NewRequest("GET", "/users/error", nil)
	w := httptest.NewRecorder()
	mux.Mux().ServeHTTP(w, req)

	switch w.Code {
	case 500:
		if !strings.Contains(w.Body.String(), "Internal") {
			t.Fatalf("expected 'Internal Server Error' but got %s", w.Body.String())
		}
	case 503:
		if !strings.Contains(w.Body.String(), "Unavailable") {
			t.Fatalf("expected '{\"error\": \"Service Unavailable\"}' but got %s", w.Body.String())
		}
	case 404:
		var errRes Error
		err := json.Unmarshal(w.Body.Bytes(), &errRes)
		if err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}
		if errRes.Message != "Not Found" {
			t.Fatalf("expected 'Not Found' but got %s", errRes.Message)
		}
	}
}

// TestFauxMuxLatency tests if latency simulation works
func TestFauxMuxLatency(t *testing.T) {
	mux := NewMux()

	err := RegisterEndpoint[User](mux, EndpointConfig{
		Method:         "GET",
		Path:           "/users",
		MinLatency:     500 * time.Millisecond,
		MaxLatency:     1000 * time.Millisecond,
		ResponseFormat: "json",
	})
	if err != nil {
		t.Fatalf("failed to register endpoint: %v", err)
	}

	start := time.Now()
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	mux.Mux().ServeHTTP(w, req)
	elapsed := time.Since(start)

	if elapsed < 500*time.Millisecond || elapsed > 1000*time.Millisecond {
		t.Fatalf("expected latency between 500ms and 1000ms but got %v", elapsed)
	}

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("expected status code %d but got %d", http.StatusOK, status)
	}
}

// TestFauxMuxListResponseLength tests if the list response returns the correct length
func TestFauxMuxListResponseLength(t *testing.T) {
	mux := NewMux()

	err := RegisterEndpoint[User](mux, EndpointConfig{
		Method:         "GET",
		Path:           "/users",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     1000 * time.Millisecond,
		ResponseFormat: "json",
		ListResponseConfig: &ListResponseConfig{
			MinItems: 5,
			MaxItems: 5,
		},
	})
	if err != nil {
		t.Fatalf("failed to register endpoint: %v", err)
	}

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	mux.Mux().ServeHTTP(w, req)

	var users []User
	err = json.Unmarshal(w.Body.Bytes(), &users)
	if err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if len(users) != 5 {
		t.Fatalf("expected 5 users but got %d", len(users))
	}
}

// TestFauxMuxMultipleMethodsSamePath tests if multiple methods can be registered for the same path
func TestFauxMuxMultipleMethodsSamePath(t *testing.T) {
	// Set up Mux and register a GET and POST endpoint for the same path "/users"
	fs := NewMux()

	err := RegisterEndpoint[User](fs, EndpointConfig{
		Method:         "GET",
		Path:           "/users",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     500 * time.Millisecond,
		ResponseFormat: JSON,
	})
	if err != nil {
		t.Fatalf("failed to register GET endpoint: %v", err)
	}

	err = RegisterEndpoint[User](fs, EndpointConfig{
		Method:         "POST",
		Path:           "/users",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     500 * time.Millisecond,
		ResponseFormat: JSON,
	})
	if err != nil {
		t.Fatalf("failed to register POST endpoint: %v", err)
	}

	server := httptest.NewServer(fs.Mux())
	defer server.Close()

	resp, err := http.Get(server.URL + "/users")
	if err != nil {
		t.Fatalf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code for GET: got %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var userGet User
	err = json.NewDecoder(resp.Body).Decode(&userGet)
	if err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	if userGet.ID == 0 || userGet.Name == "" || userGet.Email == "" {
		t.Fatalf("expected non-empty user for GET but got %+v", userGet)
	}

	resp, err = http.Post(server.URL+"/users", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code for POST: got %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var userPost User
	err = json.NewDecoder(resp.Body).Decode(&userPost)
	if err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	if userPost.ID == 0 || userPost.Name == "" || userPost.Email == "" {
		t.Fatalf("expected non-empty user for POST but got %+v", userPost)
	}

	req, err := http.NewRequest("PUT", server.URL+"/users", nil)
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send PUT request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status code for PUT: got %v, want %v", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
