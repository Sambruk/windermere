package ss12000v1tov2

import (
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/Sambruk/lakeside/ss12000v2"
	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/windermere"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/google/uuid"
)

// The Adapter converts between SS12000:2018 and SS12000:2020
// It implements the Provider interface as defined in the ss12000v2
// package. It gets its data from a Windermere server.
type Adapter struct {
	W                   *windermere.Windermere    // Windermere server to read from
	Tenant              string                    // Name of the tenant from which we're reading
	ConvertToPlacements bool                      // Whether or not to convert enrolments in SchoolUnits of type FS and FTH to Placement objects
	personExpander      *ss12000v2.PersonExpander // Helps us expand Person objects
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func userToPerson(user ss12000v1.User, skipEnrolmentsFor map[string]bool) (p ss12000v2.PersonExpanded, err error) {
	p.Id, err = uuid.Parse(user.ID)
	if err != nil {
		return
	}
	p.EduPersonPrincipalNames = &[]string{user.UserName}
	p.FamilyName = user.Name.FamilyName
	p.GivenName = user.Name.GivenName

	emails := make([]ss12000v2.Email, 0)
	p.Emails = &emails

	for i := range user.Emails {
		// In SCIM (and therefore SS12000:2018) email addresses are recommended
		// to be valid, but not required. In SS12000:2020 they are required to
		// have the format 'email' and thus required to be valid. So if there
		// are invalid email addresses, we'll simply skip them.
		if !validEmail(user.Emails[i].Value) {
			continue
		}
		var email ss12000v2.Email
		email.Value = openapi_types.Email(user.Emails[i].Value)
		if user.Emails[i].Type != "" {
			email.Type = ss12000v2.EmailType(user.Emails[i].Type)
		}
		emails = append(emails, email)
	}

	if user.Extension.CivicNo != nil {
		var civicNo ss12000v2.PersonCivicNo
		civicNo.Value = *user.Extension.CivicNo
		p.CivicNo = &civicNo
	}

	if len(user.Extension.Enrolments) > 0 {

		enrolments := make([]ss12000v2.Enrolment, 0)
		for _, ue := range user.Extension.Enrolments {
			if _, ok := skipEnrolmentsFor[ue.Value]; ok {
				continue
			}
			var pe ss12000v2.Enrolment
			var schoolUnitId uuid.UUID
			schoolUnitId, err = uuid.Parse(ue.Value)
			if err != nil {
				return
			}
			pe.EnroledAt = ss12000v2.SchoolUnitReference{Id: schoolUnitId}
			pe.SchoolYear = ue.SchoolYear
			if ue.SchoolType != nil {
				var st ss12000v2.SchoolTypesEnum
				st = ss12000v2.SchoolTypesEnum(*ue.SchoolType)
				pe.SchoolType = st
			}
			if ue.ProgramCode != nil {
				ec := *ue.ProgramCode
				pe.EducationCode = &ec
			}
			enrolments = append(enrolments, pe)
		}
		p.Enrolments = &enrolments
	}

	if len(user.Extension.UserRelations) > 0 {
		responsibles := make([]ss12000v2.PersonResponsiblesInner, len(user.Extension.UserRelations))
		for i, ur := range user.Extension.UserRelations {
			r := &responsibles[i]
			var pr ss12000v2.PersonReference
			var userId uuid.UUID
			userId, err = uuid.Parse(ur.Value)
			if err != nil {
				return
			}
			pr.Id = userId
			r.Person = &pr
			var rt ss12000v2.RelationTypesEnum
			if ur.RelationType == "Annan ansvarig vuxen" {
				rt = "Annan ansvarig"
			} else {
				// "Vårdnadshavare" is the only defined value in SS12000:2018,
				// which is the same in SS12000:2020
				rt = ss12000v2.RelationTypesEnum(ur.RelationType)
			}
			r.RelationType = &rt
		}
		p.Responsibles = &responsibles
	}

	return
}

func orgReference(id string) (or ss12000v2.OrganisationReference, err error) {
	or.Id, err = uuid.Parse(id)
	return
}

func personReference(id string) (pr ss12000v2.PersonReference, err error) {
	pr.Id, err = uuid.Parse(id)
	return
}

func dutyReference(id string) (dr ss12000v2.DutyReference, err error) {
	dr.Id, err = uuid.Parse(id)
	return
}

func studentGroupToGroup(studentGroup ss12000v1.StudentGroup) (g ss12000v2.GroupExpanded, err error) {
	g.Id, err = uuid.Parse(studentGroup.ID)
	if err != nil {
		return
	}
	g.DisplayName = studentGroup.DisplayName
	g.Organisation, err = orgReference(studentGroup.Owner.Value)
	if err != nil {
		return
	}
	if studentGroup.Type != nil {
		g.GroupType = ss12000v2.GroupTypesEnum(*studentGroup.Type)
	}
	if len(studentGroup.StudentMemberships) > 0 {
		groupMemberships := make([]ss12000v2.GroupMembership, len(studentGroup.StudentMemberships))
		for i, member := range studentGroup.StudentMemberships {
			groupMemberships[i].Person, err = personReference(member.Value)
			if err != nil {
				return
			}
		}
		g.GroupMemberships = &groupMemberships
	}

	if studentGroup.SchoolType != nil {
		var st ss12000v2.SchoolTypesEnum
		st = ss12000v2.SchoolTypesEnum(*studentGroup.SchoolType)
		g.SchoolType = &st

		if schoolTypeWithPlacements(*studentGroup.SchoolType) {
			g.GroupType = "Avdelning"
		}
	}
	return
}

func schoolTypeWithPlacements(schoolType string) bool {
	return schoolType == "FS" || schoolType == "FTH"
}

func schoolUnitToOrganisation(schoolUnit ss12000v1.SchoolUnit) (o ss12000v2.Organisation, err error) {
	o.Id, err = uuid.Parse(schoolUnit.ExternalID)
	if err != nil {
		return
	}

	if schoolUnit.SchoolTypes != nil && len(*schoolUnit.SchoolTypes) > 0 {
		st := make([]ss12000v2.SchoolTypesEnum, len(*schoolUnit.SchoolTypes))
		o.SchoolTypes = &st
		for i := range *schoolUnit.SchoolTypes {
			// TODO: convert the school types properly between 2018 and 2020
			st[i] = ss12000v2.SchoolTypesEnum((*schoolUnit.SchoolTypes)[i])
		}
	}

	o.DisplayName = schoolUnit.DisplayName

	if schoolUnit.SchoolTypes != nil && len(*schoolUnit.SchoolTypes) == 1 && schoolTypeWithPlacements((*schoolUnit.SchoolTypes)[0]) {
		o.OrganisationCode = &schoolUnit.SchoolUnitCode
	} else {
		o.SchoolUnitCode = &schoolUnit.SchoolUnitCode
	}
	o.OrganisationType = ss12000v2.OTSkolenhet

	parentId := ""
	if schoolUnit.SchoolUnitGroup != nil {
		parentId = (*schoolUnit.SchoolUnitGroup).Value
	} else if schoolUnit.Organisation != nil {
		parentId = (*schoolUnit.Organisation).Value
	}

	if parentId != "" {
		var pr ss12000v2.OrganisationParentOrganisation
		pr.Id, err = uuid.Parse(parentId)
		if err != nil {
			return
		}
		o.ParentOrganisation = &pr
	}

	if schoolUnit.MunicipalityCode != nil {
		o.MunicipalityCode = schoolUnit.MunicipalityCode
	}

	return
}

func schoolUnitGroupToOrganisation(schoolUnitGroup ss12000v1.SchoolUnitGroup) (o ss12000v2.Organisation, err error) {
	o.Id, err = uuid.Parse(schoolUnitGroup.ExternalID)
	if err != nil {
		return
	}
	o.DisplayName = schoolUnitGroup.DisplayName
	o.OrganisationType = ss12000v2.OTSkola
	return
}

func organisationToOrganisation(organisation ss12000v1.Organisation) (o ss12000v2.Organisation, err error) {
	o.Id, err = uuid.Parse(organisation.ExternalID)
	if err != nil {
		return
	}
	o.DisplayName = organisation.DisplayName
	o.OrganisationType = ss12000v2.OTHuvudman
	return
}

func activityToActivity(activity ss12000v1.Activity) (a ss12000v2.ActivityExpanded, err error) {
	a.Id, err = uuid.Parse(activity.ExternalID)
	if err != nil {
		return
	}
	a.DisplayName = activity.DisplayName
	a.Organisation.Id, err = uuid.Parse(activity.Owner.Value)
	if err != nil {
		return
	}
	a.Groups = make([]ss12000v2.GroupReference, len(activity.Groups))
	for i := range activity.Groups {
		a.Groups[i].Id, err = uuid.Parse(activity.Groups[i].Value)
		if err != nil {
			return
		}
	}

	if len(activity.Teachers) > 0 {
		dutyAssignments := make([]ss12000v2.DutyAssignment, len(activity.Teachers))
		for i := range activity.Teachers {
			dutyAssignments[i].Duty, err = dutyReference(activity.Teachers[i].Value)
			if err != nil {
				return
			}
		}
		a.Teachers = &dutyAssignments
	}

	if len(activity.ParentActivity) > 0 {
		// SS12000:2020 currently only supports a single parentActivity, so we'll just
		// take the first one.
		var pa ss12000v2.ActivityParentActivity
		pa.Id, err = uuid.Parse(activity.ParentActivity[0].Value)
		if err != nil {
			return
		}
		a.ParentActivity = &pa
	}

	return
}

func employmentToDuty(employment ss12000v1.Employment) (d ss12000v2.DutyExpanded, err error) {
	d.Id, err = uuid.Parse(employment.ID)
	if err != nil {
		return
	}
	var person ss12000v2.DutyPerson
	person.Id, err = uuid.Parse(employment.User.Value)
	if err != nil {
		return
	}
	d.Person = &person
	d.DutyAt, err = orgReference(employment.EmployedAt.Value)
	if err != nil {
		return
	}
	d.Signature = &employment.Signature
	d.DutyRole = ss12000v2.DutyExpandedDutyRole(employment.EmploymentRole)

	return
}

func (a *Adapter) findSchoolUnitsWithPlacements() (map[string]bool, error) {
	result := make(map[string]bool)
	schoolUnits, err := a.W.GetSchoolUnits(a.Tenant)
	if err != nil {
		return result, err
	}
	for _, su := range schoolUnits {
		if su.SchoolTypes != nil && len(*su.SchoolTypes) == 1 &&
			schoolTypeWithPlacements((*su.SchoolTypes)[0]) {
			result[su.ExternalID] = true
		}
	}
	return result, nil
}

func (a *Adapter) findEnrolmentsToSkip() (map[string]bool, error) {
	if !a.ConvertToPlacements {
		return make(map[string]bool), nil
	}
	return a.findSchoolUnitsWithPlacements()
}

// Persons as defined in ss12000v2.Provider
func (a *Adapter) Persons(expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	persons := make([]ss12000v2.PersonExpanded, len(users))

	for i := range users {
		persons[i], err = userToPerson(users[i], skipEnrolmentsFor)
		if err != nil {
			return nil, err
		}
	}

	a.personExpander.ExpandAll(persons, expand)

	return persons, nil
}

// LookupPersons finds persons based on UUIDs and/or civic numbers
func (a *Adapter) LookupPersons(ids map[openapi_types.UUID]bool, civicNos map[string]bool, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	persons := make([]ss12000v2.PersonExpanded, 0)

	for i := range users {
		include := false
		id, _ := uuid.Parse(users[i].ID)
		if _, ok := ids[id]; ok {
			include = true
		}
		if users[i].Extension.CivicNo != nil {
			if _, ok := civicNos[*users[i].Extension.CivicNo]; ok {
				include = true
			}
		}
		if include {
			var person ss12000v2.PersonExpanded
			person, err = userToPerson(users[i], skipEnrolmentsFor)
			if err != nil {
				return nil, err
			}
			persons = append(persons, person)
		}
	}

	a.personExpander.ExpandAll(persons, expand)

	return persons, nil
}

// PersonsWithDutyAt as defined in ss12000v2.Provider
func (a *Adapter) PersonsWithDutyAt(relationshipOrganisation openapi_types.UUID, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	employments, err := a.W.GetEmploymentsAt(a.Tenant, relationshipOrganisation.String())

	if err != nil {
		return nil, err
	}

	uniquePersons := make(map[openapi_types.UUID]bool)

	for i := range employments {
		id, err := uuid.Parse(employments[i].User.Value)
		if err != nil {
			return nil, err
		}
		uniquePersons[id] = true
	}

	return a.LookupPersons(uniquePersons, map[string]bool{}, expand)
}

func (a *Adapter) LookupResponsiblesForPersons(persons []ss12000v2.PersonExpanded, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	responsibles := make(map[openapi_types.UUID]bool)

	for i := range persons {
		if persons[i].Responsibles == nil {
			continue
		}
		for _, responsible := range *persons[i].Responsibles {
			responsibles[responsible.Person.Id] = true
		}
	}
	return a.LookupPersons(responsibles, map[string]bool{}, expand)
}

func (a *Adapter) LookupPersonsForPlacements(placements []ss12000v2.PlacementExpanded, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	uniquePersons := make(map[openapi_types.UUID]bool)

	for i := range placements {
		uniquePersons[placements[i].Child.Id] = true
	}
	return a.LookupPersons(uniquePersons, map[string]bool{}, expand)

}

func (a *Adapter) LookupResponsiblesForPlacements(placements []ss12000v2.PlacementExpanded, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	placedPersons, err := a.LookupPersonsForPlacements(placements, ss12000v2.PEONone)
	if err != nil {
		return nil, err
	}

	return a.LookupResponsiblesForPersons(placedPersons, expand)
}

func (a *Adapter) PersonsPlacedAt(relationshipOrganisation openapi_types.UUID, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	placements, err := a.PlacementsAt(relationshipOrganisation)
	if err != nil {
		return nil, err
	}
	return a.LookupPersonsForPlacements(placements, expand)
}

func (a *Adapter) PersonsResponsibleForPlacedAt(relationshipOrganisation openapi_types.UUID, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	placements, err := a.PlacementsAt(relationshipOrganisation)
	if err != nil {
		return nil, err
	}

	return a.LookupResponsiblesForPlacements(placements, expand)
}

func (a *Adapter) PersonsWithPlacement(expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	placements, err := a.Placements()
	if err != nil {
		return nil, err
	}
	return a.LookupPersonsForPlacements(placements, expand)
}

func (a *Adapter) PersonsResponsibleForPlaced(expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	placements, err := a.Placements()
	if err != nil {
		return nil, err
	}

	return a.LookupResponsiblesForPlacements(placements, expand)
}

// PersonsEnrolledAt as defined in ss12000v2.Provider
func (a *Adapter) PersonsEnrolledAt(relationshipOrganisation openapi_types.UUID, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	persons := make([]ss12000v2.PersonExpanded, 0)

	// If it's one of the organisations for which we should skip enrolments,
	// we're shouldn't find any persons enrolled at this organisation.
	if _, ok := skipEnrolmentsFor[relationshipOrganisation.String()]; ok {
		return persons, err
	}

	for i := range users {
		for j := range users[i].Extension.Enrolments {
			if users[i].Extension.Enrolments[j].Value == relationshipOrganisation.String() {
				var person ss12000v2.PersonExpanded
				person, err = userToPerson(users[i], skipEnrolmentsFor)
				if err != nil {
					return nil, err
				}
				persons = append(persons, person)
				break
			}
		}
	}

	a.personExpander.ExpandAll(persons, expand)

	return persons, nil
}

// PersonsResponsibleForEnrolledAt as defined in ss12000v2.Provider
func (a *Adapter) PersonsResponsibleForEnrolledAt(relationshipOrganisation openapi_types.UUID, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	persons := make([]ss12000v2.PersonExpanded, 0)

	// If it's one of the organisations for which we should skip enrolments,
	// we're shouldn't find any persons enrolled at this organisation.
	if _, ok := skipEnrolmentsFor[relationshipOrganisation.String()]; ok {
		return persons, err
	}

	responsibles := make(map[string]bool)

	for i := range users {
		for j := range users[i].Extension.Enrolments {
			if users[i].Extension.Enrolments[j].Value == relationshipOrganisation.String() {
				for _, ur := range users[i].Extension.UserRelations {
					responsibles[ur.Value] = true
				}
			}
		}
	}

	for i := range users {
		if _, ok := responsibles[users[i].ID]; ok {
			var person ss12000v2.PersonExpanded
			person, err = userToPerson(users[i], skipEnrolmentsFor)
			if err != nil {
				return nil, err
			}
			persons = append(persons, person)
		}
	}

	a.personExpander.ExpandAll(persons, expand)

	return persons, nil
}

// PersonsWithDuty as defined in ss12000v2.Provider
func (a *Adapter) PersonsWithDuty(expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	employments, err := a.W.GetEmployments(a.Tenant)

	if err != nil {
		return nil, err
	}

	uniquePersons := make(map[openapi_types.UUID]bool)

	for i := range employments {
		id, err := uuid.Parse(employments[i].User.Value)
		if err != nil {
			return nil, err
		}
		uniquePersons[id] = true
	}

	return a.LookupPersons(uniquePersons, map[string]bool{}, expand)
}

// PersonsEnrolled as defined in ss12000v2.Provider
func (a *Adapter) PersonsEnrolled(expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	persons := make([]ss12000v2.PersonExpanded, 0)

	for i := range users {
		if len(users[i].Extension.Enrolments) > 0 {
			var person ss12000v2.PersonExpanded
			person, err = userToPerson(users[i], skipEnrolmentsFor)
			if err != nil {
				return nil, err
			}
			// Only append if there are still enrolments after convertingo to Person
			// (which might remove enrolments in skipEnrolmentsFor)
			if person.Enrolments != nil &&
				len(*person.Enrolments) > 0 {
				persons = append(persons, person)
			}
		}
	}

	a.personExpander.ExpandAll(persons, expand)

	return persons, nil
}

// PersonsResponsibleForEnrolled as defined in ss12000v2.Provider
func (a *Adapter) PersonsResponsibleForEnrolled(expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	hasUnskippedEnrolments := func(enrolments []ss12000v1.Enrolment, toSkip map[string]bool) bool {
		for _, e := range enrolments {
			if _, ok := toSkip[e.Value]; !ok {
				return true
			}
		}
		return false
	}

	responsibles := make(map[string]bool)

	for i := range users {
		if hasUnskippedEnrolments(users[i].Extension.Enrolments, skipEnrolmentsFor) {
			for _, ur := range users[i].Extension.UserRelations {
				responsibles[ur.Value] = true
			}
		}
	}

	persons := make([]ss12000v2.PersonExpanded, 0)

	for i := range users {
		if _, ok := responsibles[users[i].ID]; ok {
			var person ss12000v2.PersonExpanded
			person, err = userToPerson(users[i], skipEnrolmentsFor)
			if err != nil {
				return nil, err
			}
			persons = append(persons, person)
		}
	}

	a.personExpander.ExpandAll(persons, expand)

	return persons, nil
}

// PersonsRelatedTo as defined in ss12000v2.Provider
func (a *Adapter) PersonsRelatedTo(relationshipOrganisation openapi_types.UUID, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	employments, err := a.W.GetEmploymentsAt(a.Tenant, relationshipOrganisation.String())

	uniquePersons := make(map[openapi_types.UUID]bool)

	for i := range employments {
		id, err := uuid.Parse(employments[i].User.Value)
		if err != nil {
			return nil, err
		}
		uniquePersons[id] = true
	}

	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	for i := range users {
		for j := range users[i].Extension.Enrolments {
			if users[i].Extension.Enrolments[j].Value == relationshipOrganisation.String() {
				id, err := uuid.Parse(users[i].ID)
				if err != nil {
					return nil, err
				}
				uniquePersons[id] = true
				for _, ur := range users[i].Extension.UserRelations {
					id, err := uuid.Parse(ur.Value)
					if err != nil {
						return nil, err
					}
					uniquePersons[id] = true
				}
			}
		}
	}

	placements, err := a.PlacementsAt(relationshipOrganisation)

	if err != nil {
		return nil, err
	}

	for _, placement := range placements {
		uniquePersons[placement.Child.Id] = true
	}

	return a.LookupPersons(uniquePersons, map[string]bool{}, expand)
}

// PersonByID as defined by ss12000v2.Provider
func (a *Adapter) PersonByID(id openapi_types.UUID, expand ss12000v2.PersonExpandOptions) (*ss12000v2.PersonExpanded, error) {
	user, err := a.W.GetUser(a.Tenant, id.String())

	if err != nil {
		return nil, err
	}

	skipEnrolmentsFor, err := a.findEnrolmentsToSkip()

	if err != nil {
		return nil, err
	}

	var person ss12000v2.PersonExpanded
	person, err = userToPerson(*user, skipEnrolmentsFor)
	if err != nil {
		return nil, err
	}

	a.personExpander.Expand(&person, expand)

	return &person, nil
}

// PersonsLookup as defined by ss12000v2.Provider
func (a *Adapter) PersonsLookup(ids []openapi_types.UUID, civicNos []string, expand ss12000v2.PersonExpandOptions) ([]ss12000v2.PersonExpanded, error) {
	idsMap := make(map[openapi_types.UUID]bool)
	civicNosMap := make(map[string]bool)

	for i := range ids {
		idsMap[ids[i]] = true
	}

	for i := range civicNos {
		civicNosMap[civicNos[i]] = true
	}

	return a.LookupPersons(idsMap, civicNosMap, expand)
}

func (a *Adapter) Reindex() {
	placements, err := a.Placements()
	if err != nil {
		log.Printf("Failed to get placements during reindexing of ss12000v1tov2 adapter: %s", err.Error())
	} else {
		a.personExpander.SetPlacements(placements)
	}

	groups, err := a.Groups()
	if err != nil {
		log.Printf("Failed to get groups during reindexing of ss12000v1tov2 adapter: %s", err.Error())
	} else {
		a.personExpander.SetGroups(groups)
	}
}

// NewAdapter is the constructor for Adapter
func NewAdapter(w *windermere.Windermere, tenant string, generatePlacements bool) *Adapter {
	a := &Adapter{
		W:                   w,
		Tenant:              tenant,
		ConvertToPlacements: generatePlacements,
		personExpander:      ss12000v2.NewPersonExpander(),
	}
	// TODO
	//	ss12000.TodoWhenNewState(sn, func() { a.Reindex() })
	a.Reindex()
	return a
}

// Groups as defined in ss12000v2.Provider
func (a *Adapter) Groups() ([]ss12000v2.GroupExpanded, error) {
	studentGroups, err := a.W.GetStudentGroups(a.Tenant)

	if err != nil {
		return nil, err
	}

	groups := make([]ss12000v2.GroupExpanded, len(studentGroups))

	for i := range studentGroups {
		groups[i], err = studentGroupToGroup(studentGroups[i])
		if err != nil {
			return nil, err
		}
	}

	return groups, nil
}

// GroupByID as defined by ss12000v2.Provider
func (a *Adapter) GroupByID(id openapi_types.UUID) (*ss12000v2.GroupExpanded, error) {
	studentGroup, err := a.W.GetStudentGroup(a.Tenant, id.String())

	if err != nil {
		return nil, err
	}
	var group ss12000v2.GroupExpanded
	group, err = studentGroupToGroup(*studentGroup)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// LookupGroups finds groups based on UUIDs
func (a *Adapter) LookupGroups(ids map[openapi_types.UUID]bool) ([]ss12000v2.GroupExpanded, error) {
	studentGroups, err := a.W.GetStudentGroups(a.Tenant)

	if err != nil {
		return nil, err
	}

	groups := make([]ss12000v2.GroupExpanded, 0)

	for i := range studentGroups {
		include := false
		id, _ := uuid.Parse(studentGroups[i].ID)
		if _, ok := ids[id]; ok {
			include = true
		}
		if include {
			var group ss12000v2.GroupExpanded
			group, err = studentGroupToGroup(studentGroups[i])
			if err != nil {
				return nil, err
			}
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// GroupsLookup as defined in ss12000v2.Provider
func (a *Adapter) GroupsLookup(ids []openapi_types.UUID) ([]ss12000v2.GroupExpanded, error) {
	idsMap := make(map[openapi_types.UUID]bool)

	for i := range ids {
		idsMap[ids[i]] = true
	}

	return a.LookupGroups(idsMap)
}

// Organisations as defined in ss12000v2.Provider
func (a *Adapter) Organisations() ([]ss12000v2.Organisation, error) {
	schoolUnits, err := a.W.GetSchoolUnits(a.Tenant)
	if err != nil {
		return nil, err
	}

	schoolUnitGroups, err := a.W.GetSchoolUnitGroups(a.Tenant)
	if err != nil {
		return nil, err
	}

	organisations, err := a.W.GetOrganisations(a.Tenant)
	if err != nil {
		return nil, err
	}

	v2Organisations := make([]ss12000v2.Organisation, len(schoolUnits)+len(schoolUnitGroups)+len(organisations))
	pos := 0
	for i := range schoolUnits {
		v2Organisations[pos], err = schoolUnitToOrganisation(schoolUnits[i])
		if err != nil {
			return nil, err
		}
		pos++
	}

	for i := range schoolUnitGroups {
		v2Organisations[pos], err = schoolUnitGroupToOrganisation(schoolUnitGroups[i])
		if err != nil {
			return nil, err
		}
		pos++
	}

	for i := range organisations {
		v2Organisations[pos], err = organisationToOrganisation(organisations[i])
		if err != nil {
			return nil, err
		}
		pos++
	}

	return v2Organisations, nil
}

// OrganisationByID as defined in ss12000v2.Provider
func (a *Adapter) OrganisationByID(id openapi_types.UUID) (*ss12000v2.Organisation, error) {
	schoolUnit, err := a.W.GetSchoolUnit(a.Tenant, id.String())
	if err == nil {
		var org ss12000v2.Organisation
		org, err = schoolUnitToOrganisation(*schoolUnit)
		if err != nil {
			return nil, err
		}
		return &org, nil
	}

	schoolUnitGroup, err := a.W.GetSchoolUnitGroup(a.Tenant, id.String())
	if err == nil {
		var org ss12000v2.Organisation
		org, err = schoolUnitGroupToOrganisation(*schoolUnitGroup)
		if err != nil {
			return nil, err
		}
		return &org, nil
	}

	organisation, err := a.W.GetOrganisation(a.Tenant, id.String())
	if err == nil {
		var org ss12000v2.Organisation
		org, err = organisationToOrganisation(*organisation)
		if err != nil {
			return nil, err
		}
		return &org, nil
	}

	return nil, fmt.Errorf("failed to find organisation with id %s", id)
}

// Activities as defined in ss12000v2.Provider
func (a *Adapter) Activities() ([]ss12000v2.ActivityExpanded, error) {
	activities, err := a.W.GetActivities(a.Tenant)

	if err != nil {
		return nil, err
	}

	v2activities := make([]ss12000v2.ActivityExpanded, len(activities))

	for i := range activities {
		v2activities[i], err = activityToActivity(activities[i])
		if err != nil {
			return nil, err
		}
	}

	return v2activities, nil
}

func (a *Adapter) ActivityByID(id openapi_types.UUID) (*ss12000v2.ActivityExpanded, error) {
	activity, err := a.W.GetActivity(a.Tenant, id.String())

	if err != nil {
		return nil, err
	}
	var v2activity ss12000v2.ActivityExpanded
	v2activity, err = activityToActivity(*activity)
	if err != nil {
		return nil, err
	}
	return &v2activity, nil
}

func (a *Adapter) ActivitiesLookup(ids, members, teachers []openapi_types.UUID) ([]ss12000v2.ActivityExpanded, error) {
	idsMap := make(map[openapi_types.UUID]bool)
	membersMap := make(map[openapi_types.UUID]bool)
	teachersMap := make(map[openapi_types.UUID]bool)

	for i := range ids {
		idsMap[ids[i]] = true
	}

	for i := range members {
		membersMap[members[i]] = true
	}

	for i := range teachers {
		teachersMap[teachers[i]] = true
	}

	return a.LookupActivities(idsMap, membersMap, teachersMap)
}

// LookupActivities finds activities based on UUIDs and/or group members and/or teachers
func (a *Adapter) LookupActivities(ids, members, teachers map[openapi_types.UUID]bool) ([]ss12000v2.ActivityExpanded, error) {
	v1activities, err := a.W.GetActivities(a.Tenant)

	if err != nil {
		return nil, err
	}

	matchingGroups := make(map[string]bool)

	if len(members) > 0 {
		groups, err := a.W.GetStudentGroups(a.Tenant)
		if err != nil {
			return nil, err
		}

		for i := range groups {
			for _, member := range groups[i].StudentMemberships {
				id, _ := uuid.Parse(member.Value)
				if _, ok := members[id]; ok {
					matchingGroups[groups[i].ID] = true
				}
			}
		}
	}

	v2activities := make([]ss12000v2.ActivityExpanded, 0)

	for i := range v1activities {
		include := false

		id, _ := uuid.Parse(v1activities[i].ExternalID)
		if _, ok := ids[id]; ok {
			include = true
		}

		if !include && len(teachers) > 0 {
			for _, teachRef := range v1activities[i].Teachers {
				teacherID, _ := uuid.Parse(teachRef.Value)
				if _, ok := teachers[teacherID]; ok {
					include = true
				}
			}
		}

		if !include && len(members) > 0 {
			for _, groupRef := range v1activities[i].Groups {
				if _, ok := matchingGroups[groupRef.Value]; ok {
					include = true
				}
			}
		}

		if include {
			var activity ss12000v2.ActivityExpanded
			activity, err = activityToActivity(v1activities[i])
			if err != nil {
				return nil, err
			}
			v2activities = append(v2activities, activity)
		}
	}

	return v2activities, nil
}

// Duties as defined in ss12000v2.Provider
func (a *Adapter) Duties() ([]ss12000v2.DutyExpanded, error) {
	employments, err := a.W.GetEmployments(a.Tenant)

	if err != nil {
		return nil, err
	}

	duties := make([]ss12000v2.DutyExpanded, len(employments))

	for i := range duties {
		duties[i], err = employmentToDuty(employments[i])
		if err != nil {
			return nil, err
		}
	}

	return duties, nil
}

func (a *Adapter) DutyByID(id openapi_types.UUID) (*ss12000v2.DutyExpanded, error) {
	employment, err := a.W.GetEmployment(a.Tenant, id.String())

	if err != nil {
		return nil, err
	}
	var duty ss12000v2.DutyExpanded
	duty, err = employmentToDuty(*employment)
	if err != nil {
		return nil, err
	}
	return &duty, nil
}

func (a *Adapter) DutiesLookup(ids []openapi_types.UUID) ([]ss12000v2.DutyExpanded, error) {
	idsMap := make(map[openapi_types.UUID]bool)

	for i := range ids {
		idsMap[ids[i]] = true
	}

	return a.LookupDuties(idsMap)
}

// LookupDuties finds duties based on UUIDs
func (a *Adapter) LookupDuties(ids map[openapi_types.UUID]bool) ([]ss12000v2.DutyExpanded, error) {
	employments, err := a.W.GetEmployments(a.Tenant)

	if err != nil {
		return nil, err
	}

	duties := make([]ss12000v2.DutyExpanded, 0)

	for i := range employments {
		include := false

		id, _ := uuid.Parse(employments[i].ID)
		if _, ok := ids[id]; ok {
			include = true
		}

		if include {
			var duty ss12000v2.DutyExpanded
			duty, err = employmentToDuty(employments[i])
			if err != nil {
				return nil, err
			}
			duties = append(duties, duty)
		}
	}

	return duties, nil
}

func (a *Adapter) LookupPlacements(suwp map[string]bool) ([]ss12000v2.PlacementExpanded, error) {
	if !a.ConvertToPlacements {
		return nil, nil
	}

	users, err := a.W.GetUsers(a.Tenant)

	if err != nil {
		return nil, err
	}

	relations := make(map[string]bool)

	createRelation := func(user, schoolUnit string) string {
		return user + " " + schoolUnit
	}

	createPlacement := func(relation string) (placement ss12000v2.PlacementExpanded) {
		strs := strings.Split(relation, " ")
		user := strs[0]
		schoolUnit := strs[1]
		id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(relation))
		placement.Id = id
		placement.Child.Id, _ = uuid.Parse(user)
		placement.PlacedAt.Id, _ = uuid.Parse(schoolUnit)
		return
	}

	// For each User with an enrolment in one of the SchoolUnits in suwp
	// instead of enrolments, we create a Placement object
	for _, u := range users {
		if u.Extension.Enrolments != nil {
			for _, enrolment := range u.Extension.Enrolments {
				if _, ok := suwp[enrolment.Value]; ok {
					relations[createRelation(u.ID, enrolment.Value)] = true
				}
			}
		}
	}

	// Also create Placement objects based on group membership in groups belonging
	// to those SchoolUnits
	for su := range suwp {
		suObj, err := a.W.GetSchoolUnit(a.Tenant, su)
		if err != nil {
			return nil, err
		}
		groups, err := a.W.GetStudentGroupsForSchoolUnit(a.Tenant, suObj.SchoolUnitCode)

		for _, group := range groups {
			for _, membership := range group.StudentMemberships {
				relations[createRelation(membership.Value, su)] = true
			}
		}
	}

	// Create the placement objects
	placements := make([]ss12000v2.PlacementExpanded, 0, len(relations))
	for relation := range relations {
		placements = append(placements, createPlacement(relation))
	}
	return placements, nil

}

func (a *Adapter) Placements() ([]ss12000v2.PlacementExpanded, error) {
	suwp, err := a.findSchoolUnitsWithPlacements()
	if err != nil {
		return nil, err
	}

	return a.LookupPlacements(suwp)
}

func (a *Adapter) PlacementsAt(id openapi_types.UUID) ([]ss12000v2.PlacementExpanded, error) {
	su, err := a.W.GetSchoolUnit(a.Tenant, id.String())
	if err != nil {
		return nil, err
	}
	suwp := make(map[string]bool)
	if su.SchoolTypes != nil && len(*su.SchoolTypes) == 1 &&
		schoolTypeWithPlacements((*su.SchoolTypes)[0]) {
		suwp[su.ExternalID] = true
	}
	return a.LookupPlacements(suwp)
}

// DeletedEntities is not properly implemented for this adapter since we dont
// keep track of the history of which objects used to exist in the SS12000:2018
// data model.
func (a *Adapter) DeletedEntities(after *time.Time, types []string) (map[string][]openapi_types.UUID, error) {
	return nil, errors.New("ss12000v1tov2.Adapter doesn't implement DeletedEntities")
}
