package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/olehluchkiv/goifaces/internal/analyzer"
	"github.com/olehluchkiv/goifaces/internal/diagram"
	"github.com/olehluchkiv/goifaces/internal/enricher"
	"github.com/olehluchkiv/goifaces/internal/resolver"
)

// AnalysisConfig holds parameters for the analysis pipeline.
type AnalysisConfig struct {
	Input             string
	Filter            string
	IncludeStdlib     bool
	IncludeUnexported bool
}

// RunAnalysis executes the full resolve → analyze → filter → enrich → prepare
// pipeline and returns interactive data ready for the UI.
func RunAnalysis(ctx context.Context, cfg AnalysisConfig, logger *slog.Logger) (diagram.InteractiveData, func(), error) {
	logger = logger.With("component", "analysis")

	// Step 1: Resolve input to local directory.
	logger.Info("resolving input", "input", cfg.Input)
	dir, cleanup, err := resolver.Resolve(ctx, cfg.Input, logger)
	if err != nil {
		return diagram.InteractiveData{}, func() {}, fmt.Errorf("resolve: %w", err)
	}

	// Step 2: Analyze packages.
	logger.Info("analyzing packages", "dir", dir)
	opts := analyzer.AnalyzeOptions{
		Filter:            cfg.Filter,
		IncludeStdlib:     cfg.IncludeStdlib,
		IncludeUnexported: cfg.IncludeUnexported,
	}
	result, err := analyzer.Analyze(ctx, dir, opts, logger)
	if err != nil {
		cleanup()
		return diagram.InteractiveData{}, func() {}, fmt.Errorf("analyze: %w", err)
	}

	// Step 3: Filter results.
	result = analyzer.Filter(result, opts)

	logger.Info("analysis complete",
		"interfaces", len(result.Interfaces),
		"types", len(result.Types),
		"relations", len(result.Relations))

	// Step 4: Apply default enrichers.
	enrichers := []enricher.Enricher{
		enricher.NewDefaultGrouper(),
		enricher.NewDefaultSimplifier(),
	}
	for _, e := range enrichers {
		result = e.Enrich(result)
	}

	// Step 5: Prepare interactive data.
	diagramOpts := diagram.DefaultDiagramOptions()
	data := diagram.PrepareInteractiveData(result, diagramOpts)
	data.PackageMapNodes = diagram.PreparePackageMapData(result)
	data.RepoAddress = cfg.Input

	return data, cleanup, nil
}
