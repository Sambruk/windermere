package ss12000v2import

import (
	"context"
	"time"

	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/ss12000v2"
	"github.com/google/uuid"
)

type ImportStatistics struct {
}

type ImportEvent struct {
}

type ImportEvents []ImportEvent

// SS12000v1Backend is the interface which the SS12000v2 import writes to.
// This will be implemented by the Windermere SCIM backend so that we can
// write to the database (or in-memory backend for test purposes).
//
// The Replace* functions will create or update the objects in the list sent
// in. If deleteOthers is set to true it will also delete all objects of that
// type which the backend already had but are not included in the list sent in.
// In other words, with deleteOthers = true it's like replacing the whole list
// of objects for that data type.
type SS12000v1Backend interface {
	ReplaceOrganisations(ctx context.Context, tenant string, orgs []*ss12000v1.Organisation, deleteOthers bool) (ImportStatistics, ImportEvents, error)
	ReplaceStudentGroups(ctx context.Context, tenant string, groups []*ss12000v1.StudentGroup, deleteOthers bool) (ImportStatistics, ImportEvents, error)
	ReplaceUsers(ctx context.Context, tenant string, users []*ss12000v1.User, deleteOthers bool) (ImportStatistics, ImportEvents, error)
	ReplaceActivities(ctx context.Context, tenant string, activities []*ss12000v1.Activity, deleteOthers bool) (ImportStatistics, ImportEvents, error)
	ReplaceEmployments(ctx context.Context, tenant string, employments []*ss12000v1.Employment, deleteOthers bool) (ImportStatistics, ImportEvents, error)
	ReplaceSchoolUnits(ctx context.Context, tenant string, schoolUnits []*ss12000v1.SchoolUnit, deleteOthers bool) (ImportStatistics, ImportEvents, error)
	ReplaceSchoolUnitGroups(ctx context.Context, tenant string, schoolUnitGroups []*ss12000v1.SchoolUnitGroup, deleteOthers bool) (ImportStatistics, ImportEvents, error)

	// When we get deletedEntities from the SS12000 API we won't know if the objects were corresponding to
	// SS12000:2018 Organisations or SchoolUnits, so we have one delete function for both which will need
	// to figure out what the ids correspond to in the backend.
	DeleteOrganisationsOrSchoolUnits(ctx context.Context, tenant string, objs []uuid.UUID) (ImportStatistics, ImportEvents, error)
	DeleteStudentGroups(ctx context.Context, tenant string, objs []uuid.UUID) (ImportStatistics, ImportEvents, error)
	DeleteUsers(ctx context.Context, tenant string, objs []uuid.UUID) (ImportStatistics, ImportEvents, error)
	DeleteActivities(ctx context.Context, tenant string, objs []uuid.UUID) (ImportStatistics, ImportEvents, error)
	DeleteEmployments(ctx context.Context, tenant string, objs []uuid.UUID) (ImportStatistics, ImportEvents, error)
}

func organisationTimestamps(orgs []ss12000v2.Organisation) (created []time.Time, modified []time.Time) {
	created = make([]time.Time, len(orgs))
	modified = make([]time.Time, len(orgs))
	for i := range orgs {
		if orgs[i].Meta == nil {
			created[i] = time.Now()
			modified[i] = time.Now()
		} else {
			created[i] = orgs[i].Meta.Created
			modified[i] = orgs[i].Meta.Modified
		}
	}
	return
}

func personTimestamps(persons []ss12000v2.Person) (created []time.Time, modified []time.Time) {
	created = make([]time.Time, len(persons))
	modified = make([]time.Time, len(persons))
	for i := range persons {
		if persons[i].Meta == nil {
			created[i] = time.Now()
			modified[i] = time.Now()
		} else {
			created[i] = persons[i].Meta.Created
			modified[i] = persons[i].Meta.Modified
		}
	}
	return
}

func groupTimestamps(groups []ss12000v2.Group) (created []time.Time, modified []time.Time) {
	created = make([]time.Time, len(groups))
	modified = make([]time.Time, len(groups))
	for i := range groups {
		if groups[i].Meta == nil {
			created[i] = time.Now()
			modified[i] = time.Now()
		} else {
			created[i] = groups[i].Meta.Created
			modified[i] = groups[i].Meta.Modified
		}
	}
	return
}

func dutyTimestamps(duties []ss12000v2.Duty) (created []time.Time, modified []time.Time) {
	created = make([]time.Time, len(duties))
	modified = make([]time.Time, len(duties))
	for i := range duties {
		if duties[i].Meta == nil {
			created[i] = time.Now()
			modified[i] = time.Now()
		} else {
			created[i] = duties[i].Meta.Created
			modified[i] = duties[i].Meta.Modified
		}
	}
	return
}

func activityTimestamps(activities []ss12000v2.Activity) (created []time.Time, modified []time.Time) {
	created = make([]time.Time, len(activities))
	modified = make([]time.Time, len(activities))
	for i := range activities {
		if activities[i].Meta == nil {
			created[i] = time.Now()
			modified[i] = time.Now()
		} else {
			created[i] = activities[i].Meta.Created
			modified[i] = activities[i].Meta.Modified
		}
	}
	return
}
