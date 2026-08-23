package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

const silentLevel = slog.Level(100)

type Settings struct {
	Enabled   bool
	Format    string
	Level     slog.Level
	AddSource bool
	Color     bool
}

// Configure installs the process-wide slog logger. Logging is intentionally
// disabled by default; passing a level override or setting SPECVIEW_LOG_LEVEL
// enables it. A CLI-provided override takes precedence over the environment.
func Configure(version string, levelOverrides ...string) (*slog.Logger, Settings) {
	settings := settingsFromEnv(levelOverrides...)

	var handler slog.Handler
	switch {
	case !settings.Enabled:
		handler = slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: silentLevel})
	case settings.Format == "json":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:     settings.Level,
			AddSource: settings.AddSource,
		})
	default:
		handler = tint.NewTextHandler(os.Stderr, &tint.Options{
			Level:      settings.Level,
			AddSource:  settings.AddSource,
			TimeFormat: time.TimeOnly,
			NoColor:    !settings.Color,
		})
	}

	logger := slog.New(handler).With(
		"app", "specview",
		"version", version,
	)
	slog.SetDefault(logger)
	logger.Debug("logging configured",
		"enabled", settings.Enabled,
		"format", settings.Format,
		"level", settings.Level.String(),
		"source", settings.AddSource,
		"color", settings.Color,
	)
	return logger, settings
}

func settingsFromEnv(levelOverrides ...string) Settings {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("SPECVIEW_LOG_FORMAT")))
	if format == "" {
		if strings.EqualFold(os.Getenv("SPECVIEW_ENV"), "production") {
			format = "json"
		} else {
			format = "console"
		}
	}
	if format != "json" {
		format = "console"
	}

	levelText := strings.TrimSpace(os.Getenv("SPECVIEW_LOG_LEVEL"))
	if len(levelOverrides) > 0 {
		if override := strings.TrimSpace(levelOverrides[0]); override != "" {
			levelText = override
		}
	}

	enabled := false
	level := slog.LevelInfo
	if levelText != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(levelText)); err == nil {
			enabled = true
			level = parsed
		}
	}

	addSource := enabled && format == "console" && level <= slog.LevelDebug
	if value := strings.TrimSpace(os.Getenv("SPECVIEW_LOG_SOURCE")); value != "" {
		addSource = parseBool(value, addSource)
	}

	color := enabled && format == "console" && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if value := strings.TrimSpace(os.Getenv("SPECVIEW_LOG_COLOR")); value != "" {
		color = parseBool(value, color)
	}

	return Settings{
		Enabled:   enabled,
		Format:    format,
		Level:     level,
		AddSource: addSource,
		Color:     color,
	}
}

func parseBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "always":
		return true
	case "0", "false", "no", "off", "never":
		return false
	default:
		return fallback
	}
}
