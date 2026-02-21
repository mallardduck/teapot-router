package testutil

import (
	"net/http"
)

// NoopResponse is a simple response function that does nothing.
func NoopResponse(w http.ResponseWriter, req *http.Request) {}

// NoopResponseHandler is an http.HandlerFunc wrapper for NoopResponse.
var NoopResponseHandler = http.HandlerFunc(NoopResponse)

// OKResponse writes a 200 OK status to the response writer.
func OKResponse(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// OKResponseHandler is an http.HandlerFunc wrapper for OKResponse.
var OKResponseHandler = http.HandlerFunc(OKResponse)

// StringResponseWriterBuilder returns a response function that writes the given response string.
func StringResponseWriterBuilder(response string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(response))
	}
}
