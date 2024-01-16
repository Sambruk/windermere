package windermere

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sambruk/windermere/scimserverlite"
	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/ss12000v2import"
	"github.com/google/uuid"
)

type V2toV1ImportBackendAdapter struct {
	backend scimserverlite.Backend
}

func NewV2toV1ImportBackendAdapter(b scimserverlite.Backend) *V2toV1ImportBackendAdapter {
	return &V2toV1ImportBackendAdapter{
		backend: b,
	}
}

func ReplaceT[T ss12000v1.Object](ctx context.Context, adapter *V2toV1ImportBackendAdapter, tenant string, resourceType string, objs []T, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	var importStatistics ss12000v2import.ImportStatistics
	var importEvents ss12000v2import.ImportEvents

	existingResources, err := adapter.backend.GetResources(tenant, resourceType)

	if err != nil {
		return importStatistics, importEvents, fmt.Errorf("V2toV1ImportBackendAdapter: failed to replace %s objects: %s", resourceType, err.Error())
	}

	allNewIds := make(map[string]bool)
	operations := make([]scimserverlite.BulkOperation, 0)

	for _, obj := range objs {
		id := obj.GetID()
		allNewIds[id] = true
		buff, err := json.Marshal(obj)
		serializedObj := string(buff)

		if err != nil {
			return importStatistics, importEvents, fmt.Errorf("V2toV1ImportBackendAdapter: failed to marshal object (type=%s) with id %s: %s", resourceType, id, err.Error())
		}

		if _, ok := existingResources[id]; ok {
			operations = append(operations, scimserverlite.NewBulkUpdateOperation(resourceType, id, serializedObj))
		} else {
			operations = append(operations, scimserverlite.NewBulkCreateOperation(resourceType, serializedObj))
		}
	}

	if deleteOthers {
		for key := range existingResources {
			if _, ok := allNewIds[key]; !ok {
				operations = append(operations, scimserverlite.NewBulkDeleteOperation(resourceType, key))
			}
		}
	}

	_, err = adapter.backend.Bulk(ctx, tenant, operations)

	// TODO: Create proper ImportStatistics and ImportEvents from bulk results
	/*	for _, result := range results {
		}*/

	if err != nil {
		return importStatistics, importEvents, fmt.Errorf("Failed to perform bulk import of %s: %s", resourceType, err.Error())
	}

	return importStatistics, importEvents, nil
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceOrganisations(ctx context.Context, tenant string, orgs []*ss12000v1.Organisation, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "Organisations", orgs, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceStudentGroups(ctx context.Context, tenant string, groups []*ss12000v1.StudentGroup, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "StudentGroups", groups, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceUsers(ctx context.Context, tenant string, users []*ss12000v1.User, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "Users", users, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceActivities(ctx context.Context, tenant string, activities []*ss12000v1.Activity, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "Activities", activities, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceEmployments(ctx context.Context, tenant string, employments []*ss12000v1.Employment, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "Employments", employments, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceSchoolUnits(ctx context.Context, tenant string, schoolUnits []*ss12000v1.SchoolUnit, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "SchoolUnits", schoolUnits, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceSchoolUnitGroups(ctx context.Context, tenant string, schoolUnitGroups []*ss12000v1.SchoolUnitGroup, deleteOthers bool) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return ReplaceT(ctx, adapter, tenant, "SchoolUnitGroups", schoolUnitGroups, deleteOthers)
}

func (adapter *V2toV1ImportBackendAdapter) Delete(ctx context.Context, tenant string, resourceTypes []string, objs []uuid.UUID) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	var importStatistics ss12000v2import.ImportStatistics
	var importEvents ss12000v2import.ImportEvents

	operations := make([]scimserverlite.BulkOperation, 0)

	for _, resourceType := range resourceTypes {
		existingResources, err := adapter.backend.GetResources(tenant, resourceType)

		if err != nil {
			return importStatistics, importEvents, fmt.Errorf("V2toV1ImportBackendAdapter: failed to delete %s objects: %s", resourceType, err.Error())
		}

		for _, id := range objs {
			if _, ok := existingResources[id.String()]; ok {
				operations = append(operations, scimserverlite.NewBulkDeleteOperation(resourceType, id.String()))
			}
		}
	}

	_, err := adapter.backend.Bulk(ctx, tenant, operations)

	// TODO: Create proper ImportStatistics and ImportEvents from bulk results
	/*	for _, result := range results {
		}*/

	if err != nil {
		return importStatistics, importEvents, fmt.Errorf("Failed to perform bulk delete of %v: %s", resourceTypes, err.Error())
	}

	return importStatistics, importEvents, nil
}

func (adapter *V2toV1ImportBackendAdapter) DeleteOrganisationsOrSchoolUnits(ctx context.Context, tenant string, objs []uuid.UUID) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return adapter.Delete(ctx, tenant, []string{"SchoolUnits", "Organisations"}, objs)
}

func (adapter *V2toV1ImportBackendAdapter) DeleteStudentGroups(ctx context.Context, tenant string, objs []uuid.UUID) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return adapter.Delete(ctx, tenant, []string{"StudentGroups"}, objs)
}

func (adapter *V2toV1ImportBackendAdapter) DeleteUsers(ctx context.Context, tenant string, objs []uuid.UUID) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return adapter.Delete(ctx, tenant, []string{"Users"}, objs)
}

func (adapter *V2toV1ImportBackendAdapter) DeleteActivities(ctx context.Context, tenant string, objs []uuid.UUID) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return adapter.Delete(ctx, tenant, []string{"Activities"}, objs)
}

func (adapter *V2toV1ImportBackendAdapter) DeleteEmployments(ctx context.Context, tenant string, objs []uuid.UUID) (ss12000v2import.ImportStatistics, ss12000v2import.ImportEvents, error) {
	return adapter.Delete(ctx, tenant, []string{"Employments"}, objs)
}
