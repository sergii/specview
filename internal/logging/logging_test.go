package logging

import (
	"log/slog"
	"testing"
)

func TestSettingsFromEnvDisablesLogsByDefault(t *testing.T) {
	clearLoggingEnv(t)

	settings := settingsFromEnv()
	if settings.Enabled {
		t.Fatal("logging must be disabled by default")
	}
	if settings.Format != "console" {
		t.Fatalf("format = %q, want console", settings.Format)
	}
}

func TestSettingsFromEnvLevelEnablesLogging(t *testing.T) {
	clearLoggingEnv(t)
	t.Setenv("SPECVIEW_LOG_LEVEL", "info")

	settings := settingsFromEnv()
	if !settings.Enabled {
		t.Fatal("SPECVIEW_LOG_LEVEL must enable logging")
	}
	if settings.Level != slog.LevelInfo {
		t.Fatalf("level = %s, want INFO", settings.Level)
	}
	if settings.AddSource {
		t.Fatal("info logging should not add source locations by default")
	}
}

func TestSettingsFromEnvCLIOverrideWinsAndDebugAddsSource(t *testing.T) {
	clearLoggingEnv(t)
	t.Setenv("SPECVIEW_LOG_LEVEL", "warn")

	settings := settingsFromEnv("debug")
	if !settings.Enabled {
		t.Fatal("CLI level override must enable logging")
	}
	if settings.Level != slog.LevelDebug {
		t.Fatalf("level = %s, want DEBUG", settings.Level)
	}
	if !settings.AddSource {
		t.Fatal("debug console logging should add source locations by default")
	}
}

func TestProductionEnvironmentDoesNotEnableLogsByItself(t *testing.T) {
	clearLoggingEnv(t)
	t.Setenv("SPECVIEW_ENV", "production")

	settings := settingsFromEnv()
	if settings.Enabled {
		t.Fatal("SPECVIEW_ENV must not enable runtime logs by itself")
	}
	if settings.Format != "json" {
		t.Fatalf("format = %q, want json", settings.Format)
	}
}

func clearLoggingEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"SPECVIEW_LOG_FORMAT",
		"SPECVIEW_LOG_LEVEL",
		"SPECVIEW_LOG_SOURCE",
		"SPECVIEW_LOG_COLOR",
		"SPECVIEW_ENV",
		"NO_COLOR",
		"TERM",
	} {
		t.Setenv(name, "")
	}
}
