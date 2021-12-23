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

type dbEmploymentRow struct {
	Tenant         string  `db:"tenant"`
	Id             string  `db:"id"`
	EmployedAt     string  `db:"employedAt"`
	User           string  `db:"user"`
	EmploymentRole string  `db:"employmentRole"`
	Signature      *string `db:"signature"`
}

func NewEmploymentRow(tenant string, employment *ss12000v1.Employment) dbEmploymentRow {
	var sig *string
	if employment.Signature != "" {
		sig = &employment.Signature
	}
	return dbEmploymentRow{
		Tenant:         tenant,
		Id:             employment.GetID(),
		EmployedAt:     employment.EmployedAt.Value,
		User:           employment.User.Value,
		EmploymentRole: employment.EmploymentRole,
		Signature:      sig,
	}
}

func (backend *SQLBackend) employmentCreator(tx *sqlx.Tx, tenant string, employment *ss12000v1.Employment) (id string, err error) {
	dbEmployment := NewEmploymentRow(tenant, employment)

	_, err = tx.NamedExec(`INSERT INTO Employments (tenant, id, employedAt, user, employmentRole, signature) VALUES (:tenant, :id, :employedAt, :user, :employmentRole, :signature)`, &dbEmployment)
	return employment.GetID(), err
}

func (backend *SQLBackend) employmentMutator(tx *sqlx.Tx, tenant string, employment *ss12000v1.Employment) (err error) {
	dbEmployment := NewEmploymentRow(tenant, employment)

	_, err = tx.NamedExec(`UPDATE Employments SET employedAt = :employedAt, user = :user, employmentRole = :employmentRole, signature = :signature WHERE tenant = :tenant AND id = :id`, &dbEmployment)
	return
}

func (backend *SQLBackend) employmentReader(tx *sqlx.Tx, mainQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}

	dbEmployments := []dbEmploymentRow{}
	err = mainNamed.Select(&dbEmployments, args)
	if err != nil {
		return nil, err
	}

	employments := make([]ss12000v1.Object, len(dbEmployments))
	for i := range dbEmployments {
		sig := ""
		if dbEmployments[i].Signature != nil {
			sig = *dbEmployments[i].Signature
		}
		employments[i] = &ss12000v1.Employment{
			ID: dbEmployments[i].Id,
			EmployedAt: ss12000v1.SCIMReference{
				Value: dbEmployments[i].EmployedAt,
			},
			User: ss12000v1.SCIMReference{
				Value: dbEmployments[i].User,
			},
			EmploymentRole: dbEmployments[i].EmploymentRole,
			Signature:      sig,
		}
	}
	return employments, nil
}

func (backend *SQLBackend) employmentReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.employmentReader(tx, `SELECT * FROM Employments WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) employmentReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	employments, err := backend.employmentReader(tx, `SELECT * FROM Employments WHERE tenant = :tenant AND id = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(employments) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(employments))
	}
	return employments[0], nil
}
