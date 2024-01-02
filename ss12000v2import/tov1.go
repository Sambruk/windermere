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
	"fmt"

	"github.com/Sambruk/windermere/ss12000v1"
	"github.com/Sambruk/windermere/ss12000v2"
)

func organisationToV1(v2org *ss12000v2.Organisation) *ss12000v1.Organisation {
	var v1org ss12000v1.Organisation
	v1org.DisplayName = v2org.DisplayName
	v1org.ExternalID = v2org.Id.String()
	return &v1org
}

// Converts an Organisation object to a SchoolUnit (assumes v2org.OrganisationType is set to OrganisationTypeEnumSkolenhet)
func schoolUnitToV1(v2org *ss12000v2.Organisation) (*ss12000v1.SchoolUnit, error) {
	var v1schoolUnit ss12000v1.SchoolUnit
	v1schoolUnit.DisplayName = v2org.DisplayName
	v1schoolUnit.ExternalID = v2org.Id.String()
	v1schoolUnit.MunicipalityCode = v2org.MunicipalityCode
	if v2org.SchoolTypes != nil {
		schoolTypes := make([]string, len(*v2org.SchoolTypes))
		for i, st := range *v2org.SchoolTypes {
			schoolTypes[i] = schoolTypeToV1(st)
		}
		v1schoolUnit.SchoolTypes = &schoolTypes
	}
	if v2org.SchoolUnitCode == nil {
		return nil, fmt.Errorf("can't convert Organisation of type SchoolUnit (SS12000 v2.1) to SchoolUnit (SS12000 v1.0) since SchoolUnitCode is missing")

	}
	v1schoolUnit.SchoolUnitCode = *v2org.SchoolUnitCode
	return &v1schoolUnit, nil
}

func emailToV1(v2email ss12000v2.Email) ss12000v1.SCIMEmail {
	var v1email ss12000v1.SCIMEmail
	v1email.Type = string(v2email.Type)
	v1email.Value = string(v2email.Value)
	return v1email
}

// Mapping table from school types in SS12000 v2.1 to v1.0
var schoolTypev2Tov1Map = map[ss12000v2.SchoolTypesEnum]string{
	ss12000v2.ABU:      "AU",
	ss12000v2.AU:       "AU",
	ss12000v2.FHS:      "FHS",
	ss12000v2.FKLASS:   "FSK",
	ss12000v2.FS:       "FS",
	ss12000v2.FTH:      "FTH",
	ss12000v2.GR:       "GR",
	ss12000v2.GRS:      "GRS",
	ss12000v2.GY:       "GY",
	ss12000v2.GYS:      "GYS",
	ss12000v2.HS:       "HS",
	ss12000v2.KKU:      "AU",
	ss12000v2.KU:       "AU",
	ss12000v2.OPPFTH:   "FTH",
	ss12000v2.SAM:      "SAM",
	ss12000v2.SARVUX:   "SUV",
	ss12000v2.SARVUXGR: "SUV",
	ss12000v2.SARVUXGY: "SUV",
	ss12000v2.SFI:      "AU",
	ss12000v2.SP:       "SP",
	ss12000v2.STF:      "AU",
	ss12000v2.TR:       "AU",
	ss12000v2.VUX:      "VUX",
	ss12000v2.VUXGR:    "VUX",
	ss12000v2.VUXGY:    "VUX",
	ss12000v2.VUXSARGR: "SUV",
	ss12000v2.VUXSARGY: "SUV",
	ss12000v2.VUXSARTR: "SUV",
	ss12000v2.VUXSFI:   "VUX",
	ss12000v2.YH:       "YH",
}

func schoolTypeToV1(v2schoolType ss12000v2.SchoolTypesEnum) string {
	return schoolTypev2Tov1Map[v2schoolType]
}

func enrolmentToV1(v2enrolment ss12000v2.Enrolment) ss12000v1.Enrolment {
	var v1enrolment ss12000v1.Enrolment
	v1enrolment.ProgramCode = v2enrolment.EducationCode
	v1enrolment.Value = v2enrolment.EnroledAt.Id.String()
	schoolType := schoolTypeToV1(v2enrolment.SchoolType)
	v1enrolment.SchoolType = &schoolType
	v1enrolment.SchoolYear = v2enrolment.SchoolYear
	return v1enrolment
}

func externalIdentifierToV1(v2externalIdentifier ss12000v2.ExternalIdentifier) ss12000v1.ExternalIdentifier {
	return ss12000v1.ExternalIdentifier{
		Value:          v2externalIdentifier.Value,
		Context:        v2externalIdentifier.Context,
		GloballyUnique: v2externalIdentifier.GloballyUnique,
	}
}

func personToV1(v2person *ss12000v2.Person) (*ss12000v1.User, error) {
	var v1user ss12000v1.User
	v1user.DisplayName = v2person.GivenName + " " + v2person.FamilyName
	v1user.ID = v2person.Id.String()

	eppns := v2person.EduPersonPrincipalNames
	if eppns == nil || len(*eppns) == 0 {
		return nil, fmt.Errorf("can't convert Person (SS12000 v2.1) to User (SS12000 v1.0) since EduPersonPrincipalName is missing")
	}
	v1user.UserName = (*eppns)[0]

	if len(*eppns) > 1 {
		const eppnURN = "urn:oid:1.3.6.1.4.1.5923.1.1.1.6" // eduPersonPrincipalName
		var extension ss12000v1.EgilUserExtension
		v1user.EgilExtension = &extension
		extension.ExternalIdentifiers = make([]ss12000v1.ExternalIdentifier, 0)
		for i := 1; i < len(*eppns); i++ {
			var ei ss12000v1.ExternalIdentifier
			ei.Value = (*eppns)[i]
			ei.Context = eppnURN
			ei.GloballyUnique = true
			extension.ExternalIdentifiers = append(extension.ExternalIdentifiers, ei)
		}
	}

	v1user.Name.GivenName = v2person.GivenName
	v1user.Name.FamilyName = v2person.FamilyName

	if v2person.Emails != nil {
		v1user.Emails = make([]ss12000v1.SCIMEmail, len(*v2person.Emails))
		for i, v2email := range *v2person.Emails {
			v1user.Emails[i] = emailToV1(v2email)
		}
	}

	if v2person.Enrolments != nil {
		v1user.Extension.Enrolments = make([]ss12000v1.Enrolment, len(*v2person.Enrolments))
		for i, v2enrolment := range *v2person.Enrolments {
			v1user.Extension.Enrolments[i] = enrolmentToV1(v2enrolment)
		}
	}

	if v2person.ExternalIdentifiers != nil {
		if v1user.EgilExtension == nil {
			v1user.EgilExtension = &ss12000v1.EgilUserExtension{}
		}
		for _, ei := range *(v2person.ExternalIdentifiers) {
			v1user.EgilExtension.ExternalIdentifiers = append(v1user.EgilExtension.ExternalIdentifiers, externalIdentifierToV1(ei))
		}
	}

	// Skipping civicNo and userRelations for now since the backend doesn't support storing that information anyway

	return &v1user, nil
}

func groupTypeToV1(groupTypeV2 ss12000v2.GroupTypesEnum) string {
	switch groupTypeV2 {
	case ss12000v2.GroupTypesEnumUndervisning:
		return "Undervisning"
	case ss12000v2.GroupTypesEnumKlass:
		return "Klass"
	case ss12000v2.GroupTypesEnumMentor:
		return "Mentor"
	case ss12000v2.GroupTypesEnumSchema:
		return "Schema"
	default:
		return "Övrigt"
	}
}

func groupToV1(v2group *ss12000v2.Group) *ss12000v1.StudentGroup {
	var v1group ss12000v1.StudentGroup

	v1group.DisplayName = v2group.DisplayName
	v1group.ID = v2group.Id.String()
	v1group.Owner.Value = v2group.Organisation.Id.String()
	if v2group.SchoolType != nil {
		var st = schoolTypeToV1(*v2group.SchoolType)
		v1group.SchoolType = &st
	}

	groupType := groupTypeToV1(v2group.GroupType)
	v1group.Type = &groupType

	v1group.StudentMemberships = make([]ss12000v1.SCIMReference, len(*v2group.GroupMemberships))
	for i, membership := range *v2group.GroupMemberships {
		v1group.StudentMemberships[i].Value = membership.Person.Id.String()
	}
	return &v1group
}

func dutyRoleToV1(dutyRole ss12000v2.DutyDutyRole) string {
	if dutyRole == ss12000v2.DutyDutyRoleRektor {
		return "Rektor"
	} else if dutyRole == ss12000v2.DutyDutyRoleLärare {
		return "Lärare"
	} else if dutyRole == ss12000v2.DutyDutyRoleFörskollärare {
		return "Förskollärare"
	} else {
		return "Annan personal"
	}
}

func dutyToV1(v2duty *ss12000v2.Duty) (*ss12000v1.Employment, error) {
	var v1employment ss12000v1.Employment

	v1employment.ID = v2duty.Id.String()
	v1employment.EmployedAt.Value = v2duty.DutyAt.Id.String()
	v1employment.EmploymentRole = dutyRoleToV1(v2duty.DutyRole)
	if v2duty.Person == nil {
		return nil, fmt.Errorf("can't convert Duty (SS12000 v2.1) to Employment (SS12000 v1.0) since Person reference is missing")
	}
	v1employment.User.Value = v2duty.Person.Id.String()
	if v2duty.Signature != nil {
		v1employment.Signature = *v2duty.Signature
	}
	return &v1employment, nil
}

func activityToV1(v2activity *ss12000v2.Activity) *ss12000v1.Activity {
	var v1activity ss12000v1.Activity

	v1activity.DisplayName = v2activity.DisplayName
	v1activity.ExternalID = v2activity.Id.String()
	v1activity.Owner.Value = (*v2activity).Organisation.Id.String()
	v1activity.Groups = make([]ss12000v1.SCIMReference, len(v2activity.Groups))
	for i, group := range (*v2activity).Groups {
		v1activity.Groups[i].Value = group.Id.String()
	}

	if v2activity.Teachers != nil {
		v1activity.Teachers = make([]ss12000v1.SCIMReference, len(*(v2activity.Teachers)))
		for i, teacher := range *(v2activity.Teachers) {
			v1activity.Teachers[i].Value = teacher.Duty.Id.String()
		}
	} else {
		v1activity.Teachers = make([]ss12000v1.SCIMReference, 0)
	}
	return &v1activity
}
