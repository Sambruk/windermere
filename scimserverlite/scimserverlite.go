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

package scimserverlite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// A TenantGetter is a function which provides the authenticated tenant from the context
// Typically scimserverlite's http handler will sit behind a middleware which handles
// authentication and figures out which tenant the client represents. The middlware
// can then store the tenant in the context and provide a TenantGetter which retrieves it.
//
// If no authentication is used the TenantGetter should simply return "".
type TenantGetter func(c context.Context) string

// Server is a light weight SCIM server
type Server struct {
	mux       *http.ServeMux
	backend   Backend
	getTenant TenantGetter
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// NewServer will create allocate and return a new Server
func NewServer(endpoints []string, backend Backend, tenantGetter TenantGetter) *Server {
	var server Server
	server.mux = http.NewServeMux()
	server.backend = backend
	server.getTenant = tenantGetter

	for _, endpoint := range endpoints {
		server.mux.HandleFunc("/"+endpoint, func(w http.ResponseWriter, r *http.Request) { genericSCIMHandler(w, r, &server) })
		server.mux.HandleFunc("/"+endpoint+"/", func(w http.ResponseWriter, r *http.Request) { genericSCIMHandler(w, r, &server) })
	}

	return &server
}

func getResourceType(url *url.URL) (string, error) {
	path := strings.Split(url.Path, "/")
	if len(path) < 1 {
		return "", fmt.Errorf("Too few components in path")
	}
	return path[len(path)-1], nil
}

func getResourceTypeAndID(url *url.URL) (string, string, error) {
	path := strings.Split(url.Path, "/")
	if len(path) < 2 {
		return "", "", fmt.Errorf("Too few components in path")
	}
	return path[len(path)-2], path[len(path)-1], nil
}

// Writes the response to a query.
// This is a bit of a hack to get the rebuild-cache functionality to work
// in the Egil SCIM client. Only GET of all resources for a type is
// implemented, no paging, no filtering or sorting.
func writeQueryResponse(w io.Writer, resources map[string]string) error {

	type queryResponse struct {
		Schemas      []string                 `json:"schemas"`
		TotalResults int                      `json:"totalResults"`
		Resources    []map[string]interface{} `json:"Resources"`
	}

	var response queryResponse

	response.Schemas = []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"}
	response.TotalResults = len(resources)
	response.Resources = make([]map[string]interface{}, 0, len(resources))

	for id, resource := range resources {
		parsed := make(map[string]interface{})
		err := json.Unmarshal([]byte(resource), &parsed)

		if err != nil {
			return err
		}

		parsed["id"] = id
		response.Resources = append(response.Resources, parsed)
	}

	body, err := json.Marshal(&response)

	if err != nil {
		return err
	}

	_, err = w.Write(body)
	return err
}

func handleBackendError(w http.ResponseWriter, e error) {
	typedError, ok := e.(SCIMTypedError)
	status := http.StatusInternalServerError
	if ok {
		errorType := typedError.Type()

		switch errorType {
		case ConflictError:
			status = http.StatusConflict
		case MissingResourceError:
			status = http.StatusNotFound
		case MalformedResourceError:
			status = http.StatusBadRequest
		}
	}
	http.Error(w, e.Error(), status)
}

const SCIMMediaType = "application/scim+json"
const SCIMDeprecatedMediaType = "application/json"

func resourceResponse(w http.ResponseWriter, backendResource string, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", SCIMMediaType)
	w.Write([]byte(backendResource))
}

func genericSCIMHandler(w http.ResponseWriter, r *http.Request, server *Server) {

	body := ""
	tenant := server.getTenant(r.Context())

	if r.Method == "POST" || r.Method == "PUT" {
		mediaType := r.Header.Get("Content-Type")
		if mediaType != SCIMMediaType &&
			mediaType != SCIMDeprecatedMediaType {
			http.Error(w, fmt.Sprintf("Bad media type: got \"%s\" (SCIM uses %s)", mediaType, SCIMMediaType),
				http.StatusUnsupportedMediaType)
			return
		}

		if b, err := io.ReadAll(r.Body); err == nil {
			body = string(b)
		} else {
			http.Error(w, "Failed to read HTTP body", http.StatusInternalServerError)
			return
		}
	}

	if r.Method == "POST" {
		resourceType, err := getResourceType(r.URL)
		if err != nil {
			http.Error(w, "Failed to get resource type from URL", http.StatusBadRequest)
			return
		}
		backendResource, err := server.backend.Create(tenant, resourceType, body)
		if err != nil {
			handleBackendError(w, err)
			return
		}
		resourceResponse(w, backendResource, http.StatusCreated)
	} else if r.Method == "PUT" {
		resourceType, resourceID, err := getResourceTypeAndID(r.URL)
		if err != nil {
			http.Error(w, "Failed to get resource type and ID from URL", http.StatusBadRequest)
			return
		}
		backendResource, err := server.backend.Update(tenant, resourceType, resourceID, body)
		if err != nil {
			handleBackendError(w, err)
			return
		}
		resourceResponse(w, backendResource, http.StatusOK)
	} else if r.Method == "DELETE" {
		resourceType, resourceID, err := getResourceTypeAndID(r.URL)
		if err != nil {
			http.Error(w, "Failed to get resource type and ID from URL", http.StatusBadRequest)
			return
		}
		err = server.backend.Delete(tenant, resourceType, resourceID)
		if err != nil {
			handleBackendError(w, err)
			return
		}
		w.WriteHeader(204)
	} else if r.Method == "GET" {
		resourceType, err := getResourceType(r.URL)
		if err != nil {
			http.Error(w, "Failed to get resource type from URL", http.StatusBadRequest)
			return
		}
		resources, err := server.backend.GetResources(tenant, resourceType)
		if err != nil {
			handleBackendError(w, err)
			return
		}
		writeQueryResponse(w, resources)
	} else {
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

// IDGenerator is a function which takes a resource and generates an ID for the resource
type IDGenerator func(string) (string, error)

// CreateIDFromExternalID will use the externalId attribute as internal ID
func CreateIDFromExternalID(resource string) (string, error) {
	var f interface{}
	err := json.Unmarshal([]byte(resource), &f)

	if err != nil {
		return "", err
	}

	m := f.(map[string]interface{})

	externalID, ok := m["externalId"]

	if !ok {
		return "", fmt.Errorf("Missing externalId in resource")
	}

	externalIDString, ok := externalID.(string)

	if !ok {
		return "", fmt.Errorf("externalId has invalid type")
	}

	return externalIDString, nil
}
