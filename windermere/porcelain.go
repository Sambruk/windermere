package windermere

import (
	"encoding/json"
	"fmt"

	"github.com/Sambruk/windermere/ss12000v1"
)

// GetSchoolUnits returns all school units
func (w *Windermere) GetSchoolUnits(tenant string) ([]ss12000v1.SchoolUnit, error) {
	objectMap, err := w.GetParsedResources(tenant, "SchoolUnits")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.SchoolUnit, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.SchoolUnit))
	}
	return result, nil
}

// GetUsers returns all users
func (w *Windermere) GetUsers(tenant string) ([]ss12000v1.User, error) {
	objectMap, err := w.GetParsedResources(tenant, "Users")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.User, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.User))
	}
	return result, nil
}

// GetEmploymentsAt returns all employment objects for a specific organisation
func (w *Windermere) GetEmploymentsAt(tenant, organisation string) ([]ss12000v1.Employment, error) {
	objectMap, err := w.GetParsedResources(tenant, "Employments")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.Employment, 0, len(objectMap))

	for _, object := range objectMap {
		employment := object.(*ss12000v1.Employment)
		if employment.EmployedAt.Value == organisation {
			result = append(result, *employment)
		}
	}
	return result, nil
}

// GetEmployments returns all employment objects
func (w *Windermere) GetEmployments(tenant string) ([]ss12000v1.Employment, error) {
	objectMap, err := w.GetParsedResources(tenant, "Employments")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.Employment, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.Employment))
	}
	return result, nil
}

// GetUser returns a user with a specific id (if it exists)
func (w *Windermere) GetUser(tenant, id string) (*ss12000v1.User, error) {
	parsedObject, err := w.GetParsedResource(tenant, "Users", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.User), nil
}

// GetStudentGroups returns all groups
func (w *Windermere) GetStudentGroups(tenant string) ([]ss12000v1.StudentGroup, error) {
	objectMap, err := w.GetParsedResources(tenant, "StudentGroups")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.StudentGroup, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.StudentGroup))
	}
	return result, nil
}

// GetStudentGroup returns a student group with a specific id (if it exists)
func (w *Windermere) GetStudentGroup(tenant, id string) (*ss12000v1.StudentGroup, error) {
	parsedObject, err := w.GetParsedResource(tenant, "StudentGroups", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.StudentGroup), nil
}

// GetSchoolUnitGroups returns all school unit groups
func (w *Windermere) GetSchoolUnitGroups(tenant string) ([]ss12000v1.SchoolUnitGroup, error) {
	objectMap, err := w.GetParsedResources(tenant, "SchoolUnitGroups")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.SchoolUnitGroup, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.SchoolUnitGroup))
	}
	return result, nil
}

// GetOrganisations returns all organisations
func (w *Windermere) GetOrganisations(tenant string) ([]ss12000v1.Organisation, error) {
	objectMap, err := w.GetParsedResources(tenant, "Organisations")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.Organisation, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.Organisation))
	}
	return result, nil
}

// GetSchoolUnit returns a school unit with a specific id (if it exists)
func (w *Windermere) GetSchoolUnit(tenant, id string) (*ss12000v1.SchoolUnit, error) {
	parsedObject, err := w.GetParsedResource(tenant, "SchoolUnits", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.SchoolUnit), nil
}

// GetSchoolUnitGroup returns a school unit group with a specific id (if it exists)
func (w *Windermere) GetSchoolUnitGroup(tenant, id string) (*ss12000v1.SchoolUnitGroup, error) {
	parsedObject, err := w.GetParsedResource(tenant, "SchoolUnitGroups", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.SchoolUnitGroup), nil
}

// GetOrganisation returns an organisation with a specific id (if it exists)
func (w *Windermere) GetOrganisation(tenant string, id string) (*ss12000v1.Organisation, error) {
	parsedObject, err := w.GetParsedResource(tenant, "Organisations", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.Organisation), nil
}

// GetActivities returns all activities
func (w *Windermere) GetActivities(tenant string) ([]ss12000v1.Activity, error) {
	objectMap, err := w.GetParsedResources(tenant, "Activities")
	if err != nil {
		return nil, err
	}
	result := make([]ss12000v1.Activity, 0, len(objectMap))

	for _, object := range objectMap {
		result = append(result, *object.(*ss12000v1.Activity))
	}
	return result, nil
}

// GetActivity returns an activity with a specific id (if it exists)
func (w *Windermere) GetActivity(tenant, id string) (*ss12000v1.Activity, error) {
	parsedObject, err := w.GetParsedResource(tenant, "Activities", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.Activity), nil
}

// GetEmployment returns an employment with a specific id (if it exists)
func (w *Windermere) GetEmployment(tenant, id string) (*ss12000v1.Employment, error) {
	parsedObject, err := w.GetParsedResource(tenant, "Employments", id)

	if err != nil {
		return nil, err
	}

	return parsedObject.(*ss12000v1.Employment), nil
}

// GetStudentGroupsForSchoolUnit returns all student groups for a specific school unit
func (w *Windermere) GetStudentGroupsForSchoolUnit(tenant, code string) ([]ss12000v1.StudentGroup, error) {
	jsonObjects, err := w.GetResources(tenant, "StudentGroups")
	if err != nil {
		return nil, err
	}

	result := make([]ss12000v1.StudentGroup, 0)

	schoolUnitUUID, err := w.getUUIDForSchoolUnitCode(tenant, code)

	if err != nil {
		return nil, fmt.Errorf("Failed to find UUID for school unit with code %s", code)
	}

	for _, jsonObject := range jsonObjects {
		var studentGroup ss12000v1.StudentGroup
		err := json.Unmarshal([]byte(jsonObject), &studentGroup)

		if err != nil {
			return nil, fmt.Errorf("Failed to parse student group definition (%v)", err)
		}

		if studentGroup.Owner.Value == schoolUnitUUID {
			result = append(result, studentGroup)
		}
	}
	return result, nil
}

func (w *Windermere) getUUIDForSchoolUnitCode(tenant, code string) (string, error) {
	schoolUnits, err := w.GetSchoolUnits(tenant)

	if err != nil {
		return "", err
	}

	for i := range schoolUnits {
		if schoolUnits[i].SchoolUnitCode == code {
			return schoolUnits[i].ExternalID, nil
		}
	}
	return "", fmt.Errorf("No school unit found with code %s", code)
}
