# FauxMux

FauxMux is a Go library designed to simulate external HTTP services for testing purposes. It allows developers to easily set up configurable endpoints with randomized responses, latency simulation, and customizable error handling. FauxMux is ideal for testing how your applications interact with external APIs under different scenarios, such as fluctuating response times and varying payload sizes.

## Features

- Define custom HTTP endpoints with various response types.
- Simulate different response latencies for testing performance.
- Generate randomized payloads using a faker library of your choice.
- Support for lists, custom error responses, and different response formats (JSON, bytes, etc.).
- Customizable error rates to simulate failures or intermittent service issues.
- Compatible with both `http.Server` and `httptest.Server` for real and unit testing scenarios.

## Installation

To install FauxMux, use `go get github.com/ullauri/fauxmux`

## Usuage Example
```go
package main

import (
	"log"
	"net/http"
	"time"

	faker "github.com/go-faker/faker/v4"
	"github.com/ullauri/fauxmux"
)

type User struct {
	ID    int    `json:"id" fake:"{number:1,100}"`
	Name  string `json:"name" fake:"{firstname} {lastname}"`
	Email string `json:"email" fake:"{email}"`
}

func main() {
	// Setup the faker function for generating fake data
	fauxmux.Setup(fauxmux.Config{
		FakeDataFunc: func(v interface{}) error {
			return faker.FakeData(v)
		},
	})

	// Initialize the FauxMux instance
	mux := fauxmux.NewMux()

	// Register a GET endpoint that returns random User data
	err := fauxmux.RegisterEndpoint[User](mux, fauxmux.EndpointConfig{
		Method:         "GET",
		Path:           "/users",
		MinLatency:     100 * time.Millisecond,
		MaxLatency:     1000 * time.Millisecond,
		ResponseFormat: fauxmux.JSON,
	})

	if err != nil {
		log.Fatalf("failed to register endpoint: %v", err)
	}

	// Start the server
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux.Mux()))
}
```

In this example we:
- define a User struct with fields like ID, Name, and Email.
- register an HTTP GET endpoint /users which returns random User data generated by the faker library.
- simulate latency between 100ms and 1000ms to mimic real-world API behavior.

## Simulating Errors
FauxMux also supports configurable error responses. For example, you can specify error frequency and provide multiple error response options for an endpoint:
```go
err = fauxmux.RegisterEndpoint[User](mux, fauxmux.EndpointConfig{
	Method:         "GET",
	Path:           "/users/error",
	MinLatency:     100 * time.Millisecond,
	MaxLatency:     1000 * time.Millisecond,
	ResponseFormat: fauxmux.JSON,
	ErrorResponseConfig: &fauxmux.ErrorResponseConfig{
		Frequency: 0.5, // 50% chance of triggering an error
		Responses: []fauxmux.ErrorResponse{
			{
				StatusCode:     500,
				Response:       "Internal Server Error",
				ResponseFormat: fauxmux.Bytes,
			},
			{
				StatusCode:     404,
				Response:       SomeError{Message: "Not Found"},
				ResponseFormat: fauxmux.JSON,
			},
		},
	},
})
```

In this example, the /users/error endpoint will randomly trigger an error 50% of the time, returning either a 500 or 404 status with a customizable error response.
