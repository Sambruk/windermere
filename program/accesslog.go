/*
 *  This file is part of Windermere (EGIL SCIM Server).
 *
 *  Copyright (C) 2019-2021 Föreningen Sambruk
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.

 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.

 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package program

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Sambruk/windermere/scimserverlite"
)

// Our own ResponseWriter so we can get the status code in the middleware
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

// WriteHeader overrides the method from http.ResponseWriter so we can store the status code
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Middleware for writing an access log
func accessLogHandler(handler http.Handler, path string, tenantGetter scimserverlite.TenantGetter) http.Handler {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		log.Fatalf("Failed to open access log: %v", err)
	}

	logger := log.New(file, "", log.Ldate|log.Ltime|log.LUTC)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		tenant := tenantGetter(r.Context())
		method := r.Method
		url := r.RequestURI
		ww := newLoggingResponseWriter(w)
		handler.ServeHTTP(ww, r)
		duration := time.Now().Sub(start)
		logger.Printf("%s %s %s %s %d %s", ip, tenant, method, url, ww.statusCode, duration.String())
	})
}
