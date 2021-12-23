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

type dbSchoolUnitRow struct {
	Tenant           string  `db:"tenant"`
	Id               string  `db:"id"`
	DisplayName      string  `db:"displayName"`
	SchoolUnitCode   string  `db:"schoolUnitCode"`
	Organisation     *string `db:"organisation"`
	SchoolUnitGroup  *string `db:"schoolUnitGroup"`
	MunicipalityCode *string `db:"municipalityCode"`
}

func NewSchoolUnitRow(tenant string, schoolUnit *ss12000v1.SchoolUnit) dbSchoolUnitRow {
	var org, sug *string

	if schoolUnit.Organisation != nil {
		org = &schoolUnit.Organisation.Value
	}

	if schoolUnit.SchoolUnitGroup != nil {
		sug = &schoolUnit.SchoolUnitGroup.Value
	}

	return dbSchoolUnitRow{
		Tenant:           tenant,
		Id:               schoolUnit.GetID(),
		DisplayName:      schoolUnit.DisplayName,
		SchoolUnitCode:   schoolUnit.SchoolUnitCode,
		Organisation:     org,
		SchoolUnitGroup:  sug,
		MunicipalityCode: schoolUnit.MunicipalityCode,
	}
}

type dbSchoolTypeRow struct {
	Tenant       string `db:"tenant"`
	SchoolUnitId string `db:"schoolUnitId"`
	SchoolType   string `db:"schoolType"`
}

func (backend *SQLBackend) createSchoolTypes(tx *sqlx.Tx, tenant string, schoolUnit *ss12000v1.SchoolUnit) (err error) {
	if schoolUnit.SchoolTypes == nil || len(*schoolUnit.SchoolTypes) == 0 {
		return nil
	}
	dbSchoolTypes := make([]dbSchoolTypeRow, len(*schoolUnit.SchoolTypes))

	for i := range *schoolUnit.SchoolTypes {
		dbSchoolTypes[i] = dbSchoolTypeRow{
			Tenant:       tenant,
			SchoolUnitId: schoolUnit.GetID(),
			SchoolType:   (*schoolUnit.SchoolTypes)[i],
		}
	}

	_, err = tx.NamedExec(`INSERT INTO SchoolTypes (tenant, schoolUnitId, schoolType) VALUES (:tenant, :schoolUnitId, :schoolType)`, dbSchoolTypes)
	return
}

func (backend *SQLBackend) schoolUnitCreator(tx *sqlx.Tx, tenant string, schoolUnit *ss12000v1.SchoolUnit) (id string, err error) {
	dbSchoolUnit := NewSchoolUnitRow(tenant, schoolUnit)

	_, err = tx.NamedExec(`INSERT INTO SchoolUnits (tenant, id, displayName, schoolUnitCode, organisation, schoolUnitGroup, municipalityCode) VALUES (:tenant, :id, :displayName, :schoolUnitCode, :organisation, :schoolUnitGroup, :municipalityCode)`, &dbSchoolUnit)
	if err != nil {
		return "", err
	}

	err = backend.createSchoolTypes(tx, tenant, schoolUnit)
	return schoolUnit.GetID(), err
}

func (backend *SQLBackend) schoolUnitMutator(tx *sqlx.Tx, tenant string, schoolUnit *ss12000v1.SchoolUnit) (err error) {
	dbSchoolUnit := NewSchoolUnitRow(tenant, schoolUnit)

	_, err = tx.NamedExec(`UPDATE SchoolUnits SET displayName = :displayName, schoolUnitCode = :schoolUnitCode, organisation = :organisation, schoolUnitGroup = :schoolUnitGroup, municipalityCode = :municipalityCode WHERE tenant = :tenant AND id = :id`, &dbSchoolUnit)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM SchoolTypes WHERE tenant = :tenant AND schoolUnitId = :schoolUnitId`,
		map[string]interface{}{
			"tenant":       tenant,
			"schoolUnitId": schoolUnit.GetID(),
		})

	if err != nil {
		return err
	}

	return backend.createSchoolTypes(tx, tenant, schoolUnit)
}

func (backend *SQLBackend) schoolUnitReader(tx *sqlx.Tx, mainQuery, schoolTypeQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}
	schoolTypeNamed, err := tx.PrepareNamed(schoolTypeQuery)
	if err != nil {
		return nil, err
	}

	dbSchoolUnits := []dbSchoolUnitRow{}
	err = mainNamed.Select(&dbSchoolUnits, args)
	if err != nil {
		return nil, err
	}

	schoolUnits := make([]ss12000v1.Object, len(dbSchoolUnits))
	index := make(map[string]int)
	for i := range dbSchoolUnits {
		var org, sug *ss12000v1.SCIMReference

		if dbSchoolUnits[i].Organisation != nil {
			org = &ss12000v1.SCIMReference{
				Value: *dbSchoolUnits[i].Organisation,
			}
		}

		if dbSchoolUnits[i].SchoolUnitGroup != nil {
			sug = &ss12000v1.SCIMReference{
				Value: *dbSchoolUnits[i].SchoolUnitGroup,
			}
		}

		schoolUnits[i] = &ss12000v1.SchoolUnit{
			ExternalID:       dbSchoolUnits[i].Id,
			DisplayName:      dbSchoolUnits[i].DisplayName,
			SchoolUnitCode:   dbSchoolUnits[i].SchoolUnitCode,
			Organisation:     org,
			SchoolUnitGroup:  sug,
			MunicipalityCode: dbSchoolUnits[i].MunicipalityCode,
		}
		index[dbSchoolUnits[i].Id] = i
	}

	dbSchoolTypes := []dbSchoolTypeRow{}
	err = schoolTypeNamed.Select(&dbSchoolTypes, args)
	if err != nil {
		return nil, err
	}

	for i := range dbSchoolTypes {
		schoolType := &dbSchoolTypes[i]
		schoolUnit := schoolUnits[index[schoolType.SchoolUnitId]].(*ss12000v1.SchoolUnit)
		if schoolUnit.SchoolTypes == nil {
			new := make([]string, 0)
			schoolUnit.SchoolTypes = &new
		}
		*schoolUnit.SchoolTypes = append(*schoolUnit.SchoolTypes, schoolType.SchoolType)
	}

	return schoolUnits, nil
}

func (backend *SQLBackend) schoolUnitReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.schoolUnitReader(tx, `SELECT * FROM SchoolUnits WHERE tenant = :tenant`,
		`SELECT * FROM SchoolTypes WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) schoolUnitReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	schoolUnits, err := backend.schoolUnitReader(tx, `SELECT * FROM SchoolUnits WHERE tenant = :tenant AND id = :id`,
		`SELECT * FROM SchoolTypes WHERE tenant = :tenant AND schoolUnitId = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(schoolUnits) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(schoolUnits))
	}
	return schoolUnits[0], nil
}
