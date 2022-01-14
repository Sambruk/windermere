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

package windermere

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Sambruk/windermere/scimserverlite"
	"github.com/Sambruk/windermere/ss12000v1"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/renameio"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Windermere struct {
	backend     scimserverlite.Backend
	backingPath string
	server      *scimserverlite.Server
}

func (wind *Windermere) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wind.server.ServeHTTP(w, r)
}

func (wind *Windermere) Shutdown() error {
	return wind.Save()
}

func New(backingType, backingSource string, tenantGetter scimserverlite.TenantGetter) (*Windermere, error) {
	var b scimserverlite.Backend
	if backingType == "file" {
		// TODO: remove this untypedObjectParser once InMemory-backend is SS12000-aware
		untypedObjectParser := func(resourceType, resource string) (interface{}, error) {
			return objectParser(resourceType, resource)
		}
		inMemoryBackend := scimserverlite.NewInMemoryBackend(scimserverlite.CreateIDFromExternalID, untypedObjectParser)

		err := loadSCIMBackend(inMemoryBackend, backingSource)

		if err != nil {
			return nil, fmt.Errorf("failed to read SS12000 model from file: %v", err)
		}
		b = inMemoryBackend
	} else {
		db, err := sqlx.Open(backingType, backingSource)

		if err != nil {
			return nil, fmt.Errorf("failed to open connection to database: %v", err)
		}

		// Recommended by the MySQL driver documentation,
		// should perhaps be configurable?
		db.SetConnMaxLifetime(time.Minute * 3)
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)

		sqlBackend, err := NewSQLBackend(db)

		if err != nil {
			return nil, fmt.Errorf("failed to initialize SQL backend: %v", err)
		}
		b = sqlBackend
	}

	endpoints := []string{"Users", "StudentGroups", "Organisations",
		"SchoolUnits", "SchoolUnitGroups", "Employments", "Activities"}

	s := scimserverlite.NewServer(endpoints, b, tenantGetter)

	result := &Windermere{
		backend:     b,
		backingPath: backingSource,
		server:      s,
	}

	return result, nil
}

// Save makes sure the datamodel is persisted to disk
func (w *Windermere) Save() error {
	inMemory, ok := w.backend.(*scimserverlite.InMemoryBackend)
	if ok {
		err := saveSCIMBackend(inMemory, w.backingPath)

		if err != nil {
			return fmt.Errorf("failed to save SS12000 model to file: %v", err)
		}
	}
	// No need to save for other backends
	return nil
}

// Clear will remove everything from the data model
func (w *Windermere) Clear(tenant string) error {
	err := w.backend.Clear(tenant)

	if err != nil {
		return fmt.Errorf("failed to clear SS12000 model: %v", err)
	}
	return nil
}

// GetResourceTypes returns the resource types for which we have objects
func (w *Windermere) GetResourceTypes(tenant string) []string {
	// TODO: Only works for InMemoryBackend for now, to make this work for
	//       any backend it would perhaps be nicer if the Backend interface
	//       had a more generic function for both this one and CountResources
	return w.backend.(*scimserverlite.InMemoryBackend).GetResourceTypes(tenant)
}

// CountResources will return the number of resources for a given resource type
func (w *Windermere) CountResources(tenant, resourceType string) int {
	// TODO: Only works for InMemoryBackend for now, see GetResourceTypes above
	return w.backend.(*scimserverlite.InMemoryBackend).CountResources(tenant, resourceType)
}

func (w *Windermere) GetResources(tenant, resourceType string) (map[string]string, error) {
	return w.backend.GetResources(tenant, resourceType)
}

func (w *Windermere) GetResource(tenant, resourceType string, id string) (string, error) {
	return w.backend.GetResource(tenant, resourceType, id)
}

func (w *Windermere) GetParsedResources(tenant, resourceType string) (map[string]interface{}, error) {
	return w.backend.GetParsedResources(tenant, resourceType)
}

func (w *Windermere) GetParsedResource(tenant, resourceType string, id string) (interface{}, error) {
	return w.backend.GetParsedResource(tenant, resourceType, id)
}

// Loads the in-memory backend from file
func loadSCIMBackend(backend *scimserverlite.InMemoryBackend, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	serializedForm, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = backend.Load(serializedForm)
	return err
}

// Saves the in-memory backend to file
func saveSCIMBackend(backend *scimserverlite.InMemoryBackend, path string) error {
	serializedForm, err := backend.Serialize()

	if err != nil {
		return err
	}

	return renameio.WriteFile(path, serializedForm, 0600)
}

func objectParser(resourceType, resource string) (ss12000v1.Object, error) {
	var target ss12000v1.Object
	switch resourceType {
	case "Users":
		var user ss12000v1.User
		target = &user
	case "StudentGroups":
		var group ss12000v1.StudentGroup
		target = &group
	case "SchoolUnits":
		var schoolUnit ss12000v1.SchoolUnit
		target = &schoolUnit
	case "SchoolUnitGroups":
		var schoolUnitGroup ss12000v1.SchoolUnitGroup
		target = &schoolUnitGroup
	case "Organisations":
		var organisation ss12000v1.Organisation
		target = &organisation
	case "Activities":
		var activity ss12000v1.Activity
		target = &activity
	case "Employments":
		var employment ss12000v1.Employment
		target = &employment
	}

	if target != nil {
		err := json.Unmarshal([]byte(resource), target)

		if err != nil {
			return nil, err
		}
	}

	return target, nil
}
