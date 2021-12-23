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

type dbActivityRow struct {
	Tenant      string `db:"tenant"`
	Id          string `db:"id"`
	DisplayName string `db:"displayName"`
	Owner       string `db:"owner"`
}

func NewActivityRow(tenant string, activity *ss12000v1.Activity) dbActivityRow {
	return dbActivityRow{
		Tenant:      tenant,
		Id:          activity.GetID(),
		DisplayName: activity.DisplayName,
		Owner:       activity.Owner.Value,
	}
}

type dbActivityTeacherRow struct {
	Tenant       string `db:"tenant"`
	ActivityId   string `db:"activityId"`
	EmploymentId string `db:"employmentId"`
}

type dbActivityGroupRow struct {
	Tenant     string `db:"tenant"`
	ActivityId string `db:"activityId"`
	GroupId    string `db:"groupId"`
}

func (backend *SQLBackend) createTeachers(tx *sqlx.Tx, tenant string, activity *ss12000v1.Activity) (err error) {
	if len(activity.Teachers) == 0 {
		return nil
	}
	dbTeachers := make([]dbActivityTeacherRow, len(activity.Teachers))

	for i := range activity.Teachers {
		dbTeachers[i] = dbActivityTeacherRow{
			Tenant:       tenant,
			ActivityId:   activity.GetID(),
			EmploymentId: activity.Teachers[i].Value,
		}
	}

	_, err = tx.NamedExec(`INSERT INTO ActivityTeachers (tenant, activityId, employmentId) VALUES (:tenant, :activityId, :employmentId)`, dbTeachers)
	return
}

func (backend *SQLBackend) createGroups(tx *sqlx.Tx, tenant string, activity *ss12000v1.Activity) (err error) {
	if len(activity.Groups) == 0 {
		return nil
	}
	dbGroups := make([]dbActivityGroupRow, len(activity.Groups))

	for i := range activity.Groups {
		dbGroups[i] = dbActivityGroupRow{
			Tenant:     tenant,
			ActivityId: activity.GetID(),
			GroupId:    activity.Groups[i].Value,
		}
	}

	_, err = tx.NamedExec(`INSERT INTO ActivityGroups (tenant, activityId, groupId) VALUES (:tenant, :activityId, :groupId)`, dbGroups)
	return
}

func (backend *SQLBackend) activityCreator(tx *sqlx.Tx, tenant string, activity *ss12000v1.Activity) (id string, err error) {
	dbActivity := NewActivityRow(tenant, activity)

	_, err = tx.NamedExec(`INSERT INTO Activities (tenant, id, displayName, owner) VALUES (:tenant, :id, :displayName, :owner)`, &dbActivity)
	if err != nil {
		return "", err
	}

	err = backend.createTeachers(tx, tenant, activity)
	if err != nil {
		return "", err
	}

	err = backend.createGroups(tx, tenant, activity)
	return activity.GetID(), err
}

func (backend *SQLBackend) activityMutator(tx *sqlx.Tx, tenant string, activity *ss12000v1.Activity) (err error) {
	dbActivity := NewActivityRow(tenant, activity)

	_, err = tx.NamedExec(`UPDATE Activities SET displayName = :displayName, owner = :owner WHERE tenant = :tenant AND id = :id`, &dbActivity)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM ActivityTeachers WHERE tenant = :tenant AND activityId = :activityId`,
		map[string]interface{}{
			"tenant":     tenant,
			"activityId": activity.GetID(),
		})

	if err != nil {
		return err
	}

	err = backend.createTeachers(tx, tenant, activity)

	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM ActivityGroups WHERE tenant = :tenant AND activityId = :activityId`,
		map[string]interface{}{
			"tenant":     tenant,
			"activityId": activity.GetID(),
		})

	if err != nil {
		return err
	}

	return backend.createGroups(tx, tenant, activity)
}

func (backend *SQLBackend) activityReader(tx *sqlx.Tx, mainQuery, teacherQuery, groupQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}
	teacherNamed, err := tx.PrepareNamed(teacherQuery)
	if err != nil {
		return nil, err
	}
	groupNamed, err := tx.PrepareNamed(groupQuery)
	if err != nil {
		return nil, err
	}

	dbActivities := []dbActivityRow{}
	err = mainNamed.Select(&dbActivities, args)
	if err != nil {
		return nil, err
	}

	activities := make([]ss12000v1.Object, len(dbActivities))
	index := make(map[string]int)
	for i := range dbActivities {
		activities[i] = &ss12000v1.Activity{
			ExternalID:  dbActivities[i].Id,
			DisplayName: dbActivities[i].DisplayName,
			Owner: ss12000v1.SCIMReference{
				Value: dbActivities[i].Owner,
			},
		}
		index[dbActivities[i].Id] = i
	}

	dbTeachers := []dbActivityTeacherRow{}
	err = teacherNamed.Select(&dbTeachers, args)
	if err != nil {
		return nil, err
	}

	for i := range dbTeachers {
		teacher := &dbTeachers[i]
		activity := activities[index[teacher.ActivityId]].(*ss12000v1.Activity)
		activity.Teachers = append(activity.Teachers, ss12000v1.SCIMReference{
			Value: teacher.EmploymentId,
		})
	}

	dbGroups := []dbActivityGroupRow{}
	err = groupNamed.Select(&dbGroups, args)
	if err != nil {
		return nil, err
	}

	for i := range dbGroups {
		group := &dbGroups[i]
		activity := activities[index[group.ActivityId]].(*ss12000v1.Activity)
		activity.Groups = append(activity.Groups, ss12000v1.SCIMReference{
			Value: group.GroupId,
		})
	}

	return activities, nil
}

func (backend *SQLBackend) activityReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.activityReader(tx, `SELECT * FROM Activities WHERE tenant = :tenant`,
		`SELECT * FROM ActivityTeachers WHERE tenant = :tenant`,
		`SELECT * FROM ActivityGroups WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) activityReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	activities, err := backend.activityReader(tx, `SELECT * FROM Activities WHERE tenant = :tenant AND id = :id`,
		`SELECT * FROM ActivityTeachers WHERE tenant = :tenant AND activityId = :id`,
		`SELECT * FROM ActivityGroups WHERE tenant = :tenant AND activityId = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(activities) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(activities))
	}
	return activities[0], nil
}
