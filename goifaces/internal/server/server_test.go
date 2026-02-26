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

func TestTreemapDepthConditionalTextContent(t *testing.T) {
	// The renderTreemap function must use depth-conditional logic when
	// setting textContent for both self-nodes and leaf-nodes:
	//   depth > 0  -> use d.name       (short basename inside nested groups)
	//   depth == 0 -> use d.relPath     (full relative path at top level)

	// Both the self-node (sn) and leaf-node (nameEl) assignments must
	// contain the ternary expression.
	depthTernary := "depth > 0 ? d.name : (d.relPath || d.name)"

	occurrences := strings.Count(interactiveHTMLTemplate, depthTernary)
	assert.Equal(t, 2, occurrences,
		"depth-conditional text logic should appear exactly twice "+
			"(once for self-node, once for leaf-node)")

	// Self-node: sn.textContent uses depth check
	assert.Contains(t, interactiveHTMLTemplate,
		"sn.textContent = "+depthTernary,
		"self-node tm-name should use depth-conditional relPath/name")

	// Leaf-node: nameEl.textContent uses depth check
	assert.Contains(t, interactiveHTMLTemplate,
		"nameEl.textContent = "+depthTernary,
		"leaf-node tm-name should use depth-conditional relPath/name")

	// The old unconditional pattern must NOT be present. Before this change
	// both assignments were simply: textContent = d.relPath || d.name
	assert.False(t, strings.Contains(interactiveHTMLTemplate,
		"textContent = d.relPath || d.name;"),
		"unconditional relPath assignment should no longer exist — "+
			"depth check is required")
}

func TestTreemapClickableNodes(t *testing.T) {
	// Treemap nodes with interfaces/types should be clickable to show an overlay.
	assert.Contains(t, interactiveHTMLTemplate, "data-clickable",
		"template should set data-clickable attribute on interactive treemap nodes")
	assert.True(t, strings.Contains(interactiveHTMLTemplate, ".treemap-node[data-clickable]"),
		"template should contain CSS selector for clickable treemap nodes")
	assert.True(t, strings.Contains(interactiveHTMLTemplate, "cursor: pointer"),
		"clickable treemap nodes should have cursor: pointer style")
	assert.Contains(t, interactiveHTMLTemplate, "function showPackageOverlay",
		"template should define showPackageOverlay function")
}

func TestTreemapOverlayCSS(t *testing.T) {
	// The overlay that shows interfaces/types for a clicked package node
	// must have proper CSS styling.
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-overlay",
		"template should contain .treemap-overlay CSS class")
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-overlay-header",
		"template should contain .treemap-overlay-header CSS class")
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-overlay-section",
		"template should contain .treemap-overlay-section CSS class")
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-overlay-item",
		"template should contain .treemap-overlay-item CSS class")
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-node.tm-selected",
		"template should contain .treemap-node.tm-selected CSS class for selected state")
	assert.Contains(t, interactiveHTMLTemplate, "function dismissOverlay",
		"template should define dismissOverlay function")
}

func TestTreemapPkgLookupMaps(t *testing.T) {
	// The template must build pkgInterfaces and pkgTypes lookup maps
	// so the overlay can find interfaces/types by package path.
	assert.Contains(t, interactiveHTMLTemplate, "var pkgInterfaces = {}",
		"template should initialize pkgInterfaces lookup map")
	assert.Contains(t, interactiveHTMLTemplate, "var pkgTypes = {}",
		"template should initialize pkgTypes lookup map")
	assert.True(t, strings.Contains(interactiveHTMLTemplate, "pkgInterfaces[iface.pkgPath]"),
		"template should populate pkgInterfaces by iface.pkgPath")
	assert.True(t, strings.Contains(interactiveHTMLTemplate, "pkgTypes[t.pkgPath]"),
		"template should populate pkgTypes by t.pkgPath")
}

func TestTreemapMinDimensions(t *testing.T) {
	// treemap-node must have min-height and min-width so blocks are always
	// large enough to display at least the name and stats text lines.
	assert.Contains(t, interactiveHTMLTemplate, "min-height: 36px",
		"treemap-node should have min-height to fit both text lines")
	assert.Contains(t, interactiveHTMLTemplate, "min-width: 80px",
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

func TestTreemapVerticalStackFallback(t *testing.T) {
	// The renderTreemap function must detect when squarify produces
	// children narrower than MIN_NODE_WIDTH and fall back to vertical stacking.
	assert.Contains(t, interactiveHTMLTemplate, "MIN_NODE_WIDTH",
		"template should define MIN_NODE_WIDTH constant")
	assert.Contains(t, interactiveHTMLTemplate, "verticalStack",
		"template should contain verticalStack fallback function")
	assert.Contains(t, interactiveHTMLTemplate, "needsVerticalStack",
		"template should check for overflow and trigger vertical stacking")
}

func TestTreemapOverlayNoMaxWidth(t *testing.T) {
	// The .treemap-overlay CSS must NOT contain max-width: 400px because the
	// overlay width is now set dynamically in JS to match the clicked box.
	// It should still have min-width: 200px as a floor.
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "max-width: 400px"),
		".treemap-overlay should not have max-width: 400px — width is set dynamically in JS")
	assert.Contains(t, interactiveHTMLTemplate, "min-width: 200px",
		".treemap-overlay should still have min-width: 200px as a minimum width floor")
}

func TestTreemapOverlayPositionsBelowClickedBox(t *testing.T) {
	// The showPackageOverlay function must position the overlay BELOW the
	// clicked box (not to the right). This means:
	//   left uses nodeRect.left (left-aligned with box, not nodeRect.right)
	//   top uses nodeRect.bottom (below box, not nodeRect.top)

	// Left positioning: must use nodeRect.left, not nodeRect.right
	assert.Contains(t, interactiveHTMLTemplate,
		"var left = nodeRect.left - vpRect.left + viewport.scrollLeft",
		"overlay left should be computed from nodeRect.left (left-aligned with box)")
	assert.False(t, strings.Contains(interactiveHTMLTemplate, "nodeRect.right - vpRect.left"),
		"overlay left should NOT use nodeRect.right — overlay goes below, not to the right")

	// Top positioning: must use nodeRect.bottom, not nodeRect.top for placement
	assert.Contains(t, interactiveHTMLTemplate,
		"var top = nodeRect.bottom - vpRect.top + viewport.scrollTop + 4",
		"overlay top should be computed from nodeRect.bottom with 4px gap")
}

func TestTreemapOverlayDynamicWidth(t *testing.T) {
	// The overlay width must be set dynamically to match the clicked box width,
	// with a minimum of 200px, using Math.max(200, nodeRect.width).
	assert.Contains(t, interactiveHTMLTemplate,
		"overlay.style.width = Math.max(200, nodeRect.width) + 'px'",
		"overlay width should be set to Math.max(200, nodeRect.width)")
}

func TestTreemapOverlayFlipAboveWhenNoSpaceBelow(t *testing.T) {
	// When spaceBelow <= 0 the overlay flips above the clicked node.
	// Verify the flip-above coordinate computation.
	assert.Contains(t, interactiveHTMLTemplate,
		"top = nodeRect.top - vpRect.top + viewport.scrollTop - 4",
		"flip-above top should be computed from nodeRect.top with 4px gap above")
	assert.Contains(t, interactiveHTMLTemplate,
		"overlay.offsetHeight",
		"flip-above branch must measure rendered overlay height")
	assert.Contains(t, interactiveHTMLTemplate,
		"top = top - oh",
		"flip-above must shift overlay upward by its rendered height")
}

func TestTreemapOverlayTopEdgeClamping(t *testing.T) {
	// When the flipped-above overlay overflows the top edge (top < 0),
	// maxHeight is shrunk and top is pinned to 0.
	assert.Contains(t, interactiveHTMLTemplate,
		"overlay.style.maxHeight = (oh + top) + 'px'",
		"when top < 0, maxHeight should shrink to (oh + top) to fit available space")
	assert.Contains(t, interactiveHTMLTemplate,
		"top = 0;",
		"top should be pinned to 0 after maxHeight clamping")
}

func TestTreemapOverlayCSSPositioning(t *testing.T) {
	// Verify the CSS positioning foundation that makes absolute overlay
	// positioning work within the viewport container.
	assert.Contains(t, interactiveHTMLTemplate,
		".treemap-viewport {\n      flex: 1;\n      overflow: hidden;\n      padding: 0.5rem;\n      position: relative;",
		".treemap-viewport must have position: relative to establish containing block")
	assert.Contains(t, interactiveHTMLTemplate,
		".treemap-overlay {\n      position: absolute;",
		".treemap-overlay must have position: absolute for left/top positioning")
	assert.Contains(t, interactiveHTMLTemplate,
		"z-index: 50",
		".treemap-overlay must have z-index: 50 to render above treemap nodes")
}

func TestTreemapOverlayDefaultMaxHeight(t *testing.T) {
	// Verify the CSS default max-height and the JS threshold that
	// triggers dynamic override.
	assert.Contains(t, interactiveHTMLTemplate,
		"max-height: 300px",
		".treemap-overlay CSS should set default max-height: 300px")
	assert.Contains(t, interactiveHTMLTemplate,
		"spaceBelow < 300",
		"JS threshold for maxHeight override should match the CSS default of 300px")
}

func TestTreemapOverlayViewportOverflowClamping(t *testing.T) {
	// When the overlay would extend past the viewport bottom, the JS must
	// clamp max-height using spaceBelow so the overlay stays within bounds.
	assert.Contains(t, interactiveHTMLTemplate, "spaceBelow",
		"overlay positioning should calculate spaceBelow for viewport clamping")
	assert.Contains(t, interactiveHTMLTemplate,
		"var spaceBelow = vpRect.height - (nodeRect.bottom - vpRect.top) - 8",
		"spaceBelow should be computed from viewport height minus overlay top offset")
	assert.Contains(t, interactiveHTMLTemplate,
		"overlay.style.maxHeight = Math.max(80, spaceBelow) + 'px'",
		"overlay maxHeight should be clamped to at least 80px when space is limited")
}
