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

func appendUnique[T any, U comparable](list1, list2 []T, getId func(*T) U) []T {
	result := make([]T, 0)
	ids := make(map[U]bool)

	longList := append(list1, list2...)

	for _, obj := range longList {
		id := getId(&obj)
		if _, ok := ids[id]; ok {
			continue
		}
		result = append(result, obj)
		ids[id] = true
	}

	return result
}

func orgId(org *ss12000v2.Organisation) uuid.UUID {
	return org.Id
}

func personId(person *ss12000v2.Person) uuid.UUID {
	return person.Id
}

func groupId(group *ss12000v2.Group) uuid.UUID {
	return group.Id
}

func dutyId(duty *ss12000v2.Duty) uuid.UUID {
	return duty.Id
}

func activityId(activity *ss12000v2.Activity) uuid.UUID {
	return activity.Id
}

// parseDeletedEntitiesResults is used to read a paginated response for deletedEntities
// It will return one page of objects and the page token which
// should be used to request the next page.
func parseDeletedEntitiesResults(r *http.Response) (*ss12000v2.DeletedEntitiesData, *string, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	var parsedResponse struct {
		Data      ss12000v2.DeletedEntitiesData `json:"data"`
		PageToken *string                       `json:"pageToken,omitempty"`
	}
	err = json.Unmarshal(b, &parsedResponse)

	if err != nil {
		log.Printf("%s\n", string(b))
		return nil, nil, fmt.Errorf("failed to parse deleted entities response: %s", err.Error())
	}

	return &parsedResponse.Data, parsedResponse.PageToken, nil
}

// Basically the same as ss12000v2.DeletedEntitiesData but only for the types
// we care about and with maps for easy lookup.
type DeletedEntities struct {
	Organisations map[uuid.UUID]bool
	Persons       map[uuid.UUID]bool
	Groups        map[uuid.UUID]bool
	Duties        map[uuid.UUID]bool
	Activities    map[uuid.UUID]bool
}

const MaxDeletedEntitiesPageSize = 200

// Gets all deleted objects (of all types we care about) since a given time.
func getAllDeletedEntities(ctx context.Context, client ss12000v2.ClientInterface, after time.Time) (DeletedEntities, error) {
	var result DeletedEntities
	result.Organisations = make(map[uuid.UUID]bool)
	result.Persons = make(map[uuid.UUID]bool)
	result.Groups = make(map[uuid.UUID]bool)
	result.Duties = make(map[uuid.UUID]bool)
	result.Activities = make(map[uuid.UUID]bool)

	var params ss12000v2.GetDeletedEntitiesParams
	params.After = &after
	types := []ss12000v2.EndPointsEnum{
		ss12000v2.EndPointsEnumOrganisation,
		ss12000v2.EndPointsEnumPerson,
		ss12000v2.EndPointsEnumGroup,
		ss12000v2.EndPointsEnumDuty,
		ss12000v2.EndPointsEnumActivity,
	}
	params.Entities = &types
	limit := MaxDeletedEntitiesPageSize
	params.Limit = &limit

	for {
		response, err := client.GetDeletedEntities(ctx, &params)
		if err != nil {
			return DeletedEntities{}, err
		}
		if response.StatusCode != http.StatusOK {
			return DeletedEntities{}, fmt.Errorf("failed to get deleted entities: %d (%s)", response.StatusCode, response.Status)
		}
		page, pageToken, err := parseDeletedEntitiesResults(response)
		if err != nil {
			return DeletedEntities{}, err
		}

		addIdsToMap := func(m map[uuid.UUID]bool, l *[]uuid.UUID) {
			if l != nil {
				for _, id := range *l {
					m[id] = true
				}
			}
		}

		addIdsToMap(result.Organisations, page.Organisations)
		addIdsToMap(result.Persons, page.Persons)
		addIdsToMap(result.Groups, page.Groups)
		addIdsToMap(result.Duties, page.Duties)
		addIdsToMap(result.Activities, page.Activitites)

		if pageToken != nil {
			params.PageToken = pageToken
		} else {
			break
		}
	}
	return result, nil
}

// Given an array of objects, and a function for getting the objects' ids, remove all
// ids from a map.
func removeIdsFromMap[T any, U comparable](m map[U]bool, objs []T, getId func(*T) U) {
	for i := range objs {
		delete(m, getId(&objs[i]))
	}
}

// Get all keys from a map
func keysFromMap[T comparable, U any](m map[T]U) []T {
	keys := make([]T, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func IncrementalImport(ctx context.Context, logger *log.Logger, tenant string, client ss12000v2.ClientInterface, backend SS12000v1Backend, importHistory ImportHistory) error {
	logger.Printf("Starting incremental SS12000 import for %s\n", tenant)

	timeOfDeletedEntitiesCall := time.Now()
	timeOfLastDeletedEntitiesCall, err := importHistory.GetTimeOfLastDeletedEntitiesCall()
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	deletedEntities, err := getAllDeletedEntities(ctx, client, timeOfLastDeletedEntitiesCall)
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get deleted entities: %s", err.Error())
	}

	// Principal organisations (huvudmän)
	createdAfter, err := importHistory.GetMostRecentlyCreated("PrincipalOrganisations")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	orgsCreatedAfter, err := getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumHuvudman, &createdAfter, nil)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently created principal organisations: %s", err.Error())
	}

	modifiedAfter, err := importHistory.GetMostRecentlyModified("PrincipalOrganisations")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	orgsModifiedAfter, err := getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumHuvudman, nil, &modifiedAfter)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently modified principal organisations: %s", err.Error())
	}

	orgs := appendUnique(orgsCreatedAfter, orgsModifiedAfter, orgId)
	removeIdsFromMap(deletedEntities.Organisations, orgs, orgId)

	v1orgs := make([]*ss12000v1.Organisation, len(orgs))
	for i, org := range orgs {
		v1orgs[i] = organisationToV1(&org)
	}

	_, _, err = backend.ReplaceOrganisations(ctx, tenant, v1orgs, false)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to incrementally replace organisations: %s", err.Error())
	} else {
		created, modified := organisationTimestamps(orgs)
		err = importHistory.RecordMostRecent(created, modified, "PrincipalOrganisations")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	// SchoolUnit organisations
	createdAfter, err = importHistory.GetMostRecentlyCreated("SchoolUnitOrganisations")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	modifiedAfter, err = importHistory.GetMostRecentlyModified("SchoolUnitOrganisations")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	allSchoolUnits, err := getAllOrganisationsOfType(ctx, client, ss12000v2.OrganisationTypeEnumSkolenhet, nil, nil)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get school units: %s", err.Error())
	}

	orgs = make([]ss12000v2.Organisation, 0)
	schoolUnits := make(map[uuid.UUID]bool)

	for _, su := range allSchoolUnits {
		if su.Meta.Created.After(createdAfter) || su.Meta.Modified.After(modifiedAfter) {
			orgs = append(orgs, su)
		}
		schoolUnits[su.Id] = true
	}

	removeIdsFromMap(deletedEntities.Organisations, orgs, orgId)

	v1schoolUnits := make([]*ss12000v1.SchoolUnit, 0, len(orgs))
	for _, org := range orgs {
		schoolUnit, err := schoolUnitToV1(&org)
		// TODO: deal with error? but perhaps the correct way is to silently ignore those school units...
		if err == nil {
			v1schoolUnits = append(v1schoolUnits, schoolUnit)
		}
	}

	_, _, err = backend.ReplaceSchoolUnits(ctx, tenant, v1schoolUnits, false)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to incrementally replace school units: %s", err.Error())
	} else {
		created, modified := organisationTimestamps(orgs)
		err = importHistory.RecordMostRecent(created, modified, "SchoolUnitOrganisations")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	// Persons
	createdAfter, err = importHistory.GetMostRecentlyCreated("Persons")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	personsCreatedAfter, err := getAllPersons(ctx, client, &createdAfter, nil)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently created persons: %s", err.Error())
	}

	modifiedAfter, err = importHistory.GetMostRecentlyModified("Persons")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	personsModifiedAfter, err := getAllPersons(ctx, client, nil, &modifiedAfter)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently modified persons: %s", err.Error())
	}

	persons := appendUnique(personsCreatedAfter, personsModifiedAfter, personId)
	removeIdsFromMap(deletedEntities.Persons, persons, personId)

	v1users := make([]*ss12000v1.User, 0, len(persons))
	for _, person := range persons {
		user, err := personToV1(&person)
		// TODO: deal with error? but perhaps the correct way is to silently ignore those persons...
		if err == nil {
			v1users = append(v1users, user)
		}
	}

	_, _, err = backend.ReplaceUsers(ctx, tenant, v1users, false)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to incrementally replace persons: %s", err.Error())
	} else {
		created, modified := personTimestamps(persons)
		err = importHistory.RecordMostRecent(created, modified, "Persons")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	// Groups
	createdAfter, err = importHistory.GetMostRecentlyCreated("Groups")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	groupsCreatedAfter, err := getAllGroups(ctx, client, &createdAfter, nil)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently created groups: %s", err.Error())
	}

	modifiedAfter, err = importHistory.GetMostRecentlyModified("Groups")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	groupsModifiedAfter, err := getAllGroups(ctx, client, nil, &modifiedAfter)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently modified groups: %s", err.Error())
	}

	groups := appendUnique(groupsCreatedAfter, groupsModifiedAfter, groupId)
	removeIdsFromMap(deletedEntities.Groups, groups, groupId)

	v1studentGroups := make([]*ss12000v1.StudentGroup, 0, len(groups))
	for _, group := range groups {
		// Skip groups not belonging to a school unit
		// TODO: what if we already had the group?
		if _, ok := schoolUnits[group.Organisation.Id]; !ok {
			continue
		}

		studentGroup := groupToV1(&group)
		v1studentGroups = append(v1studentGroups, studentGroup)
	}

	_, _, err = backend.ReplaceStudentGroups(ctx, tenant, v1studentGroups, false)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to incrementally replace student groups: %s", err.Error())
	} else {
		created, modified := groupTimestamps(groups)
		err = importHistory.RecordMostRecent(created, modified, "Groups")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	// Duties
	createdAfter, err = importHistory.GetMostRecentlyCreated("Duties")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	dutiesCreatedAfter, err := getAllDuties(ctx, client, &createdAfter, nil)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently created duties: %s", err.Error())
	}

	modifiedAfter, err = importHistory.GetMostRecentlyModified("Duties")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	dutiesModifiedAfter, err := getAllDuties(ctx, client, nil, &modifiedAfter)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently modified duties: %s", err.Error())
	}

	duties := appendUnique(dutiesCreatedAfter, dutiesModifiedAfter, dutyId)
	removeIdsFromMap(deletedEntities.Duties, duties, dutyId)

	v1employments := make([]*ss12000v1.Employment, 0, len(duties))
	for _, duty := range duties {
		employment, err := dutyToV1(&duty)
		// TODO: deal with error? but perhaps the correct way is to silently ignore those duties...
		if err == nil {
			v1employments = append(v1employments, employment)
		}
	}

	_, _, err = backend.ReplaceEmployments(ctx, tenant, v1employments, false)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to incrementally replace employments: %s", err.Error())
	} else {
		created, modified := dutyTimestamps(duties)
		err = importHistory.RecordMostRecent(created, modified, "Duties")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}

	}

	// Activities
	createdAfter, err = importHistory.GetMostRecentlyCreated("Activities")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	activitiesCreatedAfter, err := getAllActivities(ctx, client, &createdAfter, nil)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently created activities: %s", err.Error())
	}

	modifiedAfter, err = importHistory.GetMostRecentlyModified("Activities")
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to read history: %s", err.Error())
	}

	activitiesModifiedAfter, err := getAllActivities(ctx, client, nil, &modifiedAfter)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to get recently modified activities: %s", err.Error())
	}

	activities := appendUnique(activitiesCreatedAfter, activitiesModifiedAfter, activityId)
	removeIdsFromMap(deletedEntities.Activities, activities, activityId)

	v1activities := make([]*ss12000v1.Activity, 0, len(activities))
	for _, activity := range activities {
		activity := activityToV1(&activity)
		v1activities = append(v1activities, activity)
	}

	_, _, err = backend.ReplaceActivities(ctx, tenant, v1activities, false)

	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to incrementally replace activities: %s", err.Error())
	} else {
		created, modified := activityTimestamps(activities)
		err = importHistory.RecordMostRecent(created, modified, "Activities")
		if err != nil {
			return fmt.Errorf("Failed to record most recent created/modified: %s", err.Error())
		}
	}

	// Delete all deleted entities
	_, _, err = backend.DeleteOrganisationsOrSchoolUnits(ctx, tenant, keysFromMap(deletedEntities.Organisations))
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to delete organisations: %s", err.Error())
	}

	_, _, err = backend.DeleteStudentGroups(ctx, tenant, keysFromMap(deletedEntities.Groups))
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to delete student groups: %s", err.Error())
	}

	_, _, err = backend.DeleteUsers(ctx, tenant, keysFromMap(deletedEntities.Persons))
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to delete users: %s", err.Error())
	}

	_, _, err = backend.DeleteActivities(ctx, tenant, keysFromMap(deletedEntities.Activities))
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to delete activities: %s", err.Error())
	}

	_, _, err = backend.DeleteEmployments(ctx, tenant, keysFromMap(deletedEntities.Duties))
	if err != nil {
		return fmt.Errorf("IncrementalImport: failed to delete employments: %s", err.Error())
	}

	// All done deleting for all types, now we can record when we last called deletedEntities
	err = importHistory.SetTimeOfLastDeletedEntitiesCall(timeOfDeletedEntitiesCall)

	if err != nil {
		return fmt.Errorf("Failed to record time of last deletedEntities call: %s", err.Error())
	}

	logger.Printf("Incremental SS12000 import done for %s\n", tenant)
	return nil
}
