package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodsLeftAlignCSS(t *testing.T) {
	assert.True(t, strings.Contains(interactiveHTMLTemplate, ".methods-group"),
		"template should contain .methods-group CSS selector")
	assert.True(t, strings.Contains(interactiveHTMLTemplate, "text-align: left !important"),
		"template should contain left-alignment CSS rule for methods")
}

func TestTreemapAlwaysRendersText(t *testing.T) {
	// Leaf nodes must always append tm-name and tm-stats elements without
	// height-gated conditionals so that text is never hidden on small blocks.
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "TREEMAP_GAP) >= 20"),
		"leaf node tm-name should not be gated by height threshold")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "TREEMAP_GAP) >= 35"),
		"leaf node tm-stats should not be gated by height threshold")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "selfH >= 16"),
		"self-node tm-name should not be gated by height threshold")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "selfH >= 31"),
		"self-node tm-stats should not be gated by height threshold")
}

func TestTreemapMinDimensions(t *testing.T) {
	// treemap-node must have min-height and min-width so blocks are always
	// large enough to display at least the name and stats text lines.
	assert.Contains(t, interactiveHTMLTemplate, "min-height: 36px",
		"treemap-node should have min-height to fit both text lines")
	assert.Contains(t, interactiveHTMLTemplate, "min-width: 56px",
		"treemap-node should have min-width for readable text")

	// treemap-group must have min dimensions for its label
	assert.Contains(t, interactiveHTMLTemplate, "min-height: 24px",
		"treemap-group should have min-height for its label")

	// tm-name and tm-stats must not shrink away in the flex container
	assert.Contains(t, interactiveHTMLTemplate, "flex-shrink: 0",
		"text elements should not flex-shrink")

	// On hover, treemap-node should show overflow so full text is visible
	assert.Contains(t, interactiveHTMLTemplate, "overflow: visible",
		"treemap nodes or groups should allow visible overflow on hover")
}
