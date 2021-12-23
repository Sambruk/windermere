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
	"fmt"

	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/jmoiron/sqlx"
)

type dbSchoolUnitGroupRow struct {
	Tenant      string `db:"tenant"`
	Id          string `db:"id"`
	DisplayName string `db:"displayName"`
}

func NewSchoolUnitGroupRow(tenant string, schoolUnitGroup *ss12000v1.SchoolUnitGroup) dbSchoolUnitGroupRow {
	return dbSchoolUnitGroupRow{
		Tenant:      tenant,
		Id:          schoolUnitGroup.ExternalID,
		DisplayName: schoolUnitGroup.DisplayName,
	}
}

func (backend *SQLBackend) schoolUnitGroupCreator(tx *sqlx.Tx, tenant string, schoolUnitGroup *ss12000v1.SchoolUnitGroup) (id string, err error) {
	dbSchoolUnitGroup := NewSchoolUnitGroupRow(tenant, schoolUnitGroup)

	_, err = tx.NamedExec(`INSERT INTO SchoolUnitGroups (tenant, id, displayName) VALUES (:tenant, :id, :displayName)`, &dbSchoolUnitGroup)
	return schoolUnitGroup.GetID(), err
}

func (backend *SQLBackend) schoolUnitGroupMutator(tx *sqlx.Tx, tenant string, schoolUnitGroup *ss12000v1.SchoolUnitGroup) (err error) {
	dbSchoolUnitGroup := NewSchoolUnitGroupRow(tenant, schoolUnitGroup)

	_, err = tx.NamedExec(`UPDATE SchoolUnitGroups SET displayName = :displayName WHERE tenant = :tenant AND id = :id`, &dbSchoolUnitGroup)
	return
}

func (backend *SQLBackend) schoolUnitGroupReader(tx *sqlx.Tx, mainQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}

	dbSchoolUnitGroups := []dbSchoolUnitGroupRow{}
	err = mainNamed.Select(&dbSchoolUnitGroups, args)
	if err != nil {
		return nil, err
	}

	schoolUnitGroups := make([]ss12000v1.Object, len(dbSchoolUnitGroups))
	for i := range dbSchoolUnitGroups {
		schoolUnitGroups[i] = &ss12000v1.SchoolUnitGroup{
			ExternalID:  dbSchoolUnitGroups[i].Id,
			DisplayName: dbSchoolUnitGroups[i].DisplayName,
		}
	}
	return schoolUnitGroups, nil
}

func (backend *SQLBackend) schoolUnitGroupReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.schoolUnitGroupReader(tx, `SELECT * FROM SchoolUnitGroups WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) schoolUnitGroupReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	schoolUnitGroups, err := backend.schoolUnitGroupReader(tx, `SELECT * FROM SchoolUnitGroups WHERE tenant = :tenant AND id = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(schoolUnitGroups) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(schoolUnitGroups))
	}
	return schoolUnitGroups[0], nil
}
