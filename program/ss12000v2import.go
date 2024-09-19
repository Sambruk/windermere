/*
 *  This file is part of Windermere (EGIL SCIM Server).
 *
 *  Copyright (C) 2019-2024 Föreningen Sambruk
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
	"fmt"
	"log"
	"time"

	"github.com/Sambruk/windermere/ss12000v2import"
)

// This is what we store about an import in the configuration.
// It is similar to, but different from and with a different
// purpose than the RunnerConfig in the ss12000v2import package.
// The latter is not meant to be be stored to disk or mirror what
// the user configures, but rather provide everything the
// ImportRunner needs in a way so that the ImportRunner can do its
// job with minimal coupling to the rest of the import sub-system.
type ImportConfig struct {
	Tenant                     string
	APIConfiguration           ss12000v2import.APIConfiguration
	FullImportFrequency        int // seconds
	FullImportRetryWait        int // seconds
	IncrementalImportFrequency int // seconds
	IncrementalImportRetryWait int // seconds
}

// The Controller binds together the different components of the
// import sub-system and provides an interface to main and the
// HTTP configuration handler so that the individual components need
// not know about each other.
//
// The major components involved in the import sub-system are:
//
// * The import manager
//       Responsible for managing the import runners, which actually
//       carries out the imports.
// * The ss12000v1 backend where the import puts the data
//       In other words the interface to the SCIM server's storage.
// * The persistence layer
//       This is where the configurations and history about imports
//       are stored.
// * The HTTP configuration handler
//       The configuration handler uses the Controller to get, create,
//       update and remove imports.
//
// The Controller is a passive object, it doesn't have its own goroutine
// and doesn't need to be shutdown at the end of the process.
type ss12000v2ImportController struct {
	manager     *ss12000v2import.ImportManager
	persistence *ss12000v2ImportPersistence
	backend     ss12000v2import.SS12000v1Backend
}

// Creates a new ss12000v2ImportController
func NewSS12000v2ImportController(p *ss12000v2ImportPersistence, m *ss12000v2import.ImportManager, b ss12000v2import.SS12000v1Backend) *ss12000v2ImportController {
	return &ss12000v2ImportController{
		manager:     m,
		persistence: p,
		backend:     b,
	}
}

// From an ImportConfig, create a ss12000v2import.RunnerConfig
// This involves creating an OpenAPI client for SS12000 with the chosen configuration and
// providing the SS12000v1Backend as well as the ImportHistory interface so that the
// import runner can store its data and history.
func (c *ss12000v2ImportController) createRunnerConfig(config ImportConfig) (ss12000v2import.RunnerConfig, error) {
	client, err := ss12000v2import.NewClient(config.APIConfiguration)
	if err != nil {
		return ss12000v2import.RunnerConfig{}, fmt.Errorf("failed to create ss12000v2 client: %s", err.Error())
	}

	runnerConfig := ss12000v2import.RunnerConfig{
		Tenant:                     config.Tenant,
		Backend:                    c.backend,
		Client:                     client,
		History:                    c.persistence.GetHistory(config.Tenant),
		FullImportFrequency:        time.Duration(config.FullImportFrequency) * time.Second,
		FullImportRetryWait:        time.Duration(config.FullImportRetryWait) * time.Second,
		IncrementalImportFrequency: time.Duration(config.IncrementalImportFrequency) * time.Second,
		IncrementalImportRetryWait: time.Duration(config.IncrementalImportRetryWait) * time.Second,
	}
	return runnerConfig, nil
}

// This is intended to be called at start up to start all configured
// SS1200v2 imports.
func (c *ss12000v2ImportController) StartAll() {
	tenants, err := c.persistence.GetAllImports()
	if err != nil {
		log.Printf("Failed to get import configurations from persistence! (%s)", err.Error())
		return
	}
	for _, tenant := range tenants {
		config, err := c.persistence.GetImportConfig(tenant)
		if err != nil {
			log.Printf("Failed to read import configuration for %s: %s", tenant, err.Error())
			continue
		} else if config == nil {
			// Shouldn't happen since we're just iterating over the imports which
			// we just got from persistence
			log.Printf("Failed to find import configuration for %s", tenant)
			continue
		}
		runnerConfig, err := c.createRunnerConfig(*config)
		if err != nil {
			log.Printf("Failed to start import for %s: %s", config.Tenant, err.Error())
		} else {
			c.manager.AddRunner(runnerConfig)
		}
	}
}

// Adds an import (or replaces an existing one).
// This involves both storing the import configuration to persistence as well
// as actually creating and starting the import runner (unless it is configured
// as paused).
// If there already was a running import for this tenant it will be stopped
// and a new runner will be created and started with the new settings.
func (c *ss12000v2ImportController) AddImport(config ImportConfig) error {
	err := c.persistence.AddImport(config)
	if err != nil {
		return fmt.Errorf("import couldn't be created: %s", err.Error())
	}
	runnerConfig, err := c.createRunnerConfig(config)
	if err != nil {
		return fmt.Errorf("import was created but couldn't start: %s", err.Error())
	}
	c.manager.AddRunner(runnerConfig)
	return nil
}

// Deletes an import. Blocks until the current runner has stopped.
func (c *ss12000v2ImportController) DeleteImport(tenant string) error {
	c.manager.DeleteRunner(tenant)
	err := c.persistence.DeleteImport(tenant)
	if err != nil {
		return fmt.Errorf("failed to remove import from configuration: %s", err.Error())
	}
	return nil
}

// Gets a list of tenants for the configured imports.
func (c *ss12000v2ImportController) GetAllImports() ([]string, error) {
	return c.persistence.GetAllImports()
}

// Gets the import config for a specific tenant.
func (c *ss12000v2ImportController) GetImportConfig(tenant string) (*ImportConfig, error) {
	return c.persistence.GetImportConfig(tenant)
}
