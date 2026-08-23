package main

import (
	"reflect"
	"testing"
)

func TestParseCLIOptionsLoggingFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantArgs  []string
		wantLevel string
	}{
		{name: "default", args: nil, wantArgs: nil},
		{name: "verbose", args: []string{"--verbose"}, wantArgs: nil, wantLevel: "info"},
		{name: "debug before command", args: []string{"--debug", "doctor"}, wantArgs: []string{"doctor"}, wantLevel: "debug"},
		{name: "debug after command", args: []string{"serve", "--debug"}, wantArgs: []string{"serve"}, wantLevel: "debug"},
		{name: "explicit level", args: []string{"--log-level=warn", "serve"}, wantArgs: []string{"serve"}, wantLevel: "warn"},
		{name: "explicit level separate", args: []string{"serve", "--log-level", "error"}, wantArgs: []string{"serve"}, wantLevel: "error"},
		{name: "version short flag preserved", args: []string{"-v"}, wantArgs: []string{"-v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCLIOptions(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.args, tt.wantArgs) {
				t.Fatalf("args = %#v, want %#v", got.args, tt.wantArgs)
			}
			if got.logLevel != tt.wantLevel {
				t.Fatalf("log level = %q, want %q", got.logLevel, tt.wantLevel)
			}
		})
	}
}

func TestParseCLIOptionsRejectsInvalidLogLevel(t *testing.T) {
	if _, err := parseCLIOptions([]string{"--log-level", "trace"}); err == nil {
		t.Fatal("invalid log level must fail")
	}
}

func TestParseCLIOptionsRequiresLogLevelValue(t *testing.T) {
	if _, err := parseCLIOptions([]string{"--log-level"}); err == nil {
		t.Fatal("missing log level value must fail")
	}
}
