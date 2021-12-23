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
	"fmt"

	"github.com/Sambruk/windermere/scimserverlite"
	scim "github.com/Sambruk/windermere/scimserverlite"
	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/jmoiron/sqlx"
)

// SQLBackend implements scimserverlite.Backend for SQL databases
type SQLBackend struct {
	db *sqlx.DB
}

// NewSQLBackend creates a new SQLBackend
func NewSQLBackend(d *sqlx.DB) (backend *SQLBackend, err error) {
	backend = &SQLBackend{db: d}
	err = backend.initSchema()
	if err != nil {
		return nil, err
	}
	return
}

func getDBVersion(db *sqlx.DB) int {
	type version struct {
		Version int `db:"version"`
	}
	var v version
	err := db.Get(&v, "SELECT version FROM windermere_meta")
	if err != nil {
		return 0
	}
	return v.Version
}

// A safeString is a string that we can trust when creating SQL queries
// Strings gotten from the SCIM client should not be considered safe
// before sanitation.
type safeString string

func mainTable(resourceType string) (safeString, error) {
	var tableToDeleteFrom = map[string]safeString{
		"Users":            "Users",
		"StudentGroups":    "StudentGroups",
		"Organisations":    "Organisations",
		"SchoolUnitGroups": "SchoolUnitGroups",
		"SchoolUnits":      "SchoolUnits",
		"Employments":      "Employments",
		"Activities":       "Activities",
	}
	table, ok := tableToDeleteFrom[resourceType]
	if !ok {
		return "", fmt.Errorf("unrecognized resource type %s", resourceType)
	}
	return table, nil
}

// Tables to go through when we call Clear for a tenant
// Basically this is the list of tables that contain
// provisioned data (minus tables automatically cleared
// by cascade), but not for instance the meta table.
var tablesForClearTenant = []safeString{
	"Users",
	"StudentGroups",
	"Organisations",
	"SchoolUnitGroups",
	"SchoolUnits",
	"Employments",
	"Activities",
}

var migrations = [...]string{
	`	
	CREATE TABLE windermere_meta (
		version INT NOT NULL
	);

	INSERT INTO windermere_meta (version) VALUES (1);
	
	CREATE TABLE Users (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		userName TEXT NOT NULL,
		familyName TEXT NOT NULL,
		givenName TEXT NOT NULL,
		displayName TEXT NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE Emails (
		tenant TEXT NOT NULL,
		userId TEXT NOT NULL,
		value TEXT NOT NULL,
		type TEXT NULL,
		FOREIGN KEY (tenant, userId) REFERENCES Users(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX EmailsIdx ON Emails (tenant, userId);

	CREATE TABLE Enrolments (
		tenant TEXT NOT NULL,
		userId TEXT NOT NULL,
		value TEXT NOT NULL,
		schoolYear TINYINT NULL,
		FOREIGN KEY (tenant, userId) REFERENCES Users(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX EnrolmentsIdx ON Enrolments (tenant, userId);

	CREATE TABLE StudentGroups (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		displayName TEXT NOT NULL,
		owner TEXT NOT NULL,
		studentGroupType TEXT NULL,
		PRIMARY KEY (tenant, id)				
	);

	CREATE TABLE StudentMemberships (
		tenant TEXT NOT NULL,
		groupId TEXT NOT NULL,
		userId TEXT NOT NULL,
		FOREIGN KEY (tenant, groupId) REFERENCES StudentGroups(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX StudentMembershipsIdx ON StudentMemberships (tenant, groupId);

	CREATE TABLE Organisations (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		displayName TEXT NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE SchoolUnitGroups (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		displayName TEXT NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE SchoolUnits (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		displayName TEXT NOT NULL,
		schoolUnitCode TEXT NOT NULL,
		organisation TEXT NULL,
		schoolUnitGroup TEXT NULL,
		municipalityCode TEXT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE SchoolTypes (
		tenant TEXT NOT NULL,
		schoolUnitId TEXT NOT NULL,
		schoolType TEXT NOT NULL,
		FOREIGN KEY (tenant, schoolUnitId) REFERENCES SchoolUnits(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX SchoolTypesIdx ON SchoolTypes (tenant, schoolUnitId);

	CREATE TABLE Employments (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		employedAt TEXT NOT NULL,
		user TEXT NOT NULL,
		employmentRole TEXT NOT NULL,
		signature TEXT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE Activities (
		tenant TEXT NOT NULL,
		id TEXT NOT NULL,
		displayName TEXT NOT NULL,
		owner TEXT NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE ActivityTeachers (
		tenant TEXT NOT NULL,
		activityId TEXT NOT NULL,
		employmentId TEXT NOT NULL,
		FOREIGN KEY (tenant, activityId) REFERENCES Activities(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX ActivityTeachersIdx ON ActivityTeachers (tenant, activityId);

	CREATE TABLE ActivityGroups (
		tenant TEXT NOT NULL,
		activityId TEXT NOT NULL,
		groupId TEXT NOT NULL,
		FOREIGN KEY (tenant, activityId) REFERENCES Activities(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX ActivityGroupsIdx ON ActivityGroups (tenant, activityId);
	`,
}

func currentSchemaVersion() int {
	return len(migrations)
}

func getSchema(version int) string {
	return migrations[version-1]
}

func driverSpecificInit(db *sqlx.DB) error {
	if db.DriverName() == "sqlite" {
		_, err := db.Exec(`PRAGMA foreign_keys = ON;`)
		return err
	}
	return nil
}

func (backend *SQLBackend) initSchema() error {
	// Ensure we have a working connection since any error in
	// getDBVersion is interpreted as an uninitialized database.
	if err := backend.db.Ping(); err != nil {
		return err
	}
	if err := driverSpecificInit(backend.db); err != nil {
		return err
	}
	version := getDBVersion(backend.db)

	if version > currentSchemaVersion() {
		return fmt.Errorf("database schema is newer than this version of Windermere. Please perform a database schema downgrade if you wish to continue with this version of Windermere.")
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()
	// loop over all migrations in order and apply those with higher
	// version than current
	for i := version + 1; i <= currentSchemaVersion(); i++ {
		_, err = tx.Exec(getSchema(i))
		if err != nil {
			return err
		}
	}

	// Set the current schema version
	tx.NamedExec(`UPDATE windermere_meta SET version = :version`, map[string]interface{}{"version": currentSchemaVersion()})

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (backend *SQLBackend) objectCreator(tx *sqlx.Tx, tenant string, obj interface{}) (id string, err error) {
	switch v := obj.(type) {
	case *ss12000v1.User:
		return backend.userCreator(tx, tenant, v)
	case *ss12000v1.StudentGroup:
		return backend.studentGroupCreator(tx, tenant, v)
	case *ss12000v1.Organisation:
		return backend.organisationCreator(tx, tenant, v)
	case *ss12000v1.SchoolUnitGroup:
		return backend.schoolUnitGroupCreator(tx, tenant, v)
	case *ss12000v1.SchoolUnit:
		return backend.schoolUnitCreator(tx, tenant, v)
	case *ss12000v1.Employment:
		return backend.employmentCreator(tx, tenant, v)
	case *ss12000v1.Activity:
		return backend.activityCreator(tx, tenant, v)
	default:
		return "", fmt.Errorf("failed to create object of unknown type: %T", obj)
	}
}

func (backend *SQLBackend) objectMutator(tx *sqlx.Tx, tenant string, obj interface{}) (err error) {
	switch v := obj.(type) {
	case *ss12000v1.User:
		return backend.userMutator(tx, tenant, v)
	case *ss12000v1.StudentGroup:
		return backend.studentGroupMutator(tx, tenant, v)
	case *ss12000v1.Organisation:
		return backend.organisationMutator(tx, tenant, v)
	case *ss12000v1.SchoolUnitGroup:
		return backend.schoolUnitGroupMutator(tx, tenant, v)
	case *ss12000v1.SchoolUnit:
		return backend.schoolUnitMutator(tx, tenant, v)
	case *ss12000v1.Employment:
		return backend.employmentMutator(tx, tenant, v)
	case *ss12000v1.Activity:
		return backend.activityMutator(tx, tenant, v)
	default:
		return fmt.Errorf("failed to update object of unknown type: %T", obj)
	}
}

func (backend *SQLBackend) objectReaderAll(tx *sqlx.Tx, resourceType, tenant string) ([]ss12000v1.Object, error) {
	switch resourceType {
	case "Users":
		return backend.userReaderAll(tx, tenant)
	case "StudentGroups":
		return backend.studentGroupReaderAll(tx, tenant)
	case "Organisations":
		return backend.organisationReaderAll(tx, tenant)
	case "SchoolUnitGroups":
		return backend.schoolUnitGroupReaderAll(tx, tenant)
	case "SchoolUnits":
		return backend.schoolUnitReaderAll(tx, tenant)
	case "Employments":
		return backend.employmentReaderAll(tx, tenant)
	case "Activities":
		return backend.activityReaderAll(tx, tenant)
	default:
		return nil, fmt.Errorf("failed to read unknown type: %s", resourceType)
	}

}

func (backend *SQLBackend) objectReaderOne(tx *sqlx.Tx, resourceType, tenant, id string) (ss12000v1.Object, error) {
	switch resourceType {
	case "Users":
		return backend.userReaderOne(tx, tenant, id)
	case "StudentGroups":
		return backend.studentGroupReaderOne(tx, tenant, id)
	case "Organisations":
		return backend.organisationReaderOne(tx, tenant, id)
	case "SchoolUnitGroups":
		return backend.schoolUnitGroupReaderOne(tx, tenant, id)
	case "SchoolUnits":
		return backend.schoolUnitReaderOne(tx, tenant, id)
	case "Employments":
		return backend.employmentReaderOne(tx, tenant, id)
	case "Activities":
		return backend.activityReaderOne(tx, tenant, id)
	default:
		return nil, fmt.Errorf("failed to read unknown type: %s", resourceType)
	}
}

func (backend *SQLBackend) Create(tenant, resourceType, resource string) (string, error) {
	table, err := mainTable(resourceType)

	if err != nil {
		return "", err
	}

	obj, err := objectParser(resourceType, resource)

	if err != nil {
		return "", scim.NewError(scim.MalformedResourceError, "Failed to parse resource:\n"+err.Error())
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	err = ensureDoesntHaveRecord(tx, table, tenant, obj.GetID())
	if err != nil {
		return "", err
	}

	_, err = backend.objectCreator(tx, tenant, obj)

	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	// TODO: Should perhaps read back the object from the database instead,
	//       on the other hand that means another transaction...
	body, err := json.Marshal(obj)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (backend *SQLBackend) Update(tenant, resourceType, resourceID, resource string) (string, error) {
	table, err := mainTable(resourceType)

	if err != nil {
		return "", err
	}

	obj, err := objectParser(resourceType, resource)

	if err != nil {
		return "", scim.NewError(scim.MalformedResourceError, "Failed to parse resource:\n"+err.Error())
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	err = ensureHasRecord(tx, table, tenant, resourceID)
	if err != nil {
		return "", err
	}

	err = backend.objectMutator(tx, tenant, obj)

	if err != nil {
		return "", err
	}

	err = tx.Commit()

	if err != nil {
		return "", err
	}

	// TODO: Should perhaps read back the object from the database instead,
	//       on the other hand that means another transaction...
	body, err := json.Marshal(obj)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ensureHasRecord(tx *sqlx.Tx, table safeString, tenant, resourceID string) error {
	named, err := tx.PrepareNamed(`SELECT 1 FROM ` + string(table) + ` WHERE tenant = :tenant AND id = :id`)

	if err != nil {
		return err
	}

	var dest int
	err = named.Get(&dest, map[string]interface{}{
		"tenant": tenant,
		"id":     resourceID,
	})
	if err == sql.ErrNoRows {
		return scim.NewError(scim.MissingResourceError, fmt.Sprintf("couldn't find object %s", resourceID))
	} else {
		return err
	}
}

func ensureDoesntHaveRecord(tx *sqlx.Tx, table safeString, tenant, resourceID string) error {
	err := ensureHasRecord(tx, table, tenant, resourceID)
	if err == nil {
		return scim.NewError(scim.ConflictError, fmt.Sprintf("object %s already exists", resourceID))
	}
	scimError, ok := err.(scimserverlite.SCIMTypedError)
	if !ok || scimError.Type() != scimserverlite.MissingResourceError {
		return err
	}
	return nil
}

func (backend *SQLBackend) Delete(tenant, resourceType, resourceID string) error {
	table, err := mainTable(resourceType)

	if err != nil {
		return err
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	err = ensureHasRecord(tx, table, tenant, resourceID)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM `+string(table)+` WHERE tenant = :tenant AND id = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     resourceID,
		})

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (backend *SQLBackend) Clear(tenant string) error {
	tx, err := backend.db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	for _, table := range tablesForClearTenant {
		_, err = tx.NamedExec(`DELETE FROM `+string(table)+` WHERE tenant = :tenant`,
			map[string]interface{}{
				"tenant": tenant,
			})
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (backend *SQLBackend) GetResources(tenant, resourceType string) (map[string]string, error) {
	objs, err := backend.GetParsedResources(tenant, resourceType)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for key := range objs {
		bytes, err := json.Marshal(objs[key])
		if err != nil {
			return nil, err
		}
		result[key] = string(bytes)
	}
	return result, nil
}

func (backend *SQLBackend) GetResource(tenant, resourceType string, id string) (string, error) {
	obj, err := backend.GetParsedResource(tenant, resourceType, id)
	if err != nil {
		return "", err
	}
	bytes, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (backend *SQLBackend) GetParsedResources(tenant, resourceType string) (map[string]interface{}, error) {
	tx, err := backend.db.Beginx()

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	objs, err := backend.objectReaderAll(tx, resourceType, tenant)

	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i := range objs {
		id := objs[i].GetID()
		result[id] = objs[i]
	}
	return result, nil
}

func (backend *SQLBackend) GetParsedResource(tenant, resourceType string, id string) (interface{}, error) {
	table, err := mainTable(resourceType)

	if err != nil {
		return nil, err
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	err = ensureHasRecord(tx, table, tenant, id)
	if err != nil {
		return nil, err
	}

	obj, err := backend.objectReaderOne(tx, resourceType, tenant, id)

	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return obj, nil
}
