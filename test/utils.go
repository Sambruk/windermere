package test

import "testing"

// Ensure will verify that err == nil and report an error if it is not nil.
// The test will continue after reporting the error.
func Ensure(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}

// Require will verify that err == nil and fail the test immediately if it is not nil.
func Require(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("%v", err)
	}
}

// MustFail will verify that err != nil and report an error if it is nil.
// The test will continue after reporting the error.
func MustFail(t *testing.T, err error) {
	if err == nil {
		t.Errorf("expected error")
	}
}
