package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"time"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/diagram"
	"github.com/olehluchkiv/goifaces/internal/enricher"
	"github.com/olehluchkiv/goifaces/internal/enricher/llm"
	"github.com/olehluchkiv/goifaces/internal/logging"
	"github.com/olehluchkiv/goifaces/internal/resolver"
	"github.com/olehluchkiv/goifaces/internal/server"
)

func main() {
	// Use a custom FlagSet so we can parse all args regardless of position.
	// Go's default flag.Parse stops at the first non-flag argument, which
	// breaks "goifaces ./path -output file.md". We reorder args so flags
	// come first, then positional args.
	flags, positional := reorderArgs(os.Args[1:])

	fs := flag.NewFlagSet("goifaces", flag.ExitOnError)
	pathFlag := fs.String("path", "", "path or GitHub URL to analyze (alternative to positional argument)")
	port := fs.Int("port", 8080, "HTTP server port")
	filter := fs.String("filter", "", "package path prefix filter")
	includeStdlib := fs.Bool("include-stdlib", false, "include standard library interfaces")
	includeUnexported := fs.Bool("include-unexported", false, "include unexported types and interfaces")
	output := fs.String("output", "", "write Mermaid diagram to file instead of serving")
	noBrowser := fs.Bool("no-browser", false, "skip auto-opening browser")
	logFile := fs.String("log-file", "logs/goifaces.log", "log file path")
	logLevel := fs.String("log-level", "info", "log level (debug, info, warn, error)")
	enrichFlag := fs.Bool("enrich", false, "enable LLM-backed enrichment (requires GOIFACES_LLM_API_KEY env var)")

	if err := fs.Parse(flags); err != nil {
		os.Exit(1)
	}
	// Collect any remaining args from flag parsing + our positional args
	positional = append(positional, fs.Args()...)

	// Determine input: positional argument takes precedence, then -path flag
	input := ""
	if len(positional) > 0 {
		input = positional[0]
	}
	if input == "" {
		input = *pathFlag
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "Usage: goifaces [flags] <path-or-url>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Parse log level
	level, err := parseLogLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level %q: %v\n", *logLevel, err)
		os.Exit(1)
	}

	// Setup logging
	logger, logCleanup, err := logging.Setup(*logFile, level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	defer logCleanup()

	// Setup signal handling with context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Step 1: Resolve input to local directory
	fmt.Println("Resolving input...")
	dir, resolverCleanup, err := resolver.Resolve(ctx, input, logger)
	if err != nil {
		logger.Error("failed to resolve input", "error", err)
		fmt.Fprintf(os.Stderr, "Error resolving input: %v\n", err)
		os.Exit(1)
	}
	defer resolverCleanup()

	// Step 2: Analyze
	fmt.Println("Loading packages...")
	opts := analyzer.AnalyzeOptions{
		Filter:            *filter,
		IncludeStdlib:     *includeStdlib,
		IncludeUnexported: *includeUnexported,
	}

	result, err := analyzer.Analyze(ctx, dir, opts, logger)
	if err != nil {
		logger.Error("analysis failed", "error", err)
		fmt.Fprintf(os.Stderr, "Error analyzing packages: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Filter
	result = analyzer.Filter(result, opts)

	fmt.Printf("Found %d interfaces, %d types, %d relationships\n",
		len(result.Interfaces), len(result.Types), len(result.Relations))

	if len(result.Interfaces) == 0 && len(result.Types) == 0 {
		fmt.Println("No interfaces or implementations found — nothing to diagram.")
		os.Exit(0)
	}

	// Step 4: Run enricher pipeline
	var enrichers []enricher.Enricher
	if *enrichFlag {
		llmClient, llmErr := buildLLMClient(logger)
		if llmErr != nil {
			logger.Error("failed to configure LLM client", "error", llmErr)
			fmt.Fprintf(os.Stderr, "Error: %v\n", llmErr)
			os.Exit(1)
		}
		fmt.Println("LLM enrichment enabled")
		enrichers = []enricher.Enricher{
			enricher.NewLLMGrouper(ctx, llmClient, enricher.NewDefaultGrouper(), logger),
			enricher.NewLLMSimplifier(ctx, llmClient, enricher.NewDefaultSimplifier(), logger),
		}
	} else {
		enrichers = []enricher.Enricher{
			enricher.NewDefaultGrouper(),
			enricher.NewDefaultSimplifier(),
		}
	}
	for _, e := range enrichers {
		result = e.Enrich(result)
	}

	// Step 5: Generate Mermaid diagram
	diagramOpts := diagram.DefaultDiagramOptions()

	// Step 6: Output or serve
	if *output != "" {
		// File output: include %%{init:}%% for standalone .mmd rendering
		diagramOpts.IncludeInit = true
		mermaidContent := diagram.GenerateMermaid(result, diagramOpts)
		if err := os.WriteFile(*output, []byte(mermaidContent), 0o644); err != nil {
			logger.Error("failed to write output file", "error", err)
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote diagram to %s\n", *output)
	} else {
		// Server mode: interactive tabbed UI
		interactiveData := diagram.PrepareInteractiveData(result, diagramOpts)
		interactiveData.PackageMapNodes = diagram.PreparePackageMapData(result)
		interactiveData.RepoAddress = input

		openBrowser := !*noBrowser
		fmt.Printf("Starting server on http://localhost:%d\n", *port)
		if err := server.ServeInteractive(ctx, interactiveData, *port, openBrowser, logger); err != nil {
			logger.Error("server error", "error", err)
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}

// reorderArgs separates flags and positional arguments so flags can appear
// in any position (before or after the positional path argument).
// Flags that take a value (e.g., -output file.md) consume the next arg.
func reorderArgs(args []string) (flags, positional []string) {
	// Set of flags that take a value argument
	valueFlagSet := map[string]bool{
		"-path": true, "-port": true, "-filter": true,
		"-output": true, "-log-file": true, "-log-level": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if this flag takes a value (and it's not using = syntax)
			if !strings.Contains(arg, "=") && valueFlagSet[arg] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return flags, positional
}

func buildLLMClient(logger *slog.Logger) (*llm.Client, error) {
	endpoint := os.Getenv("GOIFACES_LLM_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	apiKey := os.Getenv("GOIFACES_LLM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOIFACES_LLM_API_KEY environment variable is required when --enrich is enabled")
	}
	model := os.Getenv("GOIFACES_LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	cfg := llm.Config{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Model:    model,
		Timeout:  30 * time.Second,
	}
	return llm.NewClient(cfg, logger), nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s (valid: debug, info, warn, error)", s)
	}
}
