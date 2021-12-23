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

// SCIMErrorType is a standard type of error the backend can return
type SCIMErrorType int

// Various types of errors the backend can return, which
// should result in different HTTP error codes according
// to the SCIM spec.
const (
	// ConflictError is returned when an attempt is made to create a resource that already exists
	ConflictError = iota
	// MissingResourceError is returned if the resource doesn't exist in the backend
	MissingResourceError
	// MalformedResourceError is returned if the client sent a resource that's invalid.
	// For instance missing required attributes or if an attribute has the wrong datatype
	MalformedResourceError
)

// SCIMTypedError should be used by the backend when possible
// At least for the methods that modify resources the SCIM server
// needs to know what kind of error occurred so the correct error
// code can be given to the client
type SCIMTypedError interface {
	error
	Type() SCIMErrorType
}

type scimError struct {
	errorType SCIMErrorType
	message   string
}

func (e scimError) Error() string {
	return e.message
}

func (e scimError) Type() SCIMErrorType {
	return e.errorType
}

// NewError creates a new SCIMTypedError
func NewError(t SCIMErrorType, msg string) SCIMTypedError {
	return scimError{errorType: t, message: msg}
}

// Backend is where the SCIM server stores, modifies and gets the resources
type Backend interface {
	Create(tenant, resourceType, resource string) (string, error)
	Update(tenant, resourceType, resourceID, resource string) (string, error)
	Delete(tenant, resourceType, resourceID string) error
	Clear(tenant string) error
	GetResources(tenant, resourceType string) (map[string]string, error)
	GetResource(tenant, resourceType string, id string) (string, error)
	GetParsedResources(tenant, resourceType string) (map[string]interface{}, error)
	GetParsedResource(tenant, resourceType string, id string) (interface{}, error)
}
