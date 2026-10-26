package main

import (
	"maps"
	"testing"
)

// devHash is the bcrypt hash of "home-dev-password", as in .env.example.
const devHash = "$2a$12$vBqGALXSXAKWkfA9xumGqeKtVeo8udjWiULOC3ZO44MikaeUaXN7u"

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(env(map[string]string{
		"DATABASE_URL":        "postgres://home@localhost/home",
		"ADMIN_PASSWORD_HASH": devHash,
	}))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	want := config{
		port:          "8080",
		databaseURL:   "postgres://home@localhost/home",
		passwordHash:  devHash,
		publicBaseURL: "http://localhost:8080",
		trustProxy:    false,
		loginLimit:    10,
		maxUploadMB:   5,
	}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigValues(t *testing.T) {
	cfg, err := loadConfig(env(map[string]string{
		"PORT":                "9000",
		"DATABASE_URL":        "postgres://home@db/home",
		"ADMIN_PASSWORD_HASH": devHash,
		"PUBLIC_BASE_URL":     "https://zazi.cool/",
		"TRUST_PROXY":         "true",
		"RATE_LIMIT_LOGIN":    "3",
		"MAX_UPLOAD_MB":       "8",
	}))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.port != "9000" || cfg.publicBaseURL != "https://zazi.cool" || !cfg.trustProxy || cfg.loginLimit != 3 || cfg.maxUploadMB != 8 {
		t.Errorf("config = %+v", cfg)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	valid := map[string]string{
		"DATABASE_URL":        "postgres://home@localhost/home",
		"ADMIN_PASSWORD_HASH": devHash,
	}
	cases := map[string]map[string]string{
		"missing DATABASE_URL":   {"DATABASE_URL": ""},
		"missing password hash":  {"ADMIN_PASSWORD_HASH": ""},
		"plain-text password":    {"ADMIN_PASSWORD_HASH": "home-dev-password"},
		"relative base URL":      {"PUBLIC_BASE_URL": "zazi.cool"},
		"bad port":               {"PORT": "http"},
		"bad TRUST_PROXY":        {"TRUST_PROXY": "yes please"},
		"zero RATE_LIMIT_LOGIN":  {"RATE_LIMIT_LOGIN": "0"},
		"negative MAX_UPLOAD_MB": {"MAX_UPLOAD_MB": "-1"},
	}
	for name, override := range cases {
		vars := maps.Clone(valid)
		maps.Copy(vars, override)
		if _, err := loadConfig(env(vars)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
