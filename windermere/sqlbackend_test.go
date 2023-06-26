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
	"database/sql"
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/Sambruk/windermere/scimserverlite"
	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/test"
	"github.com/jmoiron/sqlx"
)

//////////////////////////////////////////
// Test data re-used in several test cases
//////////////////////////////////////////

var tenant1 = "https://tenant.com"
var tenant2 = "https://other.com"

var bajeJSON = `
{
	"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User",
	            "urn:scim:schemas:extension:sis:school:1.0:User"],
	"externalId": "75c666db-e60e-4687-bdd3-1af191fa6799",
	"userName": "baje@skola.kommunen.se",
	"displayName": "Babs Jensen",
	"name": {
		"familyName": "Jensen",
		"givenName": "Barbara"
	},
	"emails": [
        
        {
          "value": "baje@skolan.kommunen.se" 
        } 
        
    ]
}
`

var baje ss12000v1.User

var bajeNewUserName = `
{
	"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User",
	            "urn:scim:schemas:extension:sis:school:1.0:User"],
	"externalId": "75c666db-e60e-4687-bdd3-1af191fa6799",
	"userName": "baje12@skola.kommunen.se",
	"displayName": "Babs Jensen",
	"name": {
		"familyName": "Jensen",
		"givenName": "Barbara"
	},
	"emails": [
        
        {
          "value": "baje@skolan.kommunen.se" 
        } 
        
    ]
}
`

var ananJSON = `
{
	"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User",
	            "urn:scim:schemas:extension:sis:school:1.0:User"],
	"externalId": "88c0f298-8e33-4566-ace7-6e26228a9bc6",
	"userName": "anan@skola.kommunen.se",
	"displayName": "Anders Andersson",
	"name": {
		"familyName": "Andersson",
		"givenName": "Anders"
	},
	"emails": [
        
        {
          "value": "anan@skolan.kommunen.se" 
        } 
        
    ]
}
`
var anan ss12000v1.User

var liniJSON = `
{
	"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User",
	            "urn:scim:schemas:extension:sis:school:1.0:User"],
	"externalId": "aeb9dfad-c824-49e2-89d6-84cf5e33feef",
	"userName": "lini@skola.kommunen.se",
	"displayName": "Lisa Nilsson",
	"name": {
		"familyName": "Nilsson",
		"givenName": "Lisa"
	},
	"emails": [
        
        {
          "value": "lini@skolan.kommunen.se" 
        } 
        
    ],
    "urn:scim:schemas:extension:sis:school:1.0:User": {
        "enrolments": [
            
            {
                "value": "8d371858-3fbd-4af2-ae33-84225ead4a1b",
				"schoolYear": 4
            } 
        ]
    }
}
`
var lini ss12000v1.User

var grupp1JSON = `
{
	"schemas": [
	  "urn:scim:schemas:extension:sis:school:1.0:StudentGroup"
	],
	"externalId": "39074b36-e0ed-4443-a501-5148992014b9",
	"displayName": "grupp1",
	"studentGroupType": "Klass",
	"owner": {
	  "value": "8d371858-3fbd-4af2-ae33-84225ead4a1b"
	},
	"studentMemberships": [
	  
	  {
		"value": "aeb9dfad-c824-49e2-89d6-84cf5e33feef"
	  },
	  {
		"value": "2b3a480f-d0b9-4c09-bbac-70f915964b02"
	  } 
	]
}
`

var grupp1 ss12000v1.StudentGroup

var kommunenJSON = `
{
	"schemas": ["urn:scim:schemas:extension:sis:school:1.0:Organisation"],
	"externalId": "d80428c4-8788-47d7-aca7-761681fbe66a",
	"displayName": "Kommunen"
}
`

var kommunen ss12000v1.Organisation

var skolgruppenJSON = `
{
	"schemas": ["urn:scim:schemas:extension:sis:school:1.0:SchoolUnitGroup"],
	"externalId": "b7cbd8b7-96a6-425f-b14c-d4564d989d84",
	"displayName": "skolgruppen"
}
`

var skolgruppen ss12000v1.SchoolUnitGroup

var skolenhet1JSON = `
{
    "schemas": ["urn:scim:schemas:extension:sis:school:1.0:SchoolUnit"],
    "externalId": "8d371858-3fbd-4af2-ae33-84225ead4a1b",
    "displayName": "skolenhet1",
    "schoolUnitCode": "12345678",
    "schoolUnitGroup":  {
        "value": "b7cbd8b7-96a6-425f-b14c-d4564d989d84"
    },
    "organisation":  {
        "value": "d80428c4-8788-47d7-aca7-761681fbe66a"
    },
    "municipalityCode": "9999"
}
`

var skolenhet1 ss12000v1.SchoolUnit

var bajeEmpJSON = `
{
    "schemas": ["urn:scim:schemas:extension:sis:school:1.0:Employment"],
    "externalId": "163cbddb-9fd0-53df-81e4-e022c5dd5c71",
    "employedAt":  {
        "value": "8d371858-3fbd-4af2-ae33-84225ead4a1b"
    },
    "user":  {
        "value": "75c666db-e60e-4687-bdd3-1af191fa6799"
    },
    "employmentRole": "Lärare",
    "signature": "baje"
}`

var bajeEmp ss12000v1.Employment

var grupp2ActivityJSON = `
{
    "schemas": ["urn:scim:schemas:extension:sis:school:1.0:Activity"],
    "externalId": "7f9f75d8-9c01-5c1d-83df-b0d47cf1e4c9",
    "displayName": "grupp2-Activity",
    "owner": {
        "value": "8d371858-3fbd-4af2-ae33-84225ead4a1b"
    },
    "groups": [{
        "value": "645ebd9d-55b5-4e7a-900a-92b9369c8f6a"
    }],
    "teachers": [
        
        {
            "value": "db405316-e9d1-50d2-89c5-776f91ac2c98"
        },
        
        {
            "value": "163cbddb-9fd0-53df-81e4-e022c5dd5c71"
        } 
        
    ]
}
`

var grupp2Activity ss12000v1.Activity

var initOnce sync.Once

func initTestData() {
	json.Unmarshal([]byte(bajeJSON), &baje)
	json.Unmarshal([]byte(ananJSON), &anan)
	json.Unmarshal([]byte(liniJSON), &lini)
	json.Unmarshal([]byte(grupp1JSON), &grupp1)
	json.Unmarshal([]byte(kommunenJSON), &kommunen)
	json.Unmarshal([]byte(skolgruppenJSON), &skolgruppen)
	json.Unmarshal([]byte(skolenhet1JSON), &skolenhet1)
	json.Unmarshal([]byte(bajeEmpJSON), &bajeEmp)
	json.Unmarshal([]byte(grupp2ActivityJSON), &grupp2Activity)
}

type sqltestfixture struct {
	b  *SQLBackend
	db *sqlx.DB
}

func startTest(t *testing.T) *sqltestfixture {
	initOnce.Do(initTestData)
	var f sqltestfixture
	db, err := sqlx.Open("sqlite", ":memory:")
	test.Ensure(t, err)
	parser := validatingObjectParser(CreateOptionalValidator(true, true), objectParser)
	b, err := NewSQLBackend(db, parser)
	test.Ensure(t, err)
	f.b = b
	f.db = db
	return &f
}

func TestCreate(t *testing.T) {
	f := startTest(t)
	_, err := f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	_, err = f.b.Create(tenant1, "Users", bajeJSON)
	test.MustFail(t, err)
	scimError, ok := err.(scimserverlite.SCIMTypedError)
	if !ok || scimError.Type() != scimserverlite.ConflictError {
		t.Errorf("wrong error, expected conflict, got: %v", err)
	}
	_, err = f.b.Create(tenant1, "Users", ananJSON)
	test.Ensure(t, err)
	_, err = f.b.Create(tenant2, "Users", bajeJSON)
	test.Ensure(t, err)
}

func TestUpdate(t *testing.T) {
	f := startTest(t)
	_, err := f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)

	_, err = f.b.Update(tenant1, "Users", baje.GetID(), bajeNewUserName)
	test.Ensure(t, err)
	_, err = f.b.Update(tenant2, "Users", baje.GetID(), bajeJSON)
	test.MustFail(t, err)
	scimError, ok := err.(scimserverlite.SCIMTypedError)
	if !ok || scimError.Type() != scimserverlite.MissingResourceError {
		t.Errorf("wrong error, expected conflict, got: %v", err)
	}
}

func TestDelete(t *testing.T) {
	f := startTest(t)
	err := f.b.Delete(tenant1, "Users", baje.GetID())
	test.MustFail(t, err)
	_, err = f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	err = f.b.Delete(tenant1, "Users", baje.GetID())
	test.Ensure(t, err)
	_, err = f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	err = f.b.Delete(tenant2, "Users", baje.GetID())
	test.MustFail(t, err)
}

func TestClear(t *testing.T) {
	f := startTest(t)
	_, err := f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	test.Ensure(t, f.b.Clear(tenant1))
	_, err = f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	_, err = f.b.Create(tenant2, "Users", bajeJSON)
	test.Ensure(t, err)
	test.Ensure(t, f.b.Clear(tenant1))
	_, err = f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	_, err = f.b.Create(tenant2, "Users", bajeJSON)
	test.MustFail(t, err)
}

func TestGetParsedResource(t *testing.T) {
	f := startTest(t)
	_, err := f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	obj, err := f.b.GetParsedResource(tenant1, "Users", baje.GetID())
	test.Ensure(t, err)
	if obj == nil {
		t.Fatalf("Expected valid object from GetParsedResource, got nil")
	}
	user, ok := obj.(*ss12000v1.User)
	if !ok {
		t.Fatalf("Wrong type of object returned from GetParsedResource, expected User, got %T", obj)
	}
	if user.GetID() != baje.GetID() {
		t.Errorf("GetParsedResource returned user with unexpected ID: %s", user.GetID())
	}
	if user.UserName != "baje@skola.kommunen.se" {
		t.Errorf("GetParsedResource returned user with unexpected UserName: %s", user.UserName)
	}
	_, err = f.b.Update(tenant1, "Users", baje.GetID(), bajeNewUserName)
	test.Ensure(t, err)
	obj, err = f.b.GetParsedResource(tenant1, "Users", baje.GetID())
	test.Ensure(t, err)
	if obj == nil {
		t.Fatalf("Expected valid object from GetParsedResource, got nil")
	}
	user, ok = obj.(*ss12000v1.User)
	if !ok {
		t.Fatalf("Wrong type of object returned from GetParsedResource, expected User, got %T", obj)
	}
	if user.UserName != "baje12@skola.kommunen.se" {
		t.Errorf("GetParsedResource returned user with unexpected UserName: %s", user.UserName)
	}
}

func TestGetParsedResources(t *testing.T) {
	f := startTest(t)
	users, err := f.b.GetParsedResources(tenant1, "Users")
	test.Ensure(t, err)
	if len(users) != 0 {
		t.Errorf("Expected 0 users from GetParsedResources, got %d", len(users))
	}

	_, err = f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	_, err = f.b.Create(tenant1, "Users", ananJSON)
	test.Ensure(t, err)

	users, err = f.b.GetParsedResources(tenant1, "Users")
	test.Ensure(t, err)
	if len(users) != 2 {
		t.Fatalf("Expected 2 users from GetParsedResources, got %d", len(users))
	}

	user1, ok := users[baje.GetID()].(*ss12000v1.User)
	if !ok {
		t.Fatalf("Bad object type returned from GetParsedResources, expected User, got %T", users[baje.GetID()])
	}
	if user1.UserName != "baje@skola.kommunen.se" {
		t.Errorf("GetParsedResources returned user with unexpected UserName: %s", user1.UserName)
	}

	user2, ok := users[anan.GetID()].(*ss12000v1.User)
	if !ok {
		t.Fatalf("Bad object type returned from GetParsedResources, expected User, got %T", users[anan.GetID()])
	}
	if user2.UserName != "anan@skola.kommunen.se" {
		t.Errorf("GetParsedResources returned user with unexpected UserName: %s", user2.UserName)
	}
}

func TestGetResource(t *testing.T) {
	f := startTest(t)
	_, err := f.b.Create(tenant1, "Users", bajeJSON)
	test.Ensure(t, err)
	str, err := f.b.GetResource(tenant1, "Users", baje.GetID())
	test.Ensure(t, err)
	var user ss12000v1.User
	err = json.Unmarshal([]byte(str), &user)
	test.Ensure(t, err)
	if user.GetID() != baje.GetID() {
		t.Errorf("GetResource returned user with unexpected id: %s", user.GetID())
	}
}

func TestDeleteCascade(t *testing.T) {
	f := startTest(t)
	_, err := f.b.Create(tenant1, "Users", liniJSON)
	test.Ensure(t, err)
	err = f.b.Delete(tenant1, "Users", lini.GetID())
	test.Ensure(t, err)

	named, _ := f.db.PrepareNamed(`SELECT 1 FROM Emails`)
	var dest int
	err = named.Get(&dest, map[string]interface{}{})
	if err != sql.ErrNoRows {
		t.Errorf("Expected no rows in Emails table after delete")
	}
}

func TestIdentity(t *testing.T) {
	f := startTest(t)

	roundTrip := func(tenant, resourceType, json, id string, want ss12000v1.Object, create bool) {
		var err error
		if create {
			_, err = f.b.Create(tenant, resourceType, json)
		} else {
			_, err = f.b.Update(tenant, resourceType, id, json)
		}
		test.Ensure(t, err)

		obj, err := f.b.GetParsedResource(tenant, resourceType, id)
		test.Ensure(t, err)
		if !reflect.DeepEqual(want, obj) {
			t.Errorf("object of type %s wasn't the same after round-trip, expected %v\n,got %v\n", resourceType, want, obj)
		}

	}

	roundTrip(tenant1, "Users", liniJSON, lini.GetID(), &lini, true)
	roundTrip(tenant1, "StudentGroups", grupp1JSON, grupp1.GetID(), &grupp1, true)
	roundTrip(tenant1, "Organisations", kommunenJSON, kommunen.GetID(), &kommunen, true)
	roundTrip(tenant1, "SchoolUnitGroups", skolgruppenJSON, skolgruppen.GetID(), &skolgruppen, true)
	roundTrip(tenant1, "SchoolUnits", skolenhet1JSON, skolenhet1.GetID(), &skolenhet1, true)
	roundTrip(tenant1, "Employments", bajeEmpJSON, bajeEmp.GetID(), &bajeEmp, true)

	var skolenhet1Copy ss12000v1.SchoolUnit
	json.Unmarshal([]byte(skolenhet1JSON), &skolenhet1Copy)

	skolenhet1Copy.Organisation = nil
	body, _ := json.Marshal(&skolenhet1Copy)
	roundTrip(tenant1, "SchoolUnits", string(body), skolenhet1Copy.GetID(), &skolenhet1Copy, false)
	skolenhet1Copy.SchoolUnitGroup = nil
	body, _ = json.Marshal(&skolenhet1Copy)
	roundTrip(tenant1, "SchoolUnits", string(body), skolenhet1Copy.GetID(), &skolenhet1Copy, false)
	skolenhet1Copy.SchoolTypes = &[]string{"GR"}
	body, _ = json.Marshal(&skolenhet1Copy)
	roundTrip(tenant1, "SchoolUnits", string(body), skolenhet1Copy.GetID(), &skolenhet1Copy, false)

	var bajeEmpCopy ss12000v1.Employment
	json.Unmarshal([]byte(bajeEmpJSON), &bajeEmpCopy)
	bajeEmpCopy.Signature = ""
	body, _ = json.Marshal(&bajeEmpCopy)
	roundTrip(tenant1, "Employments", string(body), bajeEmpCopy.GetID(), &bajeEmpCopy, false)

	roundTrip(tenant1, "Activities", grupp2ActivityJSON, grupp2Activity.GetID(), &grupp2Activity, true)
}

func TestValidation(t *testing.T) {
	f := startTest(t)
	badUUID := `
	{
		"schemas": ["urn:scim:schemas:extension:sis:school:1.0:Organisation"],
		"externalId": "x80428c4-8788-47d7-aca7-761681fbe66a",
		"displayName": "Kommunen"
	}
	`

	_, err := f.b.Create(tenant1, "Organisations", badUUID)
	test.MustFail(t, err)
	scimError, ok := err.(scimserverlite.SCIMTypedError)
	if !ok || scimError.Type() != scimserverlite.MalformedResourceError {
		t.Errorf("wrong error, expected malformed resource, got: %v", err)
	}

	badSchoolUnitCode := `
	{
		"schemas": ["urn:scim:schemas:extension:sis:school:1.0:SchoolUnit"],
		"externalId": "8d371858-3fbd-4af2-ae33-84225ead4a1b",
		"displayName": "skolenhet1",
		"schoolUnitCode": "123",
		"schoolUnitGroup":  {
			"value": "b7cbd8b7-96a6-425f-b14c-d4564d989d84"
		},
		"organisation":  {
			"value": "d80428c4-8788-47d7-aca7-761681fbe66a"
		},
		"municipalityCode": "9999"
	}
	`

	_, err = f.b.Create(tenant1, "SchoolUnits", badSchoolUnitCode)
	test.MustFail(t, err)
	scimError, ok = err.(scimserverlite.SCIMTypedError)
	if !ok || scimError.Type() != scimserverlite.MalformedResourceError {
		t.Errorf("wrong error, expected malformed resource, got: %v", err)
	}
}
