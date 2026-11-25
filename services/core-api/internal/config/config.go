// Package config loads core-api settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Config is the validated runtime configuration.
type Config struct {
	Port           string
	DatabaseURL    string
	OIDCIssuer     string
	OIDCJWKSURL    string
	OIDCAudience   string
	AIServiceURL   string
	AIServiceToken string
	EvidenceDir    string
	SeedDemo       bool
}

// Load reads and validates configuration using getenv (normally os.Getenv).
// All problems are reported together so operators can fix them in one go.
func Load(getenv func(string) string) (Config, error) {
	get := func(k string) string { return strings.TrimSpace(getenv(k)) }

	cfg := Config{
		Port:           get("PORT"),
		DatabaseURL:    get("DATABASE_URL"),
		OIDCIssuer:     get("OIDC_ISSUER"),
		OIDCJWKSURL:    get("OIDC_JWKS_URL"),
		OIDCAudience:   get("OIDC_AUDIENCE"),
		AIServiceURL:   strings.TrimRight(get("AI_SERVICE_URL"), "/"),
		AIServiceToken: get("AI_SERVICE_TOKEN"),
		EvidenceDir:    get("EVIDENCE_DIR"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.EvidenceDir == "" {
		cfg.EvidenceDir = "/data/evidence"
	}

	var errs []error
	for _, req := range []struct{ name, val string }{
		{"DATABASE_URL", cfg.DatabaseURL},
		{"OIDC_ISSUER", cfg.OIDCIssuer},
		{"OIDC_JWKS_URL", cfg.OIDCJWKSURL},
		{"OIDC_AUDIENCE", cfg.OIDCAudience},
		{"AI_SERVICE_URL", cfg.AIServiceURL},
		{"AI_SERVICE_TOKEN", cfg.AIServiceToken},
	} {
		if req.val == "" {
			errs = append(errs, fmt.Errorf("%s is required", req.name))
		}
	}

	if p, err := strconv.Atoi(cfg.Port); err != nil || p < 1 || p > 65535 {
		errs = append(errs, fmt.Errorf("PORT must be a number between 1 and 65535"))
	}
	for _, u := range []struct{ name, val string }{
		{"OIDC_ISSUER", cfg.OIDCIssuer},
		{"OIDC_JWKS_URL", cfg.OIDCJWKSURL},
		{"AI_SERVICE_URL", cfg.AIServiceURL},
	} {
		if u.val != "" && !isHTTPURL(u.val) {
			errs = append(errs, fmt.Errorf("%s must be an absolute http(s) URL", u.name))
		}
	}

	if v := get("SEED_DEMO"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("SEED_DEMO must be true or false"))
		}
		cfg.SeedDemo = b
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
