package testutil

import (
	"net/http"
)

func NoopResponse(w http.ResponseWriter, req *http.Request) {}

var NoopResponseHandler = http.HandlerFunc(NoopResponse)

func StringResponseWriterBuilder(response string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(response))
	}
}
