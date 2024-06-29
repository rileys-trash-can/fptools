package main

import (
	"net/http"

	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"runtime/debug"
)

var ErrorHandlerMiddleware = mux.MiddlewareFunc(func(next http.Handler) http.Handler {
	return &errMiddleware{next}
})

type errMiddleware struct {
	next http.Handler
}

type ErrorRes struct {
	Error any `json:"error"`
}

func (e *ErrorRes) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorRes{Error: fmt.Sprint(e.Error)})
}

func (m *errMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}

		log.Printf("Error in %s request of '%s': %s", r.Method, r.URL.Path, err)
		if *OptVerbose {
			log.Print("stacktrace from panic: \n" + string(debug.Stack()))
		}

		switch w.Header().Get("Content-Type") {
		case "application/json":
			w.Header().Set("Location", "/")

			w.WriteHeader(500)
			enc := json.NewEncoder(w)
			enc.Encode(&ErrorRes{
				Error: err,
			})
			return

		default:
			w.Header().Set("Content-Type", "text/plain")
			fallthrough

		case "text/plain":
			w.WriteHeader(500)
			fmt.Fprintf(w, "There was an error handeling your request: %s\n return to / to do stuff", err)
		}

	}()

	m.next.ServeHTTP(w, r)
}
