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
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/Sambruk/windermere/scimserverlite"
	scim "github.com/Sambruk/windermere/scimserverlite"
	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/util"
	"github.com/jmoiron/sqlx"
)

type ObjectParser func(resourceType, resource string) (ss12000v1.Object, error)

// SQLBackend implements scimserverlite.Backend for SQL databases
type SQLBackend struct {
	db           *sqlx.DB
	objectParser ObjectParser
}

// NewSQLBackend creates a new SQLBackend
func NewSQLBackend(d *sqlx.DB, op ObjectParser) (backend *SQLBackend, err error) {
	backend = &SQLBackend{db: d, objectParser: op}
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
	// v1, initial schema version
	`	
	CREATE TABLE windermere_meta (
		version INT NOT NULL
	);

	INSERT INTO windermere_meta (version) VALUES (1);
	
	CREATE TABLE Users (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		userName {{NTEXT}} NOT NULL,
		familyName {{NTEXT}} NOT NULL,
		givenName {{NTEXT}} NOT NULL,
		displayName {{NTEXT}} NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE Emails (
		tenant {{NVARCHAR}}(255) NOT NULL,
		userId VARCHAR(36) NOT NULL,
		value {{NTEXT}} NOT NULL,
		type {{NTEXT}} NULL,
		FOREIGN KEY (tenant, userId) REFERENCES Users(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX EmailsIdx ON Emails (tenant, userId);

	CREATE TABLE Enrolments (
		tenant {{NVARCHAR}}(255) NOT NULL,
		userId VARCHAR(36) NOT NULL,
		value VARCHAR(36) NOT NULL,
		schoolYear TINYINT NULL,
		FOREIGN KEY (tenant, userId) REFERENCES Users(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX EnrolmentsIdx ON Enrolments (tenant, userId);

	CREATE TABLE StudentGroups (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		displayName {{NTEXT}} NOT NULL,
		owner VARCHAR(36) NOT NULL,
		studentGroupType {{NTEXT}} NULL,
		PRIMARY KEY (tenant, id)				
	);

	CREATE TABLE StudentMemberships (
		tenant {{NVARCHAR}}(255) NOT NULL,
		groupId VARCHAR(36) NOT NULL,
		userId VARCHAR(36) NOT NULL,
		FOREIGN KEY (tenant, groupId) REFERENCES StudentGroups(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX StudentMembershipsIdx ON StudentMemberships (tenant, groupId);

	CREATE TABLE Organisations (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		displayName {{NTEXT}} NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE SchoolUnitGroups (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		displayName {{NTEXT}} NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE SchoolUnits (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		displayName {{NTEXT}} NOT NULL,
		schoolUnitCode {{NTEXT}} NOT NULL,
		organisation VARCHAR(36) NULL,
		schoolUnitGroup VARCHAR(36) NULL,
		municipalityCode {{NTEXT}} NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE SchoolTypes (
		tenant {{NVARCHAR}}(255) NOT NULL,
		schoolUnitId VARCHAR(36) NOT NULL,
		schoolType {{NTEXT}} NOT NULL,
		FOREIGN KEY (tenant, schoolUnitId) REFERENCES SchoolUnits(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX SchoolTypesIdx ON SchoolTypes (tenant, schoolUnitId);

	CREATE TABLE Employments (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		employedAt VARCHAR(36) NOT NULL,
		userId VARCHAR(36) NOT NULL,
		employmentRole {{NTEXT}} NOT NULL,
		signature {{NTEXT}} NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE Activities (
		tenant {{NVARCHAR}}(255) NOT NULL,
		id VARCHAR(36) NOT NULL,
		displayName {{NTEXT}} NOT NULL,
		owner VARCHAR(36) NOT NULL,
		PRIMARY KEY (tenant, id)
	);

	CREATE TABLE ActivityTeachers (
		tenant {{NVARCHAR}}(255) NOT NULL,
		activityId VARCHAR(36) NOT NULL,
		employmentId VARCHAR(36) NOT NULL,
		FOREIGN KEY (tenant, activityId) REFERENCES Activities(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX ActivityTeachersIdx ON ActivityTeachers (tenant, activityId);

	CREATE TABLE ActivityGroups (
		tenant {{NVARCHAR}}(255) NOT NULL,
		activityId VARCHAR(36) NOT NULL,
		groupId VARCHAR(36) NOT NULL,
		FOREIGN KEY (tenant, activityId) REFERENCES Activities(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX ActivityGroupsIdx ON ActivityGroups (tenant, activityId);
	`,

	// v2: adds support for external identifiers
	`
	CREATE TABLE ExternalIdentifiers (
		tenant {{NVARCHAR}}(255) NOT NULL,
		userId VARCHAR(36) NOT NULL,
		value {{NTEXT}} NOT NULL,
		context {{NTEXT}} NULL,
		globallyUnique TINYINT NOT NULL,
		FOREIGN KEY (tenant, userId) REFERENCES Users(tenant, id) ON DELETE CASCADE
	);

	CREATE INDEX ExternalIdentifiersIdx ON ExternalIdentifiers (tenant, userId);	
	`,
}

var downgrades = [...]string{
	// v1 - do nothing, we'll never downgrade below 1
	`
	`,

	// v2: removes support for external identifiers
	`
	DROP TABLE ExternalIdentifiers;
	`,
}

func currentSchemaVersion() int {
	return len(migrations)
}

// Returns SQL statements to migrate the database from (version-1) to version
func getSchema(version int) string {
	return migrations[version-1]
}

// Returns SQL statements to downgrade from version to (version-1)
func getDowngrade(version int) string {
	return downgrades[version-1]
}

func driverSpecificInit(db *sqlx.DB) error {
	if db.DriverName() == "sqlite" {
		_, err := db.Exec(`PRAGMA foreign_keys = ON;`)
		return err
	}
	return nil
}

func expandDriverSpecificTypes(driverName, schema string) string {
	removeCurlies := func(schema string) string {
		re := regexp.MustCompile(`{{(.*?)}}`)
		return string(re.ReplaceAll([]byte(schema), []byte("$1")))
	}
	// Default expansion simply removes curly brackets
	expander := removeCurlies

	if driverName == "mysql" {
		// For MySQL we'll replace NTEXT and NVARCHAR with TEXT and VARCHAR
		expander = func(schema string) string {
			re := regexp.MustCompile(`{{N(.*?)}}`)
			return removeCurlies(string(re.ReplaceAll([]byte(schema), []byte("$1"))))
		}
	}
	return expander(schema)
}

// Makes sure we have a working connection to the database,
// does driver specific initialization and retrieves
// the version number of the current database schema in the database.
// Returns error if anything fails or if the database's version is
// higher than supported by this version of Windermere.
func initDBConnection(db *sqlx.DB) (int, error) {
	// Ensure we have a working connection since any error in
	// getDBVersion is interpreted as an uninitialized database.
	const waitTime = 5 * time.Second
	for err := db.Ping(); err != nil; err = db.Ping() {
		log.Printf("Failed to connect to database: %v", err)
		log.Printf("Will retry in %d seconds", waitTime/time.Second)
		time.Sleep(waitTime)
	}
	if err := driverSpecificInit(db); err != nil {
		return 0, err
	}
	version := getDBVersion(db)

	if version > currentSchemaVersion() {
		return 0, fmt.Errorf("database schema is newer than this version of Windermere. Please perform a database schema downgrade if you wish to continue with this version of Windermere.")
	}
	return version, nil
}

func (backend *SQLBackend) initSchema() error {
	version, err := initDBConnection(backend.db)
	if err != nil {
		return err
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()
	// loop over all migrations in order and apply those with higher
	// version than current
	for i := version + 1; i <= currentSchemaVersion(); i++ {
		_, err = tx.Exec(expandDriverSpecificTypes(backend.db.DriverName(), getSchema(i)))
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

func DowngradeDBSchema(storageType, storageSource string, downgradeTo int) error {
	db, err := openDB(storageType, storageSource)

	if err != nil {
		return fmt.Errorf("failed to open connection to database: %v", err)
	}

	version, err := initDBConnection(db)
	if err != nil {
		return err
	}

	if version <= downgradeTo {
		return fmt.Errorf("current database version is %d, can't downgrade to %d", version, downgradeTo)
	}

	tx, err := db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	for i := version; i > downgradeTo; i-- {
		_, err = tx.Exec(expandDriverSpecificTypes(db.DriverName(), getDowngrade(i)))
		if err != nil {
			return err
		}
	}

	// Set the current schema version
	tx.NamedExec(`UPDATE windermere_meta SET version = :version`, map[string]interface{}{"version": downgradeTo})

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

func (backend *SQLBackend) create(tx *sqlx.Tx, tenant string, resourceType string, obj ss12000v1.Object) error {
	table, err := mainTable(resourceType)

	if err != nil {
		return err
	}

	err = ensureDoesntHaveRecord(tx, table, tenant, obj.GetID())
	if err != nil {
		return err
	}

	_, err = backend.objectCreator(tx, tenant, obj)
	return err
}

func (backend *SQLBackend) Create(tenant, resourceType, resource string) (string, error) {
	obj, err := backend.objectParser(resourceType, resource)

	if err != nil {
		return "", scim.NewError(scim.MalformedResourceError, "Failed to parse resource:\n"+err.Error())
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	err = backend.create(tx, tenant, resourceType, obj)

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

func (backend *SQLBackend) update(tx *sqlx.Tx, tenant string, resourceType string, resourceID string, obj ss12000v1.Object) error {
	table, err := mainTable(resourceType)

	if err != nil {
		return err
	}

	err = ensureHasRecord(tx, table, tenant, resourceID)
	if err != nil {
		return err
	}

	return backend.objectMutator(tx, tenant, obj)
}

func (backend *SQLBackend) Update(tenant, resourceType, resourceID, resource string) (string, error) {
	obj, err := backend.objectParser(resourceType, resource)

	if err != nil {
		return "", scim.NewError(scim.MalformedResourceError, "Failed to parse resource:\n"+err.Error())
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	err = backend.update(tx, tenant, resourceType, resourceID, obj)
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

func (backend *SQLBackend) delete(tx *sqlx.Tx, tenant string, resourceType, resourceID string) error {
	table, err := mainTable(resourceType)

	if err != nil {
		return err
	}

	err = ensureHasRecord(tx, table, tenant, resourceID)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM `+string(table)+` WHERE tenant = :tenant AND id = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     resourceID,
		})

	return err
}

func (backend *SQLBackend) Delete(tenant, resourceType, resourceID string) error {
	tx, err := backend.db.Beginx()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	err = backend.delete(tx, tenant, resourceType, resourceID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (backend *SQLBackend) Bulk(ctx context.Context, tenant string, operations []scimserverlite.BulkOperation) ([]scimserverlite.BulkOperationResult, error) {
	// TODO: Add protection against too many failures?
	//       Since failures means new transactions we don't want to do too many if there
	//       are thousands of bad bulk operations.
	//       On the other hand SCIM specification might not allow that.
	//
	// TODO: Currently parse failures are treated like database constraint errors, so parse
	//       errors can lead to many unnecessary transactions and restarted transactions.
	//       Currently the Bulk function is only used by the SS1200v2 import which means
	//       we shouldn't have parse errors. If this is used for SCIM bulk requests however
	//       we should probably pre-parse all objects and handle parse errors separately before
	//       dealing with the objects that parsed correctly. If we do that we should make sure
	//       to still return the results in the same order as the bulk operations.
	const TransactionMaxSize = 50

	if ok, msg := util.IsDone(ctx); ok {
		return nil, fmt.Errorf("Bulk operation terminated prematurely: %s", msg)
	}

	bulkSize := len(operations)
	if bulkSize == 0 {
		return nil, nil
	} else if bulkSize == 1 {
		op := operations[0]
		var err error
		switch op.Type {
		case scimserverlite.CreateOperation:
			_, err = backend.Create(tenant, op.ResourceType, op.Resource)
		case scimserverlite.UpdateOperation:
			_, err = backend.Update(tenant, op.ResourceType, op.ResourceID, op.Resource)
		case scimserverlite.DeleteOperation:
			err = backend.Delete(tenant, op.ResourceType, op.ResourceID)
		default:
			err = fmt.Errorf("Unexpected bulk operation: %v", op.Type)
		}
		return []scimserverlite.BulkOperationResult{scimserverlite.NewBulkOperationResult(op, err)}, nil
	} else if bulkSize > TransactionMaxSize {
		mid := bulkSize / 2
		listA, err := backend.Bulk(ctx, tenant, operations[0:mid])
		if err != nil {
			return nil, err
		}
		listB, err := backend.Bulk(ctx, tenant, operations[mid:])
		if err != nil {
			return nil, err
		}
		return append(listA, listB...), nil
	}

	tx, err := backend.db.Beginx()

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	resultList := make([]scimserverlite.BulkOperationResult, 0)
	for i, op := range operations {
		var err error
		var obj ss12000v1.Object
		if op.Type == scimserverlite.CreateOperation || op.Type == scimserverlite.UpdateOperation {
			obj, err = backend.objectParser(op.ResourceType, op.Resource)

			if err != nil {
				err = scim.NewError(scim.MalformedResourceError, "Failed to parse resource:\n"+err.Error())
			}
		}
		if err == nil {
			switch op.Type {
			case scimserverlite.CreateOperation:
				err = backend.create(tx, tenant, op.ResourceType, obj)
			case scimserverlite.UpdateOperation:
				err = backend.update(tx, tenant, op.ResourceType, op.ResourceID, obj)
			case scimserverlite.DeleteOperation:
				err = backend.delete(tx, tenant, op.ResourceType, op.ResourceID)
			default:
				err = fmt.Errorf("Unexpected bulk operation: %v", op.Type)
			}
		}
		if err != nil {
			tx.Rollback()
			// If the failure was already on the first operation in the transaction,
			// there's no need to retry that one, we can just accept the failure and
			// continue with the rest of the operations in a new transaction.
			if i == 0 {
				resultList = append(resultList, scim.NewBulkOperationResult(op, err))
				rest, err := backend.Bulk(ctx, tenant, operations[1:])
				if err != nil {
					return nil, err
				}
				resultList = append(resultList, rest...)
				return resultList, nil
			}

			// Otherwise we re-run the operations before the failure in a separate
			// transaction, retry this one separately, and the rest separately.
			listA, err := backend.Bulk(ctx, tenant, operations[0:i])
			if err != nil {
				return nil, err
			}
			listB, err := backend.Bulk(ctx, tenant, operations[i:i+1])
			if err != nil {
				return nil, err
			}
			listC, err := backend.Bulk(ctx, tenant, operations[i+1:])
			if err != nil {
				return nil, err
			}
			resultList = append(listA, listB...)
			resultList = append(resultList, listC...)
			return resultList, nil
		} else {
			resultList = append(resultList, scim.NewBulkOperationResult(op, nil))
		}
	}

	err = tx.Commit()

	if err == nil {
		return resultList, nil
	}

	tx.Rollback() // Not necessary after a failed Commit?

	// If we get here it means we were able to execute all operations
	// without errors but failed to commit the whole transaction.
	// In this case we'll simply run the operations in individual transactions
	// instead.
	resultList = make([]scimserverlite.BulkOperationResult, 0)
	for _, op := range operations {
		var err error
		switch op.Type {
		case scimserverlite.CreateOperation:
			_, err = backend.Create(tenant, op.ResourceType, op.Resource)
		case scimserverlite.UpdateOperation:
			_, err = backend.Update(tenant, op.ResourceType, op.ResourceID, op.Resource)
		case scimserverlite.DeleteOperation:
			err = backend.Delete(tenant, op.ResourceType, op.ResourceID)
		default:
			err = fmt.Errorf("Unexpected bulk operation: %v", op.Type)
		}
		resultList = append(resultList, scim.NewBulkOperationResult(op, err))
	}

	return resultList, nil
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
