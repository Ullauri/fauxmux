package fauxmux

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Mux struct {
	mux    *http.ServeMux
	routes sync.Map
}

// NewMux creates a new Mux instance
func NewMux() *Mux {
	return &Mux{
		mux:    http.NewServeMux(),
		routes: sync.Map{},
	}
}

// Mux returns the underlying http.ServeMux of the Mux
func (fm *Mux) Mux() *http.ServeMux {
	return fm.mux
}

// Routes returns a list of registered routes in the format "METHOD PATH"
func (fm *Mux) Routes() []string {
	paths := make([]string, 0)
	fm.routes.Range(func(path, methods any) bool {
		methods.(*sync.Map).Range(func(method, _ any) bool {
			paths = append(paths, fmt.Sprintf("%s %s", method, path))
			return true
		})
		return true
	})
	return paths
}

// RegisterEndpoint registers a new endpoint with a specific configuration for a given response type
func RegisterEndpoint[T any](fm *Mux, endpointCfg EndpointConfig) error {
	if err := endpointCfg.Validate(); err != nil {
		return err
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		latency := time.Duration(rand.Intn(int(endpointCfg.MaxLatency-endpointCfg.MinLatency))) + endpointCfg.MinLatency
		time.Sleep(latency)

		response, err := getResponseData[T](endpointCfg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
			return
		}

		switch ResponseFormat(endpointCfg.ResponseFormat) {
		case JSON:
			writeJSON(w, response)
		default:
			http.Error(w, "Invalid Response Format", http.StatusInternalServerError)
			return
		}
	})

	methodHandlers, loaded := fm.routes.LoadOrStore(endpointCfg.Path, &sync.Map{})
	methodHandlers.(*sync.Map).Store(endpointCfg.Method, handler)

	if !loaded {
		fm.mux.HandleFunc(endpointCfg.Path, func(w http.ResponseWriter, r *http.Request) {
			methodHandlers, ok := fm.routes.Load(endpointCfg.Path)
			if !ok {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}

			if methodHandler, ok := methodHandlers.(*sync.Map).Load(r.Method); ok {
				methodHandler.(http.HandlerFunc).ServeHTTP(w, r)
			} else {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	return nil
}
