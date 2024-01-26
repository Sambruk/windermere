// This file contains the HTTP handler for configuring the SS12000 v2 imports

package main

import (
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/Sambruk/windermere/ss12000v2import"
	"github.com/gorilla/csrf"
)

//go:embed www/templates/ss12000v2_import_config/*
var templatesFS embed.FS

//go:embed www/css/ss12000v2_import_config
var cssFS embed.FS

// The HTTP handler for configuring the SS12000v2 import
type ss12000v2ImportConfigurationHandler struct {
	controller *ss12000v2ImportController
	templates  *template.Template
	mux        http.Handler
}

type handlerFuncWithConfigurationHandler func(w http.ResponseWriter, r *http.Request, ch *ss12000v2ImportConfigurationHandler)

func handlerWithConfigurationHandler(ch *ss12000v2ImportConfigurationHandler, h handlerFuncWithConfigurationHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r, ch)
	}
}

func listImportsHandler(w http.ResponseWriter, r *http.Request, ch *ss12000v2ImportConfigurationHandler) {
	templateData := make(map[string]interface{})
	imports, err := ch.controller.GetAllImports()
	if err != nil {
		http.Error(w, fmt.Sprintf("Couldn't get import configurations from persistence: %s", err.Error()), http.StatusInternalServerError)
	}
	templateData["imports"] = imports
	templateData[csrf.TemplateTag] = csrf.TemplateField(r)
	err = ch.templates.ExecuteTemplate(w, "imports.html", templateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteImportHandler(w http.ResponseWriter, r *http.Request, ch *ss12000v2ImportConfigurationHandler) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		tenant := r.PostForm.Get("tenant")
		err := ch.controller.DeleteImport(tenant)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete import: %s", err.Error()), http.StatusInternalServerError)
		} else {
			// TODO: should do a relative redirect instead
			http.Redirect(w, r, "ss12000v2_import_config/imports", http.StatusSeeOther)
		}

	} else {
		http.Error(w, "Incorrect method", http.StatusBadRequest)
	}
}

func importConfigFromForm(r *http.Request) (ImportConfig, error) {
	var config ImportConfig

	config.Tenant = r.PostForm.Get("tenant")
	config.APIConfiguration.URL = r.PostForm.Get("url")
	config.APIConfiguration.Authentication = ss12000v2import.AuthEduCloud
	config.APIConfiguration.ClientId = r.PostForm.Get("client-id")
	config.APIConfiguration.ClientSecret = r.PostForm.Get("client-secret")
	fullImportFrequency, err := strconv.Atoi(r.PostForm.Get("full-import-frequency"))
	if err != nil {
		return ImportConfig{}, errors.New("Failed to parse full import frequency")
	}
	config.FullImportFrequency = fullImportFrequency
	fullImportRetryWait, err := strconv.Atoi(r.PostForm.Get("full-import-retry-wait"))
	if err != nil {
		return ImportConfig{}, errors.New("Failed to parse full import retry wait")
	}
	config.FullImportRetryWait = fullImportRetryWait

	incrementalImportFrequency, err := strconv.Atoi(r.PostForm.Get("incremental-import-frequency"))
	if err != nil {
		return ImportConfig{}, errors.New("Failed to parse incremental import frequency")
	}
	config.IncrementalImportFrequency = incrementalImportFrequency
	incrementalImportRetryWait, err := strconv.Atoi(r.PostForm.Get("incremental-import-retry-wait"))
	if err != nil {
		return ImportConfig{}, errors.New("Failed to parse incremental import retry wait")
	}
	config.IncrementalImportRetryWait = incrementalImportRetryWait
	return config, nil
}

func addImportHandler(w http.ResponseWriter, r *http.Request, ch *ss12000v2ImportConfigurationHandler) {
	if r.Method == http.MethodGet {
		templateData := make(map[string]interface{})
		templateData[csrf.TemplateTag] = csrf.TemplateField(r)

		templateData["add"] = true
		templateData["tenant"] = ""
		templateData["url"] = ""
		templateData["client_id"] = ""
		templateData["client_secret"] = ""
		templateData["full_import_frequency"] = 604800
		templateData["full_import_retry_wait"] = 5 * 60
		templateData["incremental_import_frequency"] = 3600
		templateData["incremental_import_retry_wait"] = 5 * 60

		err := ch.templates.ExecuteTemplate(w, "add_edit.html", templateData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		config, err := importConfigFromForm(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		config.APIConfiguration.Authentication = ss12000v2import.AuthAPIKey
		config.APIConfiguration.APIKeyHeader = "X-API-Key"

		err = ch.controller.AddImport(config)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create import: %s", err.Error()), http.StatusInternalServerError)
		} else {
			// TODO: should do a relative redirect instead
			http.Redirect(w, r, "ss12000v2_import_config/imports", http.StatusSeeOther)
		}

	} else {
		http.Error(w, "Incorrect method", http.StatusBadRequest)
	}
}

func editImportHandler(w http.ResponseWriter, r *http.Request, ch *ss12000v2ImportConfigurationHandler) {
	if r.Method == http.MethodGet {
		tenant := r.URL.Query().Get("tenant")

		importConfig, err := ch.controller.GetImportConfig(tenant)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read import configuration: %s", err.Error()), http.StatusInternalServerError)
			return
		} else if importConfig == nil {
			http.Error(w, fmt.Sprintf("No such import: %s", tenant), http.StatusNotFound)
			return
		}

		templateData := make(map[string]interface{})
		templateData[csrf.TemplateTag] = csrf.TemplateField(r)

		templateData["add"] = false
		templateData["tenant"] = importConfig.Tenant
		templateData["url"] = importConfig.APIConfiguration.URL
		templateData["client_id"] = importConfig.APIConfiguration.ClientId
		templateData["client_secret"] = importConfig.APIConfiguration.ClientSecret
		templateData["full_import_frequency"] = importConfig.FullImportFrequency
		templateData["full_import_retry_wait"] = importConfig.FullImportRetryWait
		templateData["incremental_import_frequency"] = importConfig.IncrementalImportFrequency
		templateData["incremental_import_retry_wait"] = importConfig.IncrementalImportRetryWait

		err = ch.templates.ExecuteTemplate(w, "add_edit.html", templateData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		config, err := importConfigFromForm(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		config.APIConfiguration.Authentication = ss12000v2import.AuthAPIKey
		config.APIConfiguration.APIKeyHeader = "X-API-Key"

		err = ch.controller.AddImport(config)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to modify import: %s", err.Error()), http.StatusInternalServerError)
		} else {
			// TODO: should do a relative redirect instead
			http.Redirect(w, r, "ss12000v2_import_config/imports", http.StatusSeeOther)
		}
	} else {
		http.Error(w, "Incorrect method", http.StatusBadRequest)
	}
}

func NewSS12000v2ImportConfigurationHandler(c *ss12000v2ImportController, secret []byte) *ss12000v2ImportConfigurationHandler {
	t := template.Must(template.ParseFS(templatesFS, "www/templates/ss12000v2_import_config/*.html"))
	m := http.NewServeMux()
	secretSum := sha256.Sum256(secret) // The secret must be 32 bytes for the csrf package
	ch := &ss12000v2ImportConfigurationHandler{
		controller: c,
		templates:  t,
		mux:        csrf.Protect(secretSum[:])(m),
	}
	m.HandleFunc("/imports", handlerWithConfigurationHandler(ch, listImportsHandler))
	m.HandleFunc("/edit", handlerWithConfigurationHandler(ch, editImportHandler))
	m.HandleFunc("/add", handlerWithConfigurationHandler(ch, addImportHandler))
	m.HandleFunc("/delete", handlerWithConfigurationHandler(ch, deleteImportHandler))
	m.Handle("/css/", http.StripPrefix("/css", http.FileServer(http.FS(cssFS))))

	return ch
}

func (ch *ss12000v2ImportConfigurationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch.mux.ServeHTTP(w, r)
}
