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

type dbOrganisationRow struct {
	Tenant      string `db:"tenant"`
	Id          string `db:"id"`
	DisplayName string `db:"displayName"`
}

func NewOrganisationRow(tenant string, organisation *ss12000v1.Organisation) dbOrganisationRow {
	return dbOrganisationRow{
		Tenant:      tenant,
		Id:          organisation.ExternalID,
		DisplayName: organisation.DisplayName,
	}
}

func (backend *SQLBackend) organisationCreator(tx *sqlx.Tx, tenant string, organisation *ss12000v1.Organisation) (id string, err error) {
	dbOrganisation := NewOrganisationRow(tenant, organisation)

	_, err = tx.NamedExec(`INSERT INTO Organisations (tenant, id, displayName) VALUES (:tenant, :id, :displayName)`, &dbOrganisation)
	return organisation.GetID(), err
}

func (backend *SQLBackend) organisationMutator(tx *sqlx.Tx, tenant string, organisation *ss12000v1.Organisation) (err error) {
	dbOrganisation := NewOrganisationRow(tenant, organisation)

	_, err = tx.NamedExec(`UPDATE Organisations SET displayName = :displayName WHERE tenant = :tenant AND id = :id`, &dbOrganisation)
	return
}

func (backend *SQLBackend) organisationReader(tx *sqlx.Tx, mainQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}

	dbOrganisations := []dbOrganisationRow{}
	err = mainNamed.Select(&dbOrganisations, args)
	if err != nil {
		return nil, err
	}

	organisations := make([]ss12000v1.Object, len(dbOrganisations))
	for i := range dbOrganisations {
		organisations[i] = &ss12000v1.Organisation{
			ExternalID:  dbOrganisations[i].Id,
			DisplayName: dbOrganisations[i].DisplayName,
		}
	}
	return organisations, nil
}

func (backend *SQLBackend) organisationReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.organisationReader(tx, `SELECT * FROM Organisations WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) organisationReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	organisations, err := backend.organisationReader(tx, `SELECT * FROM Organisations WHERE tenant = :tenant AND id = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(organisations) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(organisations))
	}
	return organisations[0], nil
}
