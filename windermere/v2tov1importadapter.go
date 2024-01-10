package windermere

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sambruk/windermere/scimserverlite"
	"github.com/Sambruk/windermere/ss12000v1"
)

type V2toV1ImportBackendAdapter struct {
	backend scimserverlite.Backend
}

func NewV2toV1ImportBackendAdapter(b scimserverlite.Backend) *V2toV1ImportBackendAdapter {
	return &V2toV1ImportBackendAdapter{
		backend: b,
	}
}

func ReplaceAllT[T ss12000v1.Object](ctx context.Context, adapter *V2toV1ImportBackendAdapter, tenant string, resourceType string, objs []T) error {
	existingResources, err := adapter.backend.GetResources(tenant, resourceType)

	if err != nil {
		return fmt.Errorf("V2toV1ImportBackendAdapter: failed to replace %s objects: %s", resourceType, err.Error())
	}

	allNewIds := make(map[string]bool)
	operations := make([]scimserverlite.BulkOperation, 0)

	for _, obj := range objs {
		id := obj.GetID()
		allNewIds[id] = true
		buff, err := json.Marshal(obj)
		serializedObj := string(buff)

		if err != nil {
			return fmt.Errorf("V2toV1ImportBackendAdapter: failed to marshal object (type=%s) with id %s: %s", resourceType, id, err.Error())
		}

		if _, ok := existingResources[id]; ok {
			operations = append(operations, scimserverlite.NewBulkUpdateOperation(resourceType, id, serializedObj))
		} else {
			operations = append(operations, scimserverlite.NewBulkCreateOperation(resourceType, serializedObj))
		}
	}

	for key := range existingResources {
		if _, ok := allNewIds[key]; !ok {
			operations = append(operations, scimserverlite.NewBulkDeleteOperation(resourceType, key))
		}
	}

	results, err := adapter.backend.Bulk(ctx, tenant, operations)

	if err != nil {
		return fmt.Errorf("Failed to perform bulk import of %s: %s", resourceType, err.Error())
	}

	// TODO: Better error handling so we can see more/all errors?
	for _, result := range results {
		if result.Error != nil {
			return fmt.Errorf("At least one object failed to import: %s %s / %s : %s", result.Type.ToString(), result.ResourceType, result.ResourceID, result.Error.Error())
		}
	}
	return nil
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllOrganisations(ctx context.Context, tenant string, orgs []*ss12000v1.Organisation) error {
	return ReplaceAllT(ctx, adapter, tenant, "Organisations", orgs)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllStudentGroups(ctx context.Context, tenant string, groups []*ss12000v1.StudentGroup) error {
	return ReplaceAllT(ctx, adapter, tenant, "StudentGroups", groups)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllUsers(ctx context.Context, tenant string, users []*ss12000v1.User) error {
	return ReplaceAllT(ctx, adapter, tenant, "Users", users)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllActivities(ctx context.Context, tenant string, activities []*ss12000v1.Activity) error {
	return ReplaceAllT(ctx, adapter, tenant, "Activities", activities)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllEmployments(ctx context.Context, tenant string, employments []*ss12000v1.Employment) error {
	return ReplaceAllT(ctx, adapter, tenant, "Employments", employments)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllSchoolUnits(ctx context.Context, tenant string, schoolUnits []*ss12000v1.SchoolUnit) error {
	return ReplaceAllT(ctx, adapter, tenant, "SchoolUnits", schoolUnits)
}

func (adapter *V2toV1ImportBackendAdapter) ReplaceAllSchoolUnitGroups(ctx context.Context, tenant string, schoolUnitGroups []*ss12000v1.SchoolUnitGroup) error {
	return ReplaceAllT(ctx, adapter, tenant, "SchoolUnitGroups", schoolUnitGroups)
}
