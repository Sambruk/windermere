package main

import (
	"log"
	"net/http"
	"runtime"
	"time"
)

// This is a workaround for the fact that stack traces are lost
// is a panic occurs in an HTTP handler when http.TimeoutHandler
// is used. See:
// https://github.com/golang/go/issues/27375

// PanicReportTimeoutHandler works like http.TimeoutHandler except
// stack traces are logged when there's a panic.
func PanicReportTimeoutHandler(h http.Handler, dt time.Duration, msg string) http.Handler {
	return http.TimeoutHandler(&panicReporterHandler{handler: h}, dt, msg)
}

type panicReporterHandler struct {
	handler http.Handler
}

// Logs something through the current http.Server's logger
func (h *panicReporterHandler) logf(r *http.Request, format string, args ...interface{}) {
	s, _ := r.Context().Value(http.ServerContextKey).(*http.Server)
	if s != nil && s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

// ServeHTTP implements http.Handler for panicReporterHandler
func (h *panicReporterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if err != http.ErrAbortHandler {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				h.logf(r, "http: panic serving %v: %v\n%s", r.RemoteAddr, err, buf)
			}
			panic(http.ErrAbortHandler)
		}

	}()
	h.handler.ServeHTTP(w, r)
}
