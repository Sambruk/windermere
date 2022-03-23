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
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// A ResourceSet contains all resources for one tenant
type ResourceSet map[string]map[string]string

// A ParsedResourceSet contains all parsed resources for one tenant
type ParsedResourceSet map[string]map[string]interface{}

// InMemoryBackend is a simple SCIM backend which stores all resources in memory
type InMemoryBackend struct {
	resources map[string]ResourceSet
	parsed    map[string]ParsedResourceSet
	idFactory IDGenerator
	parser    ObjectParser
	lock      sync.Mutex
}

const currentVersion = 1

type serialized struct {
	Version   int
	Resources map[string]ResourceSet
}

// ObjectParser is a function which parses the resource from JSON to a Go object
// It is optional to supply an ObjectParser when creating the backend, and the
// ObjectParser doesn't need to be able to parse any resource type (it can just
// return nil for types it won't parse). In those cases the "parsed" representation
// of a resource will be nil in the backend.
type ObjectParser func(resourceType, resource string) (interface{}, error)

func (backend *InMemoryBackend) initStorage() {
	backend.resources = make(map[string]ResourceSet)
	backend.parsed = make(map[string]ParsedResourceSet)
}

// NewInMemoryBackend allocates and returns a new InMemoryBackend
func NewInMemoryBackend(gen IDGenerator, parser ObjectParser) *InMemoryBackend {
	var backend InMemoryBackend
	backend.initStorage()
	backend.idFactory = gen
	backend.parser = parser

	return &backend
}

func (backend *InMemoryBackend) getResource(tenant, resourceType, resourceID string) (string, bool) {
	if _, ok := backend.resources[tenant]; !ok {
		return "", false
	}
	if _, ok := backend.resources[tenant][resourceType]; !ok {
		return "", false
	}
	r, ok := backend.resources[tenant][resourceType][resourceID]
	return r, ok
}

// Create will create a resource in the backend
func (backend *InMemoryBackend) Create(tenant, resourceType, resource string) (string, error) {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if backend.resources[tenant] == nil {
		backend.resources[tenant] = make(ResourceSet)
	}

	if backend.resources[tenant][resourceType] == nil {
		backend.resources[tenant][resourceType] = make(map[string]string)
	}

	if backend.parsed[tenant] == nil {
		backend.parsed[tenant] = make(ParsedResourceSet)
	}

	if backend.parsed[tenant][resourceType] == nil {
		backend.parsed[tenant][resourceType] = make(map[string]interface{})
	}

	resourceID, err := backend.idFactory(resource)

	if err != nil {
		return "", err
	}

	var parsed interface{}
	if backend.parser != nil {
		parsed, err = backend.parser(resourceType, resource)

		if err != nil {
			return "", NewError(MalformedResourceError, "Failed to parse resource:\n"+err.Error())
		}

	}

	backend.resources[tenant][resourceType][resourceID] = resource
	backend.parsed[tenant][resourceType][resourceID] = parsed

	return resource, nil
}

// Update will update a resource in the backend
func (backend *InMemoryBackend) Update(tenant, resourceType, resourceID, resource string) (string, error) {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	var parsed interface{}
	var err error
	if backend.parser != nil {
		parsed, err = backend.parser(resourceType, resource)

		if err != nil {
			return "", NewError(MalformedResourceError, "Failed to parse resource:\n"+err.Error())
		}

	}

	if _, ok := backend.getResource(tenant, resourceType, resourceID); !ok {
		return "", NewError(MissingResourceError, "Resource missing: "+resourceID)
	}

	backend.resources[tenant][resourceType][resourceID] = resource
	backend.parsed[tenant][resourceType][resourceID] = parsed
	return resource, nil
}

// Delete will delete a resource from the backend
func (backend *InMemoryBackend) Delete(tenant, resourceType, resourceID string) error {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if _, ok := backend.getResource(tenant, resourceType, resourceID); !ok {
		return NewError(MissingResourceError, "Resource missing: "+resourceID)
	}

	delete(backend.resources[tenant][resourceType], resourceID)
	delete(backend.parsed[tenant][resourceType], resourceID)
	return nil
}

// Clear will remove all resources for a given tenant in the backend
func (backend *InMemoryBackend) Clear(tenant string) error {
	backend.lock.Lock()
	defer backend.lock.Unlock()
	backend.resources[tenant] = make(ResourceSet)
	backend.parsed[tenant] = make(ParsedResourceSet)
	return nil
}

// Serialize returns all resources in a format which can later be read with Load()
func (backend *InMemoryBackend) Serialize() ([]byte, error) {
	backend.lock.Lock()
	resources := backend.resources
	backend.lock.Unlock()

	serializedForm := serialized{Version: currentVersion,
		Resources: resources}

	json, err := json.MarshalIndent(&serializedForm, "", "  ")

	return json, err
}

// Load reads all resources from serialized form
func (backend *InMemoryBackend) Load(serializedForm []byte) error {
	var unmarshalled serialized

	err := json.Unmarshal(serializedForm, &unmarshalled)

	if err != nil {
		return err
	}

	if unmarshalled.Version == 0 {
		var resources ResourceSet

		err := json.Unmarshal(serializedForm, &resources)

		if err != nil {
			return err
		}

		unmarshalled.Resources = make(map[string]ResourceSet)
		unmarshalled.Resources[""] = resources
	}

	parsed := make(map[string]ParsedResourceSet)

	for tenant := range unmarshalled.Resources {
		parsed[tenant] = make(ParsedResourceSet)
		for resourceType := range unmarshalled.Resources[tenant] {
			parsed[tenant][resourceType] = make(map[string]interface{})

			for id, resource := range unmarshalled.Resources[tenant][resourceType] {
				var parsedObject interface{}
				if backend.parser != nil {
					parsedObject, err = backend.parser(resourceType, resource)
				}
				parsed[tenant][resourceType][id] = parsedObject
			}
		}
	}

	backend.lock.Lock()
	backend.resources = unmarshalled.Resources
	backend.parsed = parsed
	backend.lock.Unlock()
	return nil
}

// GetResourceTypes returns the resource types for which we have objects for a given tenant
func (backend *InMemoryBackend) GetResourceTypes(tenant string) []string {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	result := []string{}

	if _, ok := backend.resources[tenant]; !ok {
		return result
	}

	for key, resources := range backend.resources[tenant] {
		if len(resources) > 0 {
			result = append(result, key)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// CountResources returns number of objects for a resource type for a given tenant
func (backend *InMemoryBackend) CountResources(tenant, resourceType string) int {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if _, ok := backend.resources[tenant]; !ok {
		return 0
	}

	resources, ok := backend.resources[tenant][resourceType]
	if !ok {
		return 0
	}
	return len(resources)
}

// GetResources returns all resources for a type
func (backend *InMemoryBackend) GetResources(tenant, resourceType string) (map[string]string, error) {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if _, ok := backend.resources[tenant]; !ok {
		return make(map[string]string), nil
	}

	resources, ok := backend.resources[tenant][resourceType]
	if !ok {
		return make(map[string]string), nil
	}

	return resources, nil
}

// GetParsedResources returns all parsed resources
// If no ObjectParser was given, or if the ObjectParser returns nil for some resource types,
// the returned map may contain nils.
func (backend *InMemoryBackend) GetParsedResources(tenant, resourceType string) (map[string]interface{}, error) {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	if _, ok := backend.parsed[tenant]; !ok {
		return make(map[string]interface{}), nil
	}

	parsed, ok := backend.parsed[tenant][resourceType]
	if !ok {
		return make(map[string]interface{}), nil
	}

	return parsed, nil
}

// GetResource returns a specific resource for a given tenant
func (backend *InMemoryBackend) GetResource(tenant, resourceType string, id string) (string, error) {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	failure := fmt.Errorf("no resource (%s) of type (%s) for tenant (%s)", id, resourceType, tenant)

	if _, ok := backend.resources[tenant]; !ok {
		return "", failure
	}

	resources, ok := backend.resources[tenant][resourceType]

	if !ok {
		return "", failure
	}

	resource, ok := resources[id]

	if !ok {
		return "", failure
	}

	return resource, nil
}

// GetParsedResource returns a specific parsed resource for a given tenant
// If no ObjectParser was given, or if the ObjectParser returns nil for some resource types,
// the returned interface{} may be nil.
func (backend *InMemoryBackend) GetParsedResource(tenant, resourceType string, id string) (interface{}, error) {
	backend.lock.Lock()
	defer backend.lock.Unlock()

	failure := fmt.Errorf("no resource (%s) of type (%s) for tenant (%s)", id, resourceType, tenant)

	if _, ok := backend.parsed[tenant]; !ok {
		return nil, failure
	}

	parsed, ok := backend.parsed[tenant][resourceType]

	if !ok {
		return nil, failure
	}

	parsedObject, ok := parsed[id]

	if !ok {
		return nil, failure
	}

	return parsedObject, nil
}
