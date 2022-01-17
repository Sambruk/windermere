package windermere

import (
	"testing"

	"github.com/Sambruk/windermere/ss12000v1"
)

func TestUUIDValidation(t *testing.T) {
	validator := UUIDValidator()

	withUUID := func(uuid string) ss12000v1.Object {
		return &ss12000v1.Organisation{
			ExternalID: uuid,
		}
	}
	MustFail(t, validator(withUUID("foo")))
	Ensure(t, validator(withUUID("fc5a14b6-d08a-4280-9a48-0952ff5d5f26")))
	// Garbage before valid UUID
	MustFail(t, validator(withUUID("ffc5a14b6-d08a-4280-9a48-0952ff5d5f26")))
	// Garbage after
	MustFail(t, validator(withUUID("fc5a14b6-d08a-4280-9a48-0952ff5d5f266")))
	// Invalid character ('g')
	MustFail(t, validator(withUUID("gc5a14b6-d08a-4280-9a48-0952ff5d5f266")))
	// Upper case letters is ok
	Ensure(t, validator(withUUID("Fc5a14b6-d08a-4280-9a48-0952ff5d5f26")))
}

func TestSchoolUnitCodeValidation(t *testing.T) {
	validator := SchoolUnitCodeValidator()

	withCode := func(schoolUnitCode string) ss12000v1.Object {
		return &ss12000v1.SchoolUnit{
			SchoolUnitCode: schoolUnitCode,
		}
	}

	Ensure(t, validator(withCode("12345679")))
	MustFail(t, validator(withCode("1234567")))
	MustFail(t, validator(withCode("abcdefgh")))

	// Non-SchoolUnit object
	obj := &ss12000v1.Organisation{}
	Ensure(t, validator(obj))
}
