package config

import (
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":     "postgres://lms@db:5432/lms",
		"OIDC_ISSUER":      "http://lms.example.test/idp/realms/lms",
		"OIDC_JWKS_URL":    "http://keycloak:8080/idp/realms/lms/protocol/openid-connect/certs",
		"OIDC_AUDIENCE":    "lms-api",
		"AI_SERVICE_URL":   "http://ai-service:8000/",
		"AI_SERVICE_TOKEN": "x",
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(envFrom(validEnv()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" || cfg.EvidenceDir != "/data/evidence" || cfg.SeedDemo {
		t.Fatalf("bad defaults: %+v", cfg)
	}
	if cfg.AIServiceURL != "http://ai-service:8000" {
		t.Fatalf("trailing slash not trimmed: %q", cfg.AIServiceURL)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	_, err := Load(envFrom(map[string]string{}))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"DATABASE_URL", "OIDC_ISSUER", "OIDC_JWKS_URL", "OIDC_AUDIENCE", "AI_SERVICE_URL", "AI_SERVICE_TOKEN"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error should mention %s: %v", name, err)
		}
	}
}

func TestLoadInvalidValues(t *testing.T) {
	cases := map[string]string{
		"PORT":          "abc",
		"SEED_DEMO":     "maybe",
		"OIDC_ISSUER":   "not a url",
		"OIDC_JWKS_URL": "ftp://x/y",
	}
	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			env := validEnv()
			env[k] = v
			if _, err := Load(envFrom(env)); err == nil || !strings.Contains(err.Error(), k) {
				t.Fatalf("expected error mentioning %s, got %v", k, err)
			}
		})
	}
}

func TestLoadSeedDemo(t *testing.T) {
	env := validEnv()
	env["SEED_DEMO"] = "true"
	env["PORT"] = "9090"
	cfg, err := Load(envFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SeedDemo || cfg.Port != "9090" {
		t.Fatalf("got %+v", cfg)
	}
}
