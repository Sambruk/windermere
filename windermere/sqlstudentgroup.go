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

type dbStudentGroupRow struct {
	Tenant           string  `db:"tenant"`
	Id               string  `db:"id"`
	DisplayName      string  `db:"displayName"`
	Owner            string  `db:"owner"`
	StudentGroupType *string `db:"studentGroupType"`
}

func NewStudentGroupRow(tenant string, group *ss12000v1.StudentGroup) dbStudentGroupRow {
	return dbStudentGroupRow{
		Tenant:           tenant,
		Id:               group.ID,
		DisplayName:      group.DisplayName,
		Owner:            group.Owner.Value,
		StudentGroupType: group.Type,
	}
}

type dbStudentMembershipRow struct {
	Tenant  string `db:"tenant"`
	GroupId string `db:"groupId"`
	UserId  string `db:"userId"`
}

func (backend *SQLBackend) createMemberships(tx *sqlx.Tx, tenant string, group *ss12000v1.StudentGroup) (err error) {
	if len(group.StudentMemberships) == 0 {
		return nil
	}
	dbMemberships := make([]dbStudentMembershipRow, len(group.StudentMemberships))

	for i := range group.StudentMemberships {
		dbMemberships[i] = dbStudentMembershipRow{
			Tenant:  tenant,
			GroupId: group.ID,
			UserId:  group.StudentMemberships[i].Value,
		}
	}

	_, err = tx.NamedExec(`INSERT INTO StudentMemberships (tenant, groupId, userId) VALUES (:tenant, :groupId, :userId)`, dbMemberships)
	return
}

func (backend *SQLBackend) studentGroupCreator(tx *sqlx.Tx, tenant string, group *ss12000v1.StudentGroup) (id string, err error) {
	dbGroup := NewStudentGroupRow(tenant, group)

	_, err = tx.NamedExec(`INSERT INTO StudentGroups (tenant, id, displayName, owner, studentGroupType) VALUES (:tenant, :id, :displayName, :owner, :studentGroupType)`, &dbGroup)
	if err != nil {
		return "", err
	}

	err = backend.createMemberships(tx, tenant, group)
	if err != nil {
		return "", err
	}
	return group.ID, err
}

func (backend *SQLBackend) studentGroupMutator(tx *sqlx.Tx, tenant string, group *ss12000v1.StudentGroup) (err error) {
	dbGroup := NewStudentGroupRow(tenant, group)

	_, err = tx.NamedExec(`UPDATE StudentGroups SET displayName = :displayName, owner = :owner, studentGroupType = :studentGroupType WHERE tenant = :tenant AND id = :id`, &dbGroup)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM StudentMemberships WHERE tenant = :tenant AND groupId = :groupId`,
		map[string]interface{}{
			"tenant":  tenant,
			"groupId": group.ID,
		})

	if err != nil {
		return err
	}

	return backend.createMemberships(tx, tenant, group)
}

func (backend *SQLBackend) studentGroupReader(tx *sqlx.Tx, mainQuery, membershipQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}
	membershipNamed, err := tx.PrepareNamed(membershipQuery)
	if err != nil {
		return nil, err
	}

	dbGroups := []dbStudentGroupRow{}
	err = mainNamed.Select(&dbGroups, args)
	if err != nil {
		return nil, err
	}

	groups := make([]ss12000v1.Object, len(dbGroups))
	index := make(map[string]int)
	for i := range dbGroups {
		groups[i] = &ss12000v1.StudentGroup{
			ID:          dbGroups[i].Id,
			DisplayName: dbGroups[i].DisplayName,
			Owner: ss12000v1.SCIMReference{
				Value: dbGroups[i].Owner,
			},
			Type: dbGroups[i].StudentGroupType,
		}
		index[dbGroups[i].Id] = i
	}

	dbMemberships := []dbStudentMembershipRow{}
	err = membershipNamed.Select(&dbMemberships, args)
	if err != nil {
		return nil, err
	}

	for i := range dbMemberships {
		membership := &dbMemberships[i]
		group := groups[index[membership.GroupId]].(*ss12000v1.StudentGroup)
		group.StudentMemberships = append(group.StudentMemberships, ss12000v1.SCIMReference{
			Value: membership.UserId,
		})
	}

	return groups, nil
}

func (backend *SQLBackend) studentGroupReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.studentGroupReader(tx, `SELECT * FROM StudentGroups WHERE tenant = :tenant`,
		`SELECT * FROM StudentMemberships WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) studentGroupReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	groups, err := backend.studentGroupReader(tx, `SELECT * FROM StudentGroups WHERE tenant = :tenant AND id = :id`,
		`SELECT * FROM StudentMemberships WHERE tenant = :tenant AND groupId = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(groups) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(groups))
	}
	return groups[0], nil
}
