package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

type Settings struct {
	Format    string
	Level     slog.Level
	AddSource bool
	Color     bool
}

func Configure(version string) (*slog.Logger, Settings) {
	settings := settingsFromEnv()

	var handler slog.Handler
	if settings.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:     settings.Level,
			AddSource: settings.AddSource,
		})
	} else {
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
		"format", settings.Format,
		"level", settings.Level.String(),
		"source", settings.AddSource,
		"color", settings.Color,
	)
	return logger, settings
}

func settingsFromEnv() Settings {
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

	level := slog.LevelDebug
	if format == "json" {
		level = slog.LevelInfo
	}
	if value := strings.TrimSpace(os.Getenv("SPECVIEW_LOG_LEVEL")); value != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(value)); err == nil {
			level = parsed
		}
	}

	addSource := format == "console"
	if value := strings.TrimSpace(os.Getenv("SPECVIEW_LOG_SOURCE")); value != "" {
		addSource = parseBool(value, addSource)
	}

	color := format == "console" && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if value := strings.TrimSpace(os.Getenv("SPECVIEW_LOG_COLOR")); value != "" {
		color = parseBool(value, color)
	}

	return Settings{
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
