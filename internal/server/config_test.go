package server

import (
	"reflect"
	"sort"
	"testing"
)

func fakeEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestMergeEnvConfig_AdoptsUnsetValues(t *testing.T) {
	got, migrated := mergeEnvConfig(settings{}, fakeEnv(map[string]string{
		"ORIEL_BASE_PATH":     "/oriel",
		"ORIEL_ALLOWED_HOSTS": "Box.Tailnet.TS.net, oriel.example.com",
		"ORIEL_PROVIDER_URL":  "http://127.0.0.1:8899",
	}))

	if got.BasePath != "/oriel" {
		t.Errorf("BasePath = %q, want /oriel", got.BasePath)
	}
	if got.ProviderURL != "http://127.0.0.1:8899" {
		t.Errorf("ProviderURL = %q", got.ProviderURL)
	}
	want := []string{"box.tailnet.ts.net", "oriel.example.com"} // normalized: lowercased, sorted
	if !reflect.DeepEqual(got.AllowedHosts, want) {
		t.Errorf("AllowedHosts = %v, want %v", got.AllowedHosts, want)
	}
	sort.Strings(migrated)
	wantMigrated := []string{"ORIEL_ALLOWED_HOSTS", "ORIEL_BASE_PATH", "ORIEL_PROVIDER_URL"}
	if !reflect.DeepEqual(migrated, wantMigrated) {
		t.Errorf("migrated = %v, want %v", migrated, wantMigrated)
	}
}

func TestMergeEnvConfig_NeverOverwritesExisting(t *testing.T) {
	existing := settings{
		BasePath:     "/already",
		ProviderURL:  "http://keep",
		AllowedHosts: []string{"kept.example.com"},
	}
	got, migrated := mergeEnvConfig(existing, fakeEnv(map[string]string{
		"ORIEL_BASE_PATH":     "/env",
		"ORIEL_ALLOWED_HOSTS": "env.example.com",
		"ORIEL_PROVIDER_URL":  "http://env",
	}))

	if got.BasePath != "/already" || got.ProviderURL != "http://keep" ||
		!reflect.DeepEqual(got.AllowedHosts, []string{"kept.example.com"}) {
		t.Errorf("settings.json values must win, got %+v", got)
	}
	if len(migrated) != 0 {
		t.Errorf("expected no migration when values already set, got %v", migrated)
	}
}

func TestMergeEnvConfig_NoEnvIsNoop(t *testing.T) {
	_, migrated := mergeEnvConfig(settings{}, fakeEnv(nil))
	if len(migrated) != 0 {
		t.Errorf("expected no migration with empty env, got %v", migrated)
	}
}
