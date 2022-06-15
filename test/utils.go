package test

import "testing"

func Ensure(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}

func MustFail(t *testing.T, err error) {
	if err == nil {
		t.Errorf("expected error")
	}
}
