package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// reorderArgs tests
// ---------------------------------------------------------------------------

func TestReorderArgs_NoArgs(t *testing.T) {
	// When no arguments are provided, both slices are nil.
	// This is the "no-data landing page" path: input stays "" and
	// main() calls server.ServeInteractiveNoData.
	flags, positional := reorderArgs(nil)
	assert.Nil(t, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_EmptySlice(t *testing.T) {
	flags, positional := reorderArgs([]string{})
	assert.Nil(t, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_PositionalOnly(t *testing.T) {
	// A bare path argument becomes positional — triggers the
	// analyze-and-serve flow in main().
	flags, positional := reorderArgs([]string{"./mypackage"})
	assert.Nil(t, flags)
	assert.Equal(t, []string{"./mypackage"}, positional)
}

func TestReorderArgs_FlagsOnly(t *testing.T) {
	// Only flags, no positional path — input stays "" and main()
	// calls server.ServeInteractiveNoData.
	flags, positional := reorderArgs([]string{"-no-browser", "-port", "9090"})
	assert.Equal(t, []string{"-no-browser", "-port", "9090"}, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_FlagsBeforePositional(t *testing.T) {
	flags, positional := reorderArgs([]string{"-port", "9090", "./pkg"})
	assert.Equal(t, []string{"-port", "9090"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_PositionalBeforeFlags(t *testing.T) {
	// The whole point of reorderArgs: allow positional args before flags.
	flags, positional := reorderArgs([]string{"./pkg", "-port", "9090"})
	assert.Equal(t, []string{"-port", "9090"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_PositionalBetweenFlags(t *testing.T) {
	flags, positional := reorderArgs([]string{"-no-browser", "./pkg", "-port", "9090"})
	assert.Equal(t, []string{"-no-browser", "-port", "9090"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_ValueFlagWithEquals(t *testing.T) {
	// When a value flag uses "=" syntax, the value is part of the same arg.
	flags, positional := reorderArgs([]string{"-output=diagram.md", "./pkg"})
	assert.Equal(t, []string{"-output=diagram.md"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_BooleanFlagDoesNotConsumeNextArg(t *testing.T) {
	// -no-browser is a boolean flag (not in valueFlagSet), so it must
	// NOT consume the following positional argument.
	flags, positional := reorderArgs([]string{"-no-browser", "./pkg"})
	assert.Equal(t, []string{"-no-browser"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_AllValueFlags(t *testing.T) {
	// Exercise every flag that takes a value argument.
	args := []string{
		"-path", "/tmp/repo",
		"-port", "3000",
		"-filter", "github.com/foo",
		"-output", "out.md",
		"-log-file", "app.log",
		"-log-level", "debug",
	}
	flags, positional := reorderArgs(args)
	assert.Equal(t, args, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_HelpFlag(t *testing.T) {
	// -help is treated as a flag (not positional). Go's FlagSet handles it
	// by printing usage and exiting. reorderArgs must not misclassify it.
	flags, positional := reorderArgs([]string{"-help"})
	assert.Equal(t, []string{"-help"}, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_DoubleHyphenHelpFlag(t *testing.T) {
	// --help also starts with "-" so it goes to flags.
	flags, positional := reorderArgs([]string{"--help"})
	assert.Equal(t, []string{"--help"}, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_PathFlagAlternative(t *testing.T) {
	// Using -path flag instead of positional — both end up in flags,
	// positional is empty, but main() reads input from -path.
	flags, positional := reorderArgs([]string{"-path", "./myrepo"})
	assert.Equal(t, []string{"-path", "./myrepo"}, flags)
	assert.Nil(t, positional)
}

func TestReorderArgs_MultiplePositionalArgs(t *testing.T) {
	// Only the first positional arg is used as input in main().
	flags, positional := reorderArgs([]string{"./first", "./second"})
	assert.Nil(t, flags)
	assert.Equal(t, []string{"./first", "./second"}, positional)
}

func TestReorderArgs_IncludeStdlibBoolFlag(t *testing.T) {
	// -include-stdlib is boolean, must not consume next arg.
	flags, positional := reorderArgs([]string{"-include-stdlib", "./pkg"})
	assert.Equal(t, []string{"-include-stdlib"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_IncludeUnexportedBoolFlag(t *testing.T) {
	// -include-unexported is boolean, must not consume next arg.
	flags, positional := reorderArgs([]string{"-include-unexported", "./pkg"})
	assert.Equal(t, []string{"-include-unexported"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_EnrichBoolFlag(t *testing.T) {
	// -enrich is boolean, must not consume next arg.
	flags, positional := reorderArgs([]string{"-enrich", "./pkg"})
	assert.Equal(t, []string{"-enrich"}, flags)
	assert.Equal(t, []string{"./pkg"}, positional)
}

func TestReorderArgs_ComplexMix(t *testing.T) {
	// Realistic invocation: goifaces ./myrepo -port 3000 -no-browser -enrich -output=out.md
	args := []string{"./myrepo", "-port", "3000", "-no-browser", "-enrich", "-output=out.md"}
	flags, positional := reorderArgs(args)
	assert.Equal(t, []string{"-port", "3000", "-no-browser", "-enrich", "-output=out.md"}, flags)
	assert.Equal(t, []string{"./myrepo"}, positional)
}

func TestReorderArgs_ValueFlagAtEnd(t *testing.T) {
	// If a value flag is at the very end with no following arg, it stays
	// as a flag (flag.Parse will report the error).
	flags, positional := reorderArgs([]string{"-port"})
	assert.Equal(t, []string{"-port"}, flags)
	assert.Nil(t, positional)
}

// ---------------------------------------------------------------------------
// parseLogLevel tests
// ---------------------------------------------------------------------------

func TestParseLogLevel_ValidLevels(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := parseLogLevel(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestParseLogLevel_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"Info", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"Error", slog.LevelError},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := parseLogLevel(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestParseLogLevel_Invalid(t *testing.T) {
	_, err := parseLogLevel("trace")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log level")
	assert.Contains(t, err.Error(), "trace")
}

func TestParseLogLevel_Empty(t *testing.T) {
	_, err := parseLogLevel("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log level")
}

// ---------------------------------------------------------------------------
// Behavioral contract tests
//
// main() cannot be unit-tested directly because it calls os.Exit and uses
// global state (os.Args, signal handling). These tests document the
// behavioral contracts by verifying the helper functions that main() uses
// to determine its execution path:
//
// Path 1: No args -> reorderArgs returns nil,nil -> input="" -> ServeInteractiveNoData
// Path 2: Path arg -> reorderArgs returns positional -> input=path -> analyze-and-serve
// Path 3: -help   -> reorderArgs puts it in flags -> FlagSet prints usage, exits
// ---------------------------------------------------------------------------

func TestNoArgsLeadsToEmptyInput(t *testing.T) {
	// Simulates: goifaces (no arguments)
	// reorderArgs produces nil positional, so input stays "".
	// In main(), this triggers server.ServeInteractiveNoData.
	flags, positional := reorderArgs(nil)

	// Determine input the same way main() does.
	input := ""
	if len(positional) > 0 {
		input = positional[0]
	}
	// No -path flag in flags either.
	assert.Empty(t, input, "no args should produce empty input (triggers no-data landing page)")
	assert.Nil(t, flags, "no args should produce nil flags")
}

func TestPathArgLeadsToNonEmptyInput(t *testing.T) {
	// Simulates: goifaces ./mypackage
	// reorderArgs puts "./mypackage" in positional, so input = "./mypackage".
	// In main(), this triggers the full analyze-and-serve flow.
	_, positional := reorderArgs([]string{"./mypackage"})

	input := ""
	if len(positional) > 0 {
		input = positional[0]
	}
	assert.Equal(t, "./mypackage", input,
		"positional path should become input (triggers analyze-and-serve)")
}

func TestOnlyPortFlagLeadsToEmptyInput(t *testing.T) {
	// Simulates: goifaces -port 3000
	// No positional arg and no -path flag value, so input stays "".
	// In main(), this triggers server.ServeInteractiveNoData on port 3000.
	_, positional := reorderArgs([]string{"-port", "3000"})

	input := ""
	if len(positional) > 0 {
		input = positional[0]
	}
	assert.Empty(t, input,
		"only port flag should produce empty input (triggers no-data landing page)")
}

func TestPathFlagLeadsToNonEmptyInputViaFlagParsing(t *testing.T) {
	// Simulates: goifaces -path ./myrepo
	// The -path value ends up in flags (not positional), so main() reads
	// it from the parsed *pathFlag after flag.Parse.
	flags, positional := reorderArgs([]string{"-path", "./myrepo"})

	// positional is empty...
	input := ""
	if len(positional) > 0 {
		input = positional[0]
	}
	assert.Empty(t, input, "positional is empty when using -path flag")

	// ...but flags contains "-path ./myrepo", which flag.Parse will set on *pathFlag.
	assert.Contains(t, flags, "-path")
	assert.Contains(t, flags, "./myrepo")
}
