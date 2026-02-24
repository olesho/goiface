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
	// The CSS overflow:hidden + text-overflow:ellipsis handles clipping.
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "TREEMAP_GAP) >= 20"),
		"leaf node tm-name should not be gated by height threshold")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "TREEMAP_GAP) >= 35"),
		"leaf node tm-stats should not be gated by height threshold")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "selfH >= 16"),
		"self-node tm-name should not be gated by height threshold")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "selfH >= 31"),
		"self-node tm-stats should not be gated by height threshold")

	// Verify the treemap node CSS still has overflow:hidden to clip gracefully
	assert.True(t, strings.Contains(interactiveHTMLTemplate, ".treemap-node {"),
		"template should contain treemap-node CSS")
	assert.True(t, strings.Contains(interactiveHTMLTemplate, "overflow: hidden"),
		"treemap-node should have overflow:hidden for text clipping")
}
