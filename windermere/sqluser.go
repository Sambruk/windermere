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

type dbUserRow struct {
	Tenant      string `db:"tenant"`
	Id          string `db:"id"`
	UserName    string `db:"userName"`
	FamilyName  string `db:"familyName"`
	GivenName   string `db:"givenName"`
	DisplayName string `db:"displayName"`
}

func NewUserRow(tenant string, user *ss12000v1.User) dbUserRow {
	return dbUserRow{
		Tenant:      tenant,
		Id:          user.ID,
		UserName:    user.UserName,
		FamilyName:  user.Name.FamilyName,
		GivenName:   user.Name.GivenName,
		DisplayName: user.DisplayName,
	}
}

type dbEmailRow struct {
	Tenant string  `db:"tenant"`
	UserId string  `db:"userId"`
	Value  string  `db:"value"`
	Type   *string `db:"type"`
}

type dbEnrolmentRow struct {
	Tenant     string `db:"tenant"`
	UserId     string `db:"userId"`
	Value      string `db:"value"`
	SchoolYear *int   `db:"schoolYear"`
}

type dbExternalIdentifierRow struct {
	Tenant         string `db:"tenant"`
	UserId         string `db:"userId"`
	Value          string `db:"value"`
	Context        string `db:"context"`
	GloballyUnique int    `db:"globallyUnique"`
}

func (backend *SQLBackend) createEmails(tx *sqlx.Tx, tenant string, user *ss12000v1.User) (err error) {
	if len(user.Emails) == 0 {
		return nil
	}
	dbEmails := make([]dbEmailRow, len(user.Emails))

	for i := range user.Emails {
		dbEmails[i] = dbEmailRow{
			Tenant: tenant,
			UserId: user.ID,
			Value:  user.Emails[i].Value,
		}
		if user.Emails[i].Type != "" {
			dbEmails[i].Type = &user.Emails[i].Type
		}
	}

	_, err = tx.NamedExec(`INSERT INTO Emails (tenant, userId, value, type) VALUES (:tenant, :userId, :value, :type)`, dbEmails)
	return
}

func (backend *SQLBackend) createEnrolments(tx *sqlx.Tx, tenant string, user *ss12000v1.User) (err error) {
	if len(user.Extension.Enrolments) == 0 {
		return nil
	}
	dbEnrolments := make([]dbEnrolmentRow, len(user.Extension.Enrolments))

	for i := range user.Extension.Enrolments {
		dbEnrolments[i] = dbEnrolmentRow{
			Tenant:     tenant,
			UserId:     user.ID,
			Value:      user.Extension.Enrolments[i].Value,
			SchoolYear: user.Extension.Enrolments[i].SchoolYear,
		}
	}

	_, err = tx.NamedExec(`INSERT INTO Enrolments (tenant, userId, value, schoolYear) VALUES (:tenant, :userId, :value, :schoolYear)`, dbEnrolments)
	return
}

func (backend *SQLBackend) createExternalIdentifiers(tx *sqlx.Tx, tenant string, user *ss12000v1.User) (err error) {
	if user.EgilExtension == nil || len(user.EgilExtension.ExternalIdentifiers) == 0 {
		return nil
	}
	dbExternalIdentifiers := make([]dbExternalIdentifierRow, len(user.EgilExtension.ExternalIdentifiers))

	for i := range user.EgilExtension.ExternalIdentifiers {
		globallyUniqueInt := 0
		if user.EgilExtension.ExternalIdentifiers[i].GloballyUnique {
			globallyUniqueInt = 1
		}
		dbExternalIdentifiers[i] = dbExternalIdentifierRow{
			Tenant:         tenant,
			UserId:         user.ID,
			Value:          user.EgilExtension.ExternalIdentifiers[i].Value,
			Context:        user.EgilExtension.ExternalIdentifiers[i].Context,
			GloballyUnique: globallyUniqueInt,
		}
	}

	_, err = tx.NamedExec(`INSERT INTO ExternalIdentifiers (tenant, userId, value, context, globallyUnique) VALUES (:tenant, :userId, :value, :context, :globallyUnique)`, dbExternalIdentifiers)
	return
}

func (backend *SQLBackend) userCreator(tx *sqlx.Tx, tenant string, user *ss12000v1.User) (id string, err error) {
	dbUser := NewUserRow(tenant, user)

	_, err = tx.NamedExec(`INSERT INTO Users (tenant, id, userName, familyName, givenName, displayName) VALUES (:tenant, :id, :userName, :familyName, :givenName, :displayName)`, &dbUser)
	if err != nil {
		return "", err
	}

	err = backend.createEmails(tx, tenant, user)
	if err != nil {
		return "", err
	}
	err = backend.createEnrolments(tx, tenant, user)
	if err != nil {
		return "", err
	}
	err = backend.createExternalIdentifiers(tx, tenant, user)
	return user.ID, err
}

func (backend *SQLBackend) userMutator(tx *sqlx.Tx, tenant string, user *ss12000v1.User) (err error) {
	dbUser := NewUserRow(tenant, user)

	_, err = tx.NamedExec(`UPDATE Users SET userName = :userName, familyName = :familyName, givenName = :givenName, displayName = :displayName WHERE tenant = :tenant AND id = :id`, &dbUser)
	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM Emails WHERE tenant = :tenant AND userId = :userId`,
		map[string]interface{}{
			"tenant": tenant,
			"userId": user.ID,
		})

	if err != nil {
		return err
	}

	err = backend.createEmails(tx, tenant, user)

	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM Enrolments WHERE tenant = :tenant AND userId = :userId`,
		map[string]interface{}{
			"tenant": tenant,
			"userId": user.ID,
		})

	if err != nil {
		return err
	}

	err = backend.createEnrolments(tx, tenant, user)

	if err != nil {
		return err
	}

	_, err = tx.NamedExec(`DELETE FROM ExternalIdentifiers WHERE tenant = :tenant AND userId = :userId`,
		map[string]interface{}{
			"tenant": tenant,
			"userId": user.ID,
		})

	if err != nil {
		return err
	}

	return backend.createExternalIdentifiers(tx, tenant, user)
}

func (backend *SQLBackend) userReader(tx *sqlx.Tx, mainQuery, emailQuery, enrolmentQuery, externalIdentifiersQuery string, args map[string]interface{}) ([]ss12000v1.Object, error) {
	mainNamed, err := tx.PrepareNamed(mainQuery)
	if err != nil {
		return nil, err
	}
	emailNamed, err := tx.PrepareNamed(emailQuery)
	if err != nil {
		return nil, err
	}

	enrolmentNamed, err := tx.PrepareNamed(enrolmentQuery)
	if err != nil {
		return nil, err
	}

	externalIdentifiersNamed, err := tx.PrepareNamed(externalIdentifiersQuery)
	if err != nil {
		return nil, err
	}

	dbUsers := []dbUserRow{}
	err = mainNamed.Select(&dbUsers, args)
	if err != nil {
		return nil, err
	}

	users := make([]ss12000v1.Object, len(dbUsers))
	index := make(map[string]int)
	for i := range dbUsers {
		users[i] = &ss12000v1.User{
			ID:       dbUsers[i].Id,
			UserName: dbUsers[i].UserName,
			Name: ss12000v1.SCIMName{
				FamilyName: dbUsers[i].FamilyName,
				GivenName:  dbUsers[i].GivenName,
			},
			DisplayName: dbUsers[i].DisplayName,
		}
		index[dbUsers[i].Id] = i
	}

	dbEmails := []dbEmailRow{}
	err = emailNamed.Select(&dbEmails, args)
	if err != nil {
		return nil, err
	}

	for i := range dbEmails {
		email := &dbEmails[i]
		user := users[index[email.UserId]].(*ss12000v1.User)
		t := ""
		if email.Type != nil {
			t = *email.Type
		}
		user.Emails = append(user.Emails, ss12000v1.SCIMEmail{
			Value: email.Value,
			Type:  t,
		})
	}

	dbEnrolments := []dbEnrolmentRow{}
	err = enrolmentNamed.Select(&dbEnrolments, args)
	if err != nil {
		return nil, err
	}

	for i := range dbEnrolments {
		enrolment := &dbEnrolments[i]
		user := users[index[enrolment.UserId]].(*ss12000v1.User)
		user.Extension.Enrolments = append(user.Extension.Enrolments,
			ss12000v1.Enrolment{
				Value:      enrolment.Value,
				SchoolYear: enrolment.SchoolYear,
			})
	}

	dbExternalIdentifiers := []dbExternalIdentifierRow{}
	err = externalIdentifiersNamed.Select(&dbExternalIdentifiers, args)
	if err != nil {
		return nil, err
	}

	for i := range dbExternalIdentifiers {
		externalIdentifier := &dbExternalIdentifiers[i]
		user := users[index[externalIdentifier.UserId]].(*ss12000v1.User)

		if user.EgilExtension == nil {
			user.EgilExtension = &ss12000v1.EgilUserExtension{}
		}
		var ei ss12000v1.ExternalIdentifier
		ei.Value = externalIdentifier.Value
		ei.Context = externalIdentifier.Context
		ei.GloballyUnique = externalIdentifier.GloballyUnique != 0

		user.EgilExtension.ExternalIdentifiers = append(user.EgilExtension.ExternalIdentifiers, ei)
	}

	return users, nil
}

func (backend *SQLBackend) userReaderAll(tx *sqlx.Tx, tenant string) ([]ss12000v1.Object, error) {
	return backend.userReader(tx, `SELECT * FROM Users WHERE tenant = :tenant`,
		`SELECT * FROM Emails WHERE tenant = :tenant`,
		`SELECT * FROM Enrolments WHERE tenant = :tenant`,
		`SELECT * FROM ExternalIdentifiers WHERE tenant = :tenant`,
		map[string]interface{}{
			"tenant": tenant,
		})
}

func (backend *SQLBackend) userReaderOne(tx *sqlx.Tx, tenant, id string) (ss12000v1.Object, error) {
	users, err := backend.userReader(tx, `SELECT * FROM Users WHERE tenant = :tenant AND id = :id`,
		`SELECT * FROM Emails WHERE tenant = :tenant AND userId = :id`,
		`SELECT * FROM Enrolments WHERE tenant = :tenant AND userId = :id`,
		`SELECT * FROM ExternalIdentifiers WHERE tenant = :tenant AND userId = :id`,
		map[string]interface{}{
			"tenant": tenant,
			"id":     id,
		})
	if err != nil {
		return nil, err
	} else if len(users) != 1 {
		return nil, fmt.Errorf("expected one object with id %s, found %d", id, len(users))
	}
	return users[0], nil
}
