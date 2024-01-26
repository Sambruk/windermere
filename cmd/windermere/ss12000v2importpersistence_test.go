package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/Sambruk/windermere/ss12000v2import"
	"github.com/Sambruk/windermere/test"
)

func TestImportPersistence(t *testing.T) {
	const tenant1 = "tenant1"
	config1 := ImportConfig{
		Tenant: tenant1,
		APIConfiguration: ss12000v2import.APIConfiguration{
			URL:            "https://example.com/ss12000/v2.1",
			Authentication: ss12000v2import.AuthAPIKey,
			ClientSecret:   "verysecretkey",
			APIKeyHeader:   "X-API-Key",
		},
		FullImportFrequency:        7 * 24 * 60 * 60,
		FullImportRetryWait:        60 * 60,
		IncrementalImportFrequency: 1 * 24 * 60 * 60,
		IncrementalImportRetryWait: 60 * 60,
	}

	p, err := NewSS12000v2ImportPersistence(":memory:")
	test.Ensure(t, err)

	tenants, err := p.GetAllImports()
	test.Ensure(t, err)

	if len(tenants) > 0 {
		t.Error("Unexpected configs in new persistence")
	}

	config, err := p.GetImportConfig(tenant1)
	test.Ensure(t, err)

	if config != nil {
		t.Error("Unexpected config in new persistence")
	}

	// Deleting a non-existant config is not expected to give an error
	err = p.DeleteImport(tenant1)
	test.Ensure(t, err)

	err = p.AddImport(config1)
	test.Ensure(t, err)

	config, err = p.GetImportConfig(tenant1)
	test.Ensure(t, err)

	if config == nil {
		t.Error("Expected to find tenant1")
	}

	if *config != config1 {
		t.Error("tenant1 looked different after storing/retrieving")
	}

	tenants, err = p.GetAllImports()
	test.Ensure(t, err)

	if !reflect.DeepEqual(tenants, []string{tenant1}) {
		t.Errorf("Unexpected list of tentants: %v", tenants)
	}

	config1.FullImportFrequency = 123

	// AddImport is also used to modify existing
	err = p.AddImport(config1)
	test.Ensure(t, err)

	config, err = p.GetImportConfig(tenant1)
	test.Ensure(t, err)

	if config == nil {
		t.Error("Expected to find tenant1")
	}

	if *config != config1 {
		t.Error("tenant1 looked different after storing/retrieving")
	}

	tenants, err = p.GetAllImports()
	test.Ensure(t, err)

	if !reflect.DeepEqual(tenants, []string{tenant1}) {
		t.Errorf("Unexpected list of tentants: %v", tenants)
	}

	err = p.DeleteImport(tenant1)
	test.Ensure(t, err)

	tenants, err = p.GetAllImports()
	test.Ensure(t, err)

	if len(tenants) != 0 {
		t.Error("Unexpected tenants after delete")
	}
}

func TestImportPersistenceHistory(t *testing.T) {
	const tenant1 = "tenant1"
	config1 := ImportConfig{
		Tenant: tenant1,
	}

	p, err := NewSS12000v2ImportPersistence(":memory:")
	test.Ensure(t, err)

	err = p.AddImport(config1)
	test.Ensure(t, err)

	ih := p.GetHistory(tenant1)

	timestamp, err := ih.GetTimeOfLastCompletedFullImport()
	test.Ensure(t, err)

	if !timestamp.IsZero() {
		t.Error("Expected zero time of last completed full import for new import")
	}

	now := time.Now()
	err = ih.SetTimeOfLastCompletedFullImport(now)
	test.Ensure(t, err)

	// Make sure changing the configuration doesn't affect the history
	config1.FullImportFrequency = 123
	p.AddImport(config1)

	timestamp, err = ih.GetTimeOfLastCompletedFullImport()
	test.Ensure(t, err)

	if timestamp.Format(time.RFC3339) != now.Format(time.RFC3339) {
		t.Errorf("Unexpected time of last completed full import after changing configuration (got: %s, wanted: %s)", timestamp.String(), now.String())
	}

	timestamp, err = ih.GetMostRecentlyCreated("foo")
	test.Ensure(t, err)

	if !timestamp.IsZero() {
		t.Error("Expected zero time of most recently created before it's set")
	}

	err = ih.RecordMostRecent([]time.Time{now}, []time.Time{}, "foo")
	test.Ensure(t, err)

	timestamp, err = ih.GetMostRecentlyCreated("foo")
	test.Ensure(t, err)

	if timestamp.Format(time.RFC3339) != now.Format(time.RFC3339) {
		t.Errorf("Unexpected time of of most recently created (got: %s, wanted: %s)", timestamp.String(), now.String())
	}
}
