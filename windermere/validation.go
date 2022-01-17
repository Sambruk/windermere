package windermere

import (
	"fmt"
	"regexp"

	"github.com/Sambruk/windermere/ss12000v1"
)

// A function that does some kind of validation of an SS12000 object
type Validator func(obj ss12000v1.Object) error

// Dummy validator that does nothing
func NoValidation(obj ss12000v1.Object) error {
	return nil
}

// MultiValidator creates a single Validator from several.
// The validators will be applied in the order in the slice.
func MultiValidator(validators []Validator) Validator {
	return func(obj ss12000v1.Object) error {
		for i := range validators {
			if err := validators[i](obj); err != nil {
				return err
			}
		}
		return nil
	}
}

// UUIDValidator ensures the object has a valid UUID
func UUIDValidator() Validator {
	re := regexp.MustCompile(`(?i)^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	return func(obj ss12000v1.Object) error {
		if !re.Match([]byte(obj.GetID())) {
			return fmt.Errorf("invalid UUID: %s", obj.GetID())
		}
		return nil
	}
}

// SchoolUnitCodeValidator ensures the object has a valid schoolUnitCode (if it's a school unit)
func SchoolUnitCodeValidator() Validator {
	re := regexp.MustCompile(`[0-9]{8}`)
	return func(obj ss12000v1.Object) error {
		schoolUnit, ok := obj.(*ss12000v1.SchoolUnit)
		if !ok {
			return nil
		}
		if !re.Match([]byte(schoolUnit.SchoolUnitCode)) {
			return fmt.Errorf("invalid school unit code: %s", schoolUnit.SchoolUnitCode)
		}
		return nil
	}
}

// Convenience function for creating a validator with specified validators included
func CreateOptionalValidator(uuid, schoolUnitCode bool) Validator {
	validators := make([]Validator, 0)

	if uuid {
		validators = append(validators, UUIDValidator())
	}

	if schoolUnitCode {
		validators = append(validators, SchoolUnitCodeValidator())
	}

	return MultiValidator(validators)
}
