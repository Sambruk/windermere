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

	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/ss12000v2"
	"github.com/google/uuid"
)

// SS12000v1Backend is the interface which the SS12000v2 import writes to.
// This will be implemented by the Windermere SCIM backend so that we can
// write to the database (or in-memory backend for test purposes).
//
// The ReplaceAll* functions will ensure that objects which the backend already
// had are deleted if they don't exist in the new list of objects. In other words
// it's like replacing the whole list of objects for that data type.
type SS12000v1Backend interface {
	ReplaceAllOrganisations(ctx context.Context, tenant string, orgs []*ss12000v1.Organisation) error
	ReplaceAllStudentGroups(ctx context.Context, tenant string, groups []*ss12000v1.StudentGroup) error
	ReplaceAllUsers(ctx context.Context, tenant string, users []*ss12000v1.User) error
	ReplaceAllActivities(ctx context.Context, tenant string, activities []*ss12000v1.Activity) error
	ReplaceAllEmployments(ctx context.Context, tenant string, employments []*ss12000v1.Employment) error
	ReplaceAllSchoolUnits(ctx context.Context, tenant string, schoolUnits []*ss12000v1.SchoolUnit) error
	ReplaceAllSchoolUnitGroups(ctx context.Context, tenant string, schoolUnitGroups []*ss12000v1.SchoolUnitGroup) error
}

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
		log.Printf("%s\n", string(b))
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
func getAllOrganisationsOfType(ctx context.Context, client ss12000v2.ClientInterface, organisationType ss12000v2.OrganisationTypeEnum) ([]ss12000v2.Organisation, error) {
	var orgs = make([]ss12000v2.Organisation, 0)
	var params ss12000v2.GetOrganisationsParams
	orgType := []ss12000v2.OrganisationTypeEnum{organisationType}
	params.Type = &orgType
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
func getAllPersons(ctx context.Context, client ss12000v2.ClientInterface) ([]ss12000v2.Person, error) {
	var persons = make([]ss12000v2.Person, 0)
	var params ss12000v2.GetPersonsParams
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
func getAllGroups(ctx context.Context, client ss12000v2.ClientInterface) ([]ss12000v2.Group, error) {
	var groups = make([]ss12000v2.Group, 0)
	var params ss12000v2.GetGroupsParams
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
func getAllDuties(ctx context.Context, client ss12000v2.ClientInterface) ([]ss12000v2.Duty, error) {
	var duties = make([]ss12000v2.Duty, 0)
	var params ss12000v2.GetDutiesParams
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
func getAllActivities(ctx context.Context, client ss12000v2.ClientInterface) ([]ss12000v2.Activity, error) {
	var activities = make([]ss12000v2.Activity, 0)
	var params ss12000v2.GetActivitiesParams
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
func FullImport(ctx context.Context, tenant string, client ss12000v2.ClientInterface, backend SS12000v1Backend) error {
	log.Printf("Starting full SS12000 import for %s\n", tenant)
	orgs, err := getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumHuvudman)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get principal organisations: %s", err.Error())
	}

	v1orgs := make([]*ss12000v1.Organisation, len(orgs))
	for i, org := range orgs {
		v1orgs[i] = organisationToV1(&org)
	}

	err = backend.ReplaceAllOrganisations(ctx, tenant, v1orgs)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace organisations: %s", err.Error())
	}

	orgs, err = getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumSkolenhet)

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

	err = backend.ReplaceAllSchoolUnits(ctx, tenant, v1schoolUnits)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace school units: %s", err.Error())
	}

	persons, err := getAllPersons(ctx, client)

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

	err = backend.ReplaceAllUsers(ctx, tenant, v1users)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace users: %s", err.Error())
	}

	groups, err := getAllGroups(ctx, client)

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

	err = backend.ReplaceAllStudentGroups(ctx, tenant, v1studentGroups)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace student groups: %s", err.Error())
	}

	duties, err := getAllDuties(ctx, client)

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

	err = backend.ReplaceAllEmployments(ctx, tenant, v1employments)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace employments: %s", err.Error())
	}

	activities, err := getAllActivities(ctx, client)

	if err != nil {
		return fmt.Errorf("FullImport: failed to get activities: %s", err.Error())
	}

	v1activities := make([]*ss12000v1.Activity, 0, len(activities))
	for _, activity := range activities {
		v1activities = append(v1activities, activityToV1(&activity))
	}

	err = backend.ReplaceAllActivities(ctx, tenant, v1activities)

	if err != nil {
		return fmt.Errorf("FullImport: failed to replace activities: %s", err.Error())
	}
	log.Printf("Full SS12000 import done for %s\n", tenant)
	return nil
}
