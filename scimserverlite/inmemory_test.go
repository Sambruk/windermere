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
	"reflect"
	"sort"
	"testing"
)

type testUser struct {
	Name string
	Age  int
}

type testGroup struct {
}

const UserType = "Users"
const GroupType = "Groups"
const T1 = "tenant1"
const T2 = "tenant2"
const T3 = "tentant3"
const UserA = `
{
	"name": "Barbara Jensen",
	"age": 47
}
`
const UserB = `
{
	"name": "Barbara Jensen",
	"age": 48
}
`

const GroupA = "{}"

func objectParser(resourceType, resource string) (interface{}, error) {
	var target interface{}
	switch resourceType {
	case "Users":
		var user testUser
		target = &user
	case "Groups":
		var group testGroup
		target = &group
	default:
		return nil, fmt.Errorf("unrecognized type")
	}

	err := json.Unmarshal([]byte(resource), target)

	if err != nil {
		return nil, err
	}

	return target, nil
}

func newSerialIDGenerator() IDGenerator {
	id := 0

	return func(s string) (string, error) {
		result := fmt.Sprintf("%d", id)
		id++
		return result, nil
	}
}

func Ensure(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}

func MustFail(t *testing.T, err error) {
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestCRUD(t *testing.T) {
	b := NewInMemoryBackend(newSerialIDGenerator(), objectParser)
	_, err := b.Create(T1, UserType, UserA)
	Ensure(t, err)

	resource, err := b.GetResource(T1, UserType, "0")
	Ensure(t, err)
	if resource != UserA {
		t.Errorf("GetResource returned:\n%s\n, expected:\n%s\n", resource, UserA)
	}

	_, err = b.Update(T1, UserType, "0", UserB)
	Ensure(t, err)

	resource, err = b.GetResource(T1, UserType, "0")
	Ensure(t, err)
	if resource != UserB {
		t.Errorf("GetResource returned:\n%s\n, expected:\n%s\n", resource, UserA)
	}

	obj, err := b.GetParsedResource(T1, UserType, "0")
	Ensure(t, err)

	user, ok := obj.(*testUser)
	if !ok {
		t.Errorf("GetParsedResource returned non-testUser object")
	}

	if user.Age != 48 || user.Name != "Barbara Jensen" {
		t.Errorf("GetParsedResource returned incorrect object: %v", user)
	}

	resource, err = b.GetResource(T1, GroupType, "0")
	MustFail(t, err)

	Ensure(t, b.Delete(T1, UserType, "0"))
}

func TestMultiTenancy(t *testing.T) {
	b := NewInMemoryBackend(newSerialIDGenerator(), objectParser)
	_, err := b.Create(T1, UserType, UserA)
	Ensure(t, err)
	_, err = b.Create(T2, UserType, UserB)
	Ensure(t, err)

	resource, err := b.GetResource(T1, UserType, "0")
	Ensure(t, err)
	if resource != UserA {
		t.Errorf("GetResource returned:\n%s\n, expected:\n%s\n", resource, UserA)
	}

	resource, err = b.GetResource(T2, UserType, "1")
	Ensure(t, err)
	if resource != UserB {
		t.Errorf("GetResource returned:\n%s\n, expected:\n%s\n", resource, UserB)
	}

	resource, err = b.GetResource(T1, UserType, "1")
	MustFail(t, err)

	resource, err = b.GetResource(T2, UserType, "0")
	MustFail(t, err)

	Ensure(t, b.Clear(T1))

	resource, err = b.GetResource(T1, UserType, "0")
	MustFail(t, err)

	resource, err = b.GetResource(T2, UserType, "1")
	Ensure(t, err)
	if resource != UserB {
		t.Errorf("GetResource returned:\n%s\n, expected:\n%s\n", resource, UserB)
	}

	resource, err = b.GetResource(T3, UserType, "0")
	MustFail(t, err)

	resources, err := b.GetResources(T1, UserType)
	Ensure(t, err)
	if len(resources) != 0 {
		t.Errorf("Expected no resources, got %d", len(resources))
	}
	resources, err = b.GetResources(T2, UserType)
	Ensure(t, err)
	if len(resources) != 1 {
		t.Errorf("Expected one resource, got %d", len(resources))
	}

	parsedResources, err := b.GetParsedResources(T1, UserType)
	Ensure(t, err)
	if len(parsedResources) != 0 {
		t.Errorf("Expected no parsed resources, got %d", len(parsedResources))
	}

	parsedResources, err = b.GetParsedResources(T2, UserType)
	Ensure(t, err)
	if len(parsedResources) != 1 {
		t.Errorf("Expected one parsed resource, got %d", len(parsedResources))
	}
}

func TestSerialize(t *testing.T) {
	b := NewInMemoryBackend(newSerialIDGenerator(), objectParser)
	_, err := b.Create(T1, UserType, UserA)
	Ensure(t, err)
	_, err = b.Create(T1, UserType, UserB)
	Ensure(t, err)
	_, err = b.Create(T1, GroupType, GroupA)
	Ensure(t, err)

	saved, err := b.Serialize()
	Ensure(t, err)

	b2 := NewInMemoryBackend(newSerialIDGenerator(), objectParser)
	err = b2.Load(saved)
	Ensure(t, err)

	resourceTypes := b2.GetResourceTypes(T1)
	sort.Strings(resourceTypes)
	resourceTypesExpected := []string{GroupType, UserType}
	if !reflect.DeepEqual(resourceTypes, resourceTypesExpected) {
		t.Errorf("Bad resource types after Serialize/Load, wanted %v, got %v", resourceTypesExpected, resourceTypes)
	}
	nUsers := b2.CountResources(T1, UserType)
	if nUsers != 2 {
		t.Errorf("Bad number of users, wanted 2, got %d", nUsers)
	}
	nGroups := b2.CountResources(T1, GroupType)
	if nGroups != 1 {
		t.Errorf("Bad number of groups, wanted 1, got %d", nGroups)
	}
	nFoo := b2.CountResources(T1, "Foo")
	if nFoo != 0 {
		t.Errorf("Bad number of Foo, wanted 0, got %d", nFoo)
	}
}

func TestLoadFromOld(t *testing.T) {
	saved := `
	{
		"Users": {
			"0": "{\"Name\": \"Barbara Jensen\",\"Age\": 47}",
			"1": "{\"Name\": \"John Smith\",\"Age\": 33}"
		}
	}`

	b := NewInMemoryBackend(newSerialIDGenerator(), objectParser)
	err := b.Load([]byte(saved))
	Ensure(t, err)

	obj, err := b.GetParsedResource("", UserType, "0")
	Ensure(t, err)
	var babs testUser
	json.Unmarshal([]byte(UserA), &babs)

	if !reflect.DeepEqual(obj, &babs) {
		t.Errorf("Bad user after load from old format, wanted %v, got %v", &babs, obj)
	}
}
