/*
 *  This file is part of Windermere (EGIL SCIM Server).
 *
 *  Copyright (C) 2019-2024 Föreningen Sambruk
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

package ss12000v2import

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/ss12000v2"
	"github.com/google/uuid"
)

// parsePaginatedResults is used to read a paginated response for a given SS12000v2 data type.
// It will return the array of objects included in the returned page and the page token which
// should be used to request the next page.
func parsePaginatedResults[T any](r *http.Response) ([]T, *string, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	var parsedResponse struct {
		Data      []T     `json:"data"`
		PageToken *string `json:"pageToken,omitempty"`
	}
	err = json.Unmarshal(b, &parsedResponse)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse paginated response: %s", err.Error())
	}

	return parsedResponse.Data, parsedResponse.PageToken, nil
}

const MaxOrganisationPageSize = 100
const MaxPersonPageSize = 50
const MaxGroupPageSize = 50
const MaxDutyPageSize = 100
const MaxActivityPageSize = 50

// Using an SS12000v2 client, this function gets all Organisation objects of a given OrganisationType
func getAllOrganisationsOfType(
	ctx context.Context,
	client ss12000v2.ClientInterface,
	organisationType ss12000v2.OrganisationTypeEnum,
	createdAfter *time.Time,
	modifiedAfter *time.Time) ([]ss12000v2.Organisation, error) {

	var orgs = make([]ss12000v2.Organisation, 0)
	var params ss12000v2.GetOrganisationsParams
	orgType := []ss12000v2.OrganisationTypeEnum{organisationType}
	params.Type = &orgType
	params.MetaCreatedAfter = createdAfter
	params.MetaModifiedAfter = modifiedAfter
	limit := MaxOrganisationPageSize
	params.Limit = &limit

	for {
		response, err := client.GetOrganisations(ctx, &params)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get organisations: %d (%s)", response.StatusCode, response.Status)
		}
		pageOfOrgs, pageToken, err := parsePaginatedResults[ss12000v2.Organisation](response)
		if err != nil {
			return nil, err
		}

		orgs = append(orgs, pageOfOrgs...)

		if pageToken != nil {
			params.PageToken = pageToken
		} else {
			break
		}
	}

	return orgs, nil
}

// Using an SS12000v2 client, this function gets all Person objects
func getAllPersons(
	ctx context.Context,
	client ss12000v2.ClientInterface,
	createdAfter *time.Time,
	modifiedAfter *time.Time) ([]ss12000v2.Person, error) {
	var persons = make([]ss12000v2.Person, 0)
	var params ss12000v2.GetPersonsParams
	params.MetaCreatedAfter = createdAfter
	params.MetaModifiedAfter = modifiedAfter
	limit := MaxPersonPageSize
	params.Limit = &limit

	for {
		response, err := client.GetPersons(ctx, &params)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get persons: %d (%s)", response.StatusCode, response.Status)
		}
		pageOfPersons, pageToken, err := parsePaginatedResults[ss12000v2.Person](response)
		if err != nil {
			return nil, err
		}

		persons = append(persons, pageOfPersons...)

		if pageToken != nil {
			params.PageToken = pageToken
		} else {
			break
		}
	}

	return persons, nil
}

// Using an SS12000v2 client, this function gets all Group objects
func getAllGroups(
	ctx context.Context,
	client ss12000v2.ClientInterface,
	createdAfter *time.Time,
	modifiedAfter *time.Time) ([]ss12000v2.Group, error) {
	var groups = make([]ss12000v2.Group, 0)
	var params ss12000v2.GetGroupsParams
	params.MetaCreatedAfter = createdAfter
	params.MetaModifiedAfter = modifiedAfter
	limit := MaxGroupPageSize
	params.Limit = &limit

	for {
		response, err := client.GetGroups(ctx, &params)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get groups: %d (%s)", response.StatusCode, response.Status)
		}
		pageOfGroups, pageToken, err := parsePaginatedResults[ss12000v2.Group](response)
		if err != nil {
			return nil, err
		}

		groups = append(groups, pageOfGroups...)

		if pageToken != nil {
			params.PageToken = pageToken
		} else {
			break
		}
	}

	return groups, nil
}

// Using an SS12000v2 client, this function gets all Duty objects
func getAllDuties(
	ctx context.Context,
	client ss12000v2.ClientInterface,
	createdAfter *time.Time,
	modifiedAfter *time.Time) ([]ss12000v2.Duty, error) {
	var duties = make([]ss12000v2.Duty, 0)
	var params ss12000v2.GetDutiesParams
	params.MetaCreatedAfter = createdAfter
	params.MetaModifiedAfter = modifiedAfter
	limit := MaxDutyPageSize
	params.Limit = &limit

	for {
		response, err := client.GetDuties(ctx, &params)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get duties: %d (%s)", response.StatusCode, response.Status)
		}
		pageOfDuties, pageToken, err := parsePaginatedResults[ss12000v2.Duty](response)
		if err != nil {
			return nil, err
		}

		duties = append(duties, pageOfDuties...)

		if pageToken != nil {
			params.PageToken = pageToken
		} else {
			break
		}
	}

	return duties, nil
}

// Using an SS12000v2 client, this function gets all Activity objects
func getAllActivities(ctx context.Context,
	client ss12000v2.ClientInterface,
	createdAfter *time.Time,
	modifiedAfter *time.Time) ([]ss12000v2.Activity, error) {
	var activities = make([]ss12000v2.Activity, 0)
	var params ss12000v2.GetActivitiesParams
	params.MetaCreatedAfter = createdAfter
	params.MetaModifiedAfter = modifiedAfter
	limit := MaxActivityPageSize
	params.Limit = &limit

	for {
		response, err := client.GetActivities(ctx, &params)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get duties: %d (%s)", response.StatusCode, response.Status)
		}
		pageOfActivities, pageToken, err := parsePaginatedResults[ss12000v2.Activity](response)
		if err != nil {
			return nil, err
		}

		activities = append(activities, pageOfActivities...)

		if pageToken != nil {
			params.PageToken = pageToken
		} else {
			break
		}
	}

	return activities, nil
}

// FullImport will do a full import by reading from an SS12000 API with an SS12000 v2 client.
// All objects which we can access, and which are relevant to Windermere, will be imported.
// Objects which we already had, but which are not available from the SS12000 API will be
// removed from our backend.
func FullImport(ctx context.Context, logger *log.Logger, tenant string, client ss12000v2.ClientInterface, backend SS12000v1Backend, importHistory ImportHistory) error {
	logger.Printf("Starting full SS12000 import for %s\n", tenant)
	timeOfFullImportStart := time.Now()
	orgs, err := getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumHuvudman, nil, nil)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get principal organisations: %s", err.Error())
	}

	v1orgs := make([]*ss12000v1.Organisation, len(orgs))
	for i, org := range orgs {
		v1orgs[i] = organisationToV1(&org)
	}

	_, _, err = backend.ReplaceOrganisations(ctx, tenant, v1orgs, true)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace organisations: %s", err.Error())
	} else {
		created, modified := organisationTimestamps(orgs)
		err = importHistory.RecordMostRecent(created, modified, "PrincipalOrganisations")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	orgs, err = getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumSkolenhet, nil, nil)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get school units: %s", err.Error())
	}

	schoolUnits := make(map[uuid.UUID]bool)
	v1schoolUnits := make([]*ss12000v1.SchoolUnit, 0, len(orgs))
	for _, org := range orgs {
		schoolUnits[org.Id] = true
		schoolUnit, err := schoolUnitToV1(&org)
		// TODO: deal with error? but perhaps the correct way is to silently ignore those school units...
		if err == nil {
			v1schoolUnits = append(v1schoolUnits, schoolUnit)
		}
	}

	_, _, err = backend.ReplaceSchoolUnits(ctx, tenant, v1schoolUnits, true)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace school units: %s", err.Error())
	} else {
		created, modified := organisationTimestamps(orgs)
		err = importHistory.RecordMostRecent(created, modified, "SchoolUnitOrganisations")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	persons, err := getAllPersons(ctx, client, nil, nil)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get persons: %s", err.Error())
	}

	v1users := make([]*ss12000v1.User, 0, len(persons))
	for _, person := range persons {
		user, err := personToV1(&person)
		// TODO: deal with error? but perhaps the correct way is to silently ignore those persons...
		if err == nil {
			v1users = append(v1users, user)
		}
	}

	_, _, err = backend.ReplaceUsers(ctx, tenant, v1users, true)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace users: %s", err.Error())
	} else {
		created, modified := personTimestamps(persons)
		err = importHistory.RecordMostRecent(created, modified, "Persons")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	groups, err := getAllGroups(ctx, client, nil, nil)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get groups: %s", err.Error())
	}

	v1studentGroups := make([]*ss12000v1.StudentGroup, 0, len(groups))
	for _, group := range groups {
		// Skip groups not belonging to a school unit
		if _, ok := schoolUnits[group.Organisation.Id]; !ok {
			continue
		}
		studentGroup := groupToV1(&group)
		v1studentGroups = append(v1studentGroups, studentGroup)

	}

	_, _, err = backend.ReplaceStudentGroups(ctx, tenant, v1studentGroups, true)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace student groups: %s", err.Error())
	} else {
		created, modified := groupTimestamps(groups)
		err = importHistory.RecordMostRecent(created, modified, "Groups")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}

	}

	duties, err := getAllDuties(ctx, client, nil, nil)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get duties: %s", err.Error())
	}

	v1employments := make([]*ss12000v1.Employment, 0, len(duties))
	for _, duty := range duties {
		employment, err := dutyToV1(&duty)
		// TODO: deal with error? but perhaps the correct way is to silently ignore those duties...
		if err == nil {
			v1employments = append(v1employments, employment)
		}
	}

	_, _, err = backend.ReplaceEmployments(ctx, tenant, v1employments, true)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace employments: %s", err.Error())
	} else {
		created, modified := dutyTimestamps(duties)
		err = importHistory.RecordMostRecent(created, modified, "Duties")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	activities, err := getAllActivities(ctx, client, nil, nil)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get activities: %s", err.Error())
	}

	v1activities := make([]*ss12000v1.Activity, 0, len(activities))
	for _, activity := range activities {
		v1activities = append(v1activities, activityToV1(&activity))
	}

	_, _, err = backend.ReplaceActivities(ctx, tenant, v1activities, true)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace activities: %s", err.Error())
	} else {
		created, modified := activityTimestamps(activities)
		err = importHistory.RecordMostRecent(created, modified, "Activities")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	// We haven't done a deletedEntities call yet, but the first time we do an
	// incremental import we shouldn't get deleted entities from the beginning of time...
	err = importHistory.SetTimeOfLastDeletedEntitiesCall(timeOfFullImportStart)
	if err != nil {
		return fmt.Errorf("Failed to record time of last deletedEntities call: %s", err.Error())
	}
	logger.Printf("Full SS12000 import done for %s\n", tenant)
	return nil
}
