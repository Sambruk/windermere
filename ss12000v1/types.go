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

package ss12000v1

import (
	"encoding/json"
	"fmt"
)

// DefaultSendOrder returns the default send order to use for SS12000:2018
func DefaultSendOrder() []string {
	return []string{
		"Organisation",
		"SchoolUnitGroup",
		"SchoolUnit",
		"Student",
		"Teacher",
		"Employment",
		"StudentGroup",
		"Activity",
	}
}

// Makes sure a JSON object has all required attributes,
// on the root level.
//
// required is the list of required attributes
// data is the unparsed JSON
func ensureRequired(required []string, data []byte) error {
	preParsed := make(map[string]interface{})

	if err := json.Unmarshal(data, &preParsed); err != nil {
		return err
	}

	for _, req := range required {
		if _, ok := preParsed[req]; !ok {
			return fmt.Errorf("missing required attribute %s", req)
		}
	}
	return nil
}

// Object is an SS12000:2018 object
type Object interface {
	// GetID returns the objects UUID (id/externalId)
	GetID() string
}

// Organisation represents an organisation
type Organisation struct {
	ExternalID  string `json:"externalId"`
	DisplayName string `json:"displayName"`
}

// GetID returns the objects UUID (id/externalId)
func (o *Organisation) GetID() string {
	return o.ExternalID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (o *Organisation) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"externalId", "displayName"}, data)
	if err != nil {
		return
	}
	type organisation2 Organisation
	err = json.Unmarshal(data, (*organisation2)(o))
	return
}

// SchoolUnitGroup represents a school unit group
type SchoolUnitGroup struct {
	ExternalID  string `json:"externalId"`
	DisplayName string `json:"displayName"`
}

// GetID returns the objects UUID (id/externalId)
func (sug *SchoolUnitGroup) GetID() string {
	return sug.ExternalID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (sug *SchoolUnitGroup) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"externalId", "displayName"}, data)
	if err != nil {
		return
	}
	type schoolUnitGroup2 SchoolUnitGroup
	err = json.Unmarshal(data, (*schoolUnitGroup2)(sug))
	return
}

// SchoolUnit represents a school unit
type SchoolUnit struct {
	ExternalID       string         `json:"externalId"`
	SchoolUnitCode   string         `json:"schoolUnitCode"`
	DisplayName      string         `json:"displayName"`
	Organisation     *SCIMReference `json:"organisation"`
	SchoolUnitGroup  *SCIMReference `json:"schoolUnitGroup"`
	SchoolTypes      *[]string      `json:"schoolTypes"`
	MunicipalityCode *string        `json:"municipalityCode"`
}

// GetID returns the objects UUID (id/externalId)
func (su *SchoolUnit) GetID() string {
	return su.ExternalID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (su *SchoolUnit) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"externalId", "displayName", "schoolUnitCode"}, data)
	if err != nil {
		return
	}
	type schoolUnit2 SchoolUnit
	err = json.Unmarshal(data, (*schoolUnit2)(su))
	return
}

// ActivityJSON is used to implement UnmarshalJSON for Activity
type ActivityJSON struct {
	ExternalID     string          `json:"externalId"`
	DisplayName    string          `json:"displayName"`
	Owner          SCIMReference   `json:"owner"`
	Group          *SCIMReference  `json:"group"`  // Incorrect according to spec, but used traditionally by the EGIL client
	Groups         []SCIMReference `json:"groups"` // According to spec
	Teachers       []SCIMReference `json:"teachers"`
	ParentActivity []SCIMReference `json:"parentActivity"`
}

// Activity represents an activity
type Activity struct {
	ExternalID     string          `json:"externalId"`
	DisplayName    string          `json:"displayName"`
	Owner          SCIMReference   `json:"owner"` // The school unit
	Groups         []SCIMReference `json:"groups"`
	Teachers       []SCIMReference `json:"teachers"`
	ParentActivity []SCIMReference `json:"parentActivity"`
}

// GetID returns the objects UUID (id/externalId)
func (a *Activity) GetID() string {
	return a.ExternalID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (a *Activity) UnmarshalJSON(data []byte) error {
	err := ensureRequired([]string{"externalId", "displayName", "owner"}, data)
	if err != nil {
		return err
	}

	var ajson ActivityJSON
	if err := json.Unmarshal(data, &ajson); err != nil {
		return err
	}

	a.ExternalID = ajson.ExternalID
	a.DisplayName = ajson.DisplayName
	a.Owner = ajson.Owner

	if ajson.Group != nil {
		a.Groups = []SCIMReference{*ajson.Group}
	} else {
		a.Groups = ajson.Groups
	}
	a.Teachers = ajson.Teachers
	a.ParentActivity = ajson.ParentActivity

	return nil
}

// SCIMReference is a reference to another SCIM resource
type SCIMReference struct {
	Value string `json:"value"`
	Ref   string `json:"$ref"`
}

// StudentGroup represents a student group
type StudentGroup struct {
	ID                 string          `json:"externalId"`         // ID is the UUID for the student group
	DisplayName        string          `json:"displayName"`        // DisplayName is the human readable name
	Owner              SCIMReference   `json:"owner"`              // The school unit
	Type               *string         `json:"studentGroupType"`   // Type is the type of group (klass, undervisning...)
	StudentMemberships []SCIMReference `json:"studentMemberships"` // StudentMemberships is a list of students in the group
	SchoolType         *string         `json:"schoolType"`         // SchoolType is the type of education ("skolform", GR, GY etc.)
}

// GetID returns the objects UUID (id/externalId)
func (sg *StudentGroup) GetID() string {
	return sg.ID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (sg *StudentGroup) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"externalId", "displayName", "owner"}, data)
	if err != nil {
		return
	}
	type studentGroup2 StudentGroup
	err = json.Unmarshal(data, (*studentGroup2)(sg))
	return
}

// SCIMName is a persons name as defined in SCIM
type SCIMName struct {
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (sn *SCIMName) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"familyName", "givenName"}, data)
	if err != nil {
		return
	}
	type scimName2 SCIMName
	err = json.Unmarshal(data, (*scimName2)(sn))
	return
}

// SCIMEmail is an email address as defined in SCIM
type SCIMEmail struct {
	Value string `json:"value,omitempty"` // Value is the actual email address
	Type  string `json:"type,omitempty"`  // Type is the type of email (work, home, etc.)
}

// Enrolment is an enrolment as defined in SS12000:2018
type Enrolment struct {
	Value       string  `json:"value"`
	Ref         string  `json:"$ref"`
	SchoolYear  *int    `json:"schoolYear,omitempty"`
	SchoolType  *string `json:"schoolType,omitempty"`
	ProgramCode *string `json:"programCode,omitempty"`
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (e *Enrolment) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"value"}, data)
	if err != nil {
		return
	}
	type enrolment2 Enrolment
	err = json.Unmarshal(data, (*enrolment2)(e))
	return
}

// UserRelation is a user relation as defined in SS12000:2018
type UserRelation struct {
	Value        string  `json:"value"`
	Ref          string  `json:"$ref"`
	RelationType string  `json:"relationType"`
	DisplayName  *string `json:"displayName,omitempty"`
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (ur *UserRelation) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"value", "relationType"}, data)
	if err != nil {
		return
	}
	type userRelation2 UserRelation
	err = json.Unmarshal(data, (*userRelation2)(ur))
	return
}

// UserExtension is SS12000:2018's extension to the SCIM user object
type UserExtension struct {
	Enrolments      []Enrolment    `json:"enrolments,omitempty"`
	CivicNo         *string        `json:"civicNo,omitempty"`
	SecurityMarking *bool          `json:"securityMarking,omitempty"`
	UserRelations   []UserRelation `json:"userRelations,omitempty"`
}

// ExternalIdentifier is taken from SS12000:2020 in order to support
// import from a SS12000:2020 source.
type ExternalIdentifier struct {
	Value          string `json:"value"`
	Context        string `json:"context"`
	GloballyUnique bool   `json:"globallyUnique"`
}

// EgilUserExtension is a non-standard extension, currently only containing
// external identifiers.
type EgilUserExtension struct {
	ExternalIdentifiers []ExternalIdentifier `json:"externalIdentifiers,omitempty"`
}

// User is an SS12000:2018 user
type User struct {
	ID            string             `json:"externalId"`                                         // ID is the UUID for the student group
	UserName      string             `json:"userName"`                                           // UserName is the user's EPPN
	Name          SCIMName           `json:"name"`                                               // Name is the user's real name (given/family name etc.)
	DisplayName   string             `json:"displayName"`                                        // DisplayName is what to show (required in EGIL, not in SS12000:2018 it seems)
	Emails        []SCIMEmail        `json:"emails"`                                             // Emails is the user's email addresses
	Extension     UserExtension      `json:"urn:scim:schemas:extension:sis:school:1.0:User"`     // Extension is the SS12000:2018 SCIM extension
	EgilExtension *EgilUserExtension `json:"urn:scim:schemas:extension:egil:1.0:User,omitempty"` // Non-standard extension for external identifiers
}

// GetID returns the objects UUID (id/externalId)
func (u *User) GetID() string {
	return u.ID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (u *User) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"externalId", "userName", "name", "displayName"}, data)
	if err != nil {
		return
	}
	type user2 User
	err = json.Unmarshal(data, (*user2)(u))
	return
}

// IsEnrolledAt checks if a user is enrolled at a school unit with a given school unit code
func (u *User) IsEnrolledAt(schoolUnitCode string) bool {
	for _, e := range u.Extension.Enrolments {
		if e.Value == schoolUnitCode {
			return true
		}
	}
	return false
}

// Employment is an SS12000:2018 employment object
type Employment struct {
	ID             string        `json:"externalId"`     // ID is the UUID for the student group
	EmployedAt     SCIMReference `json:"employedAt"`     // EmployedAt is where the person is employed
	User           SCIMReference `json:"user"`           // User is the employed user
	EmploymentRole string        `json:"employmentRole"` // EmploymentRole is the type of employment
	Signature      string        `json:"signature"`      // Teacher signature
}

// GetID returns the objects UUID (id/externalId)
func (e *Employment) GetID() string {
	return e.ID
}

// UnmarshalJSON implements the interface for custom unmarshalling
func (e *Employment) UnmarshalJSON(data []byte) (err error) {
	err = ensureRequired([]string{"externalId", "employedAt", "user", "employmentRole"}, data)
	if err != nil {
		return
	}
	type employment2 Employment
	err = json.Unmarshal(data, (*employment2)(e))
	return
}
