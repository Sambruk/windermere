package test

import "testing"

func Ensure(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("%v", err)
		t.FailNow()
	}
}

func MustFail(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error")
	}
}
