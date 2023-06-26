package windermere

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
)

// This middleware takes care of a compatibility problem for Skolsynk for Google.
// The Skolsynk for Google client performs PUT towards the resource type end point
// rather than the URI for the resource it tries to update.
// For instance PUT to /StudentGroup rather than /StudentGroup/<id>
// This middleware checks if the request is a PUT, and the URL looks like "/ResourceType",
// and if so it will parse the body to figure out the id and append that to the URL.
func putCompatibilityHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		split := strings.Split(r.URL.Path, "/")
		if r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/") && len(split) == 2 {
			r2 := r.Clone(r.Context())
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				http.Error(w, "Failed to read HTTP body", http.StatusInternalServerError)
				return
			}

			type scimResource struct {
				Id         string `json:"id"`
				ExternalId string `json:"externalId"`
			}
			var parsed scimResource
			err = json.Unmarshal(body, &parsed)

			if err != nil {
				http.Error(w, "Failed to parse body (also invalid PUT to resource type)", http.StatusBadRequest)
				return
			}

			id := ""
			if parsed.Id != "" {
				id = parsed.Id
			} else if parsed.ExternalId != "" {
				id = parsed.ExternalId
			}

			if id == "" {
				http.Error(w, "Invalid PUT to resource type didn't include id or externalId in body", http.StatusBadRequest)
				return
			}

			r2.URL.Path = path.Join(r.URL.Path, id)
			r2.URL.RawPath = path.Join(r.URL.RawPath, id)
			r2.Body = io.NopCloser(bytes.NewBuffer(body))
			h.ServeHTTP(w, r2)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
