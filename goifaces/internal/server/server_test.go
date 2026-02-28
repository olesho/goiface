package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLandingPageTemplateExists(t *testing.T) {
	assert.Contains(t, landingHTMLTemplate, `id="repo-path"`,
		"landing page should contain repo-path input")
	assert.Contains(t, landingHTMLTemplate, `id="analyze-btn"`,
		"landing page should contain analyze button")
	assert.Contains(t, landingHTMLTemplate, `id="status"`,
		"landing page should contain status div")
	assert.Contains(t, landingHTMLTemplate, `fetch('/api/load'`,
		"landing page should contain fetch call to /api/load")
	assert.Contains(t, landingHTMLTemplate, `window.location.reload()`,
		"landing page should reload on success")
	assert.Contains(t, landingHTMLTemplate, `function loadRepo()`,
		"landing page should define loadRepo function")
}

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

func TestTreemapGridLayout(t *testing.T) {
	// The treemap container uses CSS Grid for responsive tile layout.
	assert.Contains(t, interactiveHTMLTemplate, "display: grid",
		"treemap-container should use CSS grid layout")
	assert.Contains(t, interactiveHTMLTemplate, "auto-fill",
		"treemap grid should use auto-fill for responsive columns")
	assert.Contains(t, interactiveHTMLTemplate, "gridColumn",
		"tiles should span grid columns based on content size")
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
		".treemap-viewport {\n      flex: 1;\n      overflow: auto;\n      padding: 0.5rem;\n      position: relative;",
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

func TestSharedSelectionStateFromOverlayToSidebar(t *testing.T) {
	// Selecting an item in Package Map overlay must mutate shared state
	// and call updateSelectionUI() to sync the Structures sidebar.

	// Overlay checkbox change handler sets shared state for interfaces
	assert.Contains(t, interactiveHTMLTemplate,
		"selectedIfaceIDs[iface.id] = true;",
		"overlay interface checkbox should set selectedIfaceIDs[iface.id] = true")

	// Overlay checkbox change handler sets shared state for types
	assert.Contains(t, interactiveHTMLTemplate,
		"selectedTypeIDs[t.id] = true;",
		"overlay type checkbox should set selectedTypeIDs[t.id] = true")

	// After mutation, updateSelectionUI() syncs sidebar checkboxes
	assert.Contains(t, interactiveHTMLTemplate,
		`cb.checked = !!selectedTypeIDs[cb.value]`,
		"updateSelectionUI should sync sidebar impl checkboxes from shared state")
	assert.Contains(t, interactiveHTMLTemplate,
		`cb.checked = !!selectedIfaceIDs[cb.value]`,
		"updateSelectionUI should sync sidebar iface checkboxes from shared state")
}

func TestSharedSelectionStateFromSidebarToOverlay(t *testing.T) {
	// Checking a checkbox in Structures sidebar rebuilds shared state
	// and syncs overlay checkboxes when the overlay is open.

	// onSelectionChange rebuilds shared state from sidebar checkboxes
	assert.Contains(t, interactiveHTMLTemplate,
		".impl-cb:checked",
		"onSelectionChange should read checked impl checkboxes to rebuild state")
	assert.Contains(t, interactiveHTMLTemplate,
		".iface-cb:checked",
		"onSelectionChange should read checked iface checkboxes to rebuild state")

	// updateSelectionUI syncs overlay checkboxes when open
	assert.Contains(t, interactiveHTMLTemplate,
		"if (activeOverlay)",
		"updateSelectionUI should check if overlay is open before syncing")
	assert.Contains(t, interactiveHTMLTemplate,
		`cb.getAttribute('data-id')`,
		"overlay sync should read data-id attribute from overlay checkboxes")
	assert.Contains(t, interactiveHTMLTemplate,
		`cb.getAttribute('data-kind')`,
		"overlay sync should read data-kind attribute from overlay checkboxes")

	// Overlay checkboxes initialized from shared state on creation
	assert.Contains(t, interactiveHTMLTemplate,
		"cb.checked = !!selectedIfaceIDs[iface.id]",
		"overlay interface checkboxes should be initialized from shared state")
	assert.Contains(t, interactiveHTMLTemplate,
		"cb.checked = !!selectedTypeIDs[t.id]",
		"overlay type checkboxes should be initialized from shared state")
}

func TestSharedSelectionStateDeselection(t *testing.T) {
	// Deselecting in either tab must update the other.

	// Overlay deselection uses delete to remove from shared state
	assert.Contains(t, interactiveHTMLTemplate,
		"delete selectedIfaceIDs[iface.id]",
		"overlay should use delete to deselect interface from shared state")
	assert.Contains(t, interactiveHTMLTemplate,
		"delete selectedTypeIDs[t.id]",
		"overlay should use delete to deselect type from shared state")

	// Sidebar deselection: onSelectionChange rebuilds from :checked only,
	// so unchecked items are naturally excluded
	assert.Contains(t, interactiveHTMLTemplate,
		"selectedTypeIDs = {};",
		"onSelectionChange should reset selectedTypeIDs before rebuilding")
	assert.Contains(t, interactiveHTMLTemplate,
		"selectedIfaceIDs = {};",
		"onSelectionChange should reset selectedIfaceIDs before rebuilding")
}

func TestSharedSelectionStateBulkActions(t *testing.T) {
	// Bulk actions (All/Clear buttons) must update shared state and
	// Package Map indicators via onSelectionChange().

	// impls-all sets all .impl-cb to checked, then calls onSelectionChange()
	assert.Contains(t, interactiveHTMLTemplate,
		`document.getElementById('impls-all')`,
		"template should have impls-all bulk select button")
	assert.Contains(t, interactiveHTMLTemplate,
		`document.getElementById('impls-clear')`,
		"template should have impls-clear bulk deselect button")
	assert.Contains(t, interactiveHTMLTemplate,
		`document.getElementById('ifaces-all')`,
		"template should have ifaces-all bulk select button")
	assert.Contains(t, interactiveHTMLTemplate,
		`document.getElementById('ifaces-clear')`,
		"template should have ifaces-clear bulk deselect button")

	// Each bulk button handler calls onSelectionChange()
	// impls-all: sets checked=true, calls onSelectionChange
	assert.Contains(t, interactiveHTMLTemplate,
		"document.querySelectorAll('.impl-cb').forEach(function(cb) { cb.checked = true; });\n        onSelectionChange();",
		"impls-all handler should check all impl checkboxes then call onSelectionChange")
	assert.Contains(t, interactiveHTMLTemplate,
		"document.querySelectorAll('.impl-cb').forEach(function(cb) { cb.checked = false; });\n        onSelectionChange();",
		"impls-clear handler should uncheck all impl checkboxes then call onSelectionChange")

	// updateSelectionUI (called via onSelectionChange → updateSelectionUI) calls
	// updatePackageMapHighlights and updatePackageMapBadges
	assert.Contains(t, interactiveHTMLTemplate,
		"updatePackageMapHighlights();\n        updatePackageMapBadges();",
		"updateSelectionUI should call updatePackageMapHighlights and updatePackageMapBadges")
}

func TestSharedSelectionStateTabSwitchPreservation(t *testing.T) {
	// Tab switching must preserve selection state — module-level variables
	// persist naturally, and switchTab must NOT reset them.

	// selectedTypeIDs and selectedIfaceIDs are module-level variables
	assert.Contains(t, interactiveHTMLTemplate,
		"var selectedTypeIDs = {};",
		"selectedTypeIDs should be declared as module-level variable")
	assert.Contains(t, interactiveHTMLTemplate,
		"var selectedIfaceIDs = {};",
		"selectedIfaceIDs should be declared as module-level variable")

	// switchTab must NOT clear selection state
	assert.Contains(t, interactiveHTMLTemplate,
		"function switchTab(tab) {",
		"template should define switchTab function")

	// Extract the switchTab function body and verify it doesn't reset state.
	// The function sets currentTab, toggles CSS classes, and conditionally
	// renders the treemap — but must NOT touch selectedTypeIDs or selectedIfaceIDs.
	switchTabIdx := strings.Index(interactiveHTMLTemplate, "function switchTab(tab) {")
	if switchTabIdx < 0 {
		t.Fatal("switchTab function must exist in the template")
	}
	// Find the next function declaration after switchTab to bound the body
	rest := interactiveHTMLTemplate[switchTabIdx+1:]
	nextFnIdx := strings.Index(rest, "\n      function ")
	if nextFnIdx < 0 {
		nextFnIdx = 1000
	}
	switchTabBody := interactiveHTMLTemplate[switchTabIdx : switchTabIdx+1+nextFnIdx]
	assert.False(t, strings.Contains(switchTabBody, "selectedTypeIDs = {}"),
		"switchTab must NOT reset selectedTypeIDs")
	assert.False(t, strings.Contains(switchTabBody, "selectedIfaceIDs = {}"),
		"switchTab must NOT reset selectedIfaceIDs")
}

func TestSwitchTabStructuresTriggersdiagramUpdate(t *testing.T) {
	// Switching to the "structures" tab must call triggerDiagramUpdate()
	// inside a requestAnimationFrame callback so the diagram is re-rendered
	// with the current selection state.

	// The switchTab function must contain the else-if branch for 'structures'
	assert.Contains(t, interactiveHTMLTemplate,
		`} else if (tab === 'structures') {`,
		"switchTab should have an else-if branch for the structures tab")

	// Extract the switchTab function body to verify the structures branch
	// calls triggerDiagramUpdate inside requestAnimationFrame.
	switchTabIdx := strings.Index(interactiveHTMLTemplate, "function switchTab(tab) {")
	if switchTabIdx < 0 {
		t.Fatal("switchTab function must exist in the template")
	}
	rest := interactiveHTMLTemplate[switchTabIdx+1:]
	nextFnIdx := strings.Index(rest, "\n      function ")
	if nextFnIdx < 0 {
		nextFnIdx = 1000
	}
	switchTabBody := interactiveHTMLTemplate[switchTabIdx : switchTabIdx+1+nextFnIdx]

	// The structures branch must use requestAnimationFrame
	assert.Contains(t, switchTabBody,
		"else if (tab === 'structures') {\n          requestAnimationFrame(function() {\n            triggerDiagramUpdate();\n          });",
		"structures branch should call triggerDiagramUpdate() inside requestAnimationFrame")

	// triggerDiagramUpdate must be called exactly once in the switchTab body
	// (only in the structures branch, not unconditionally)
	count := strings.Count(switchTabBody, "triggerDiagramUpdate()")
	assert.Equal(t, 1, count,
		"triggerDiagramUpdate should be called exactly once in switchTab (in the structures branch)")
}

func TestSharedSelectionStateDiagramReRender(t *testing.T) {
	// Selection changes from Package Map must trigger diagram re-render
	// via the chain: updateSelectionUI() → triggerDiagramUpdate() → buildMermaid.

	// updateSelectionUI calls triggerDiagramUpdate at the end
	assert.Contains(t, interactiveHTMLTemplate,
		"updatingUI = false;\n        triggerDiagramUpdate();",
		"updateSelectionUI should call triggerDiagramUpdate after clearing updatingUI flag")

	// triggerDiagramUpdate calls buildMermaid and renderSelectionDiagram
	assert.Contains(t, interactiveHTMLTemplate,
		"var mermaidSrc = buildMermaid(typeIDs, ifaceIDs)",
		"triggerDiagramUpdate should call buildMermaid with selected IDs")
	assert.Contains(t, interactiveHTMLTemplate,
		"renderSelectionDiagram(mermaidSrc)",
		"triggerDiagramUpdate should call renderSelectionDiagram with mermaid source")

	// When selection is empty, triggerDiagramUpdate shows placeholder
	assert.Contains(t, interactiveHTMLTemplate,
		"showPlaceholder();",
		"triggerDiagramUpdate should show placeholder when selection is empty")
}

func TestPackageMapHasSelectionClass(t *testing.T) {
	// Box with selected items has the fat border class (.tm-has-selection).
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-node.tm-has-selection",
		"CSS selector for tm-has-selection should exist")
	assert.Contains(t, interactiveHTMLTemplate, "border: 3px solid #1976d2",
		"light-theme fat border should be 3px solid #1976d2")
	assert.Contains(t, interactiveHTMLTemplate, "el.classList.add('tm-has-selection')",
		"JS should add tm-has-selection class when package has selections")
	assert.Contains(t, interactiveHTMLTemplate, "function updatePackageMapHighlights()",
		"updatePackageMapHighlights function should exist")
}

func TestPackageMapNoSelectionRemovesClass(t *testing.T) {
	// Box without selected items does NOT have the class (class is removed).
	assert.Contains(t, interactiveHTMLTemplate, "el.classList.remove('tm-has-selection')",
		"JS should remove tm-has-selection class when package has no selections")
	assert.Contains(t, interactiveHTMLTemplate, "activePkgs[pkg]",
		"activePkgs lookup should drive add/remove decision")
}

func TestPackageMapBadgeShowsCorrectCount(t *testing.T) {
	// Count badge shows correct number of selected items per package.
	assert.Contains(t, interactiveHTMLTemplate, "function updatePackageMapBadges()",
		"updatePackageMapBadges function should exist")
	assert.Contains(t, interactiveHTMLTemplate, "badge.textContent = count",
		"badge text should be set to the computed count")
	assert.Contains(t, interactiveHTMLTemplate, "badge.className = 'tm-selection-count'",
		"badge should get the correct CSS class")
	assert.Contains(t, interactiveHTMLTemplate, "if (selectedIfaceIDs[ifaces[i].id]) count++",
		"badge count should include selected interfaces")
	assert.Contains(t, interactiveHTMLTemplate, "if (selectedTypeIDs[types[i].id]) count++",
		"badge count should include selected types")
}

func TestPackageMapBadgeHiddenWhenCountZero(t *testing.T) {
	// Count badge is hidden when count is 0.
	assert.Contains(t, interactiveHTMLTemplate, "badge.style.display = 'none'",
		"badge should be hidden when count drops to 0")
	assert.Contains(t, interactiveHTMLTemplate, "badge.style.display = ''",
		"badge should be shown (display reset) when count > 0")
}

func TestPackageMapSelectionUpdatesIndicatorsRealTime(t *testing.T) {
	// Selecting/deselecting items updates border and badge in real-time.
	assert.Contains(t, interactiveHTMLTemplate, "updatePackageMapHighlights();",
		"updatePackageMapHighlights should be called during UI sync")
	assert.Contains(t, interactiveHTMLTemplate, "updatePackageMapBadges();",
		"updatePackageMapBadges should be called during UI sync")
	assert.Contains(t, interactiveHTMLTemplate, "function updateSelectionUI()",
		"updateSelectionUI orchestrator function should exist")
	assert.Contains(t, interactiveHTMLTemplate, "var updatingUI = false",
		"updatingUI re-entrancy guard variable should be initialized")
}

func TestPackageMapClearAllRemovesHighlightsAndBadges(t *testing.T) {
	// Clearing all selections removes all highlights and badges.
	assert.Contains(t, interactiveHTMLTemplate,
		"document.querySelectorAll('.treemap-node[data-pkgpath]').forEach",
		"both highlight and badge functions should iterate all treemap nodes")
	assert.Equal(t, 2,
		strings.Count(interactiveHTMLTemplate, "document.querySelectorAll('.treemap-node[data-pkgpath]').forEach"),
		"querySelectorAll on treemap nodes should appear exactly twice (highlights + badges)")
}

func TestPackageMapBadgeCSSStyle(t *testing.T) {
	// Badge CSS styling is correct for both light and dark themes.
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-node .tm-selection-count",
		"CSS selector for badge should exist")
	assert.Contains(t, interactiveHTMLTemplate, ".treemap-node .tm-selection-count {\n      position: absolute;",
		"badge should be absolutely positioned within its CSS block")
	assert.Contains(t, interactiveHTMLTemplate, "top: 2px",
		"badge should be anchored 2px from top")
	assert.Contains(t, interactiveHTMLTemplate, "right: 2px",
		"badge should be anchored 2px from right")
	assert.Contains(t, interactiveHTMLTemplate, "pointer-events: none;\n      z-index: 5;",
		"badge should not intercept clicks (anchored to badge z-index context)")
	assert.Contains(t, interactiveHTMLTemplate, "z-index: 5",
		"badge should render above node content")
}

func TestSharedSelectionStateReEntrancyGuard(t *testing.T) {
	// Re-entrancy guard prevents infinite loops between overlay and sidebar sync.

	// updatingUI flag is declared
	assert.Contains(t, interactiveHTMLTemplate,
		"var updatingUI = false;",
		"template should declare updatingUI re-entrancy guard variable")

	// updateSelectionUI sets updatingUI = true at start
	assert.Contains(t, interactiveHTMLTemplate,
		"updatingUI = true;",
		"updateSelectionUI should set updatingUI = true at the start")

	// updateSelectionUI clears updatingUI before triggerDiagramUpdate
	assert.Contains(t, interactiveHTMLTemplate,
		"updatingUI = false;",
		"updateSelectionUI should clear updatingUI before calling triggerDiagramUpdate")

	// onSelectionChange checks the guard and returns early
	assert.Contains(t, interactiveHTMLTemplate,
		"if (updatingUI) return;",
		"onSelectionChange should return early when updatingUI is true")
}

func TestSwitchTabPkgMapHTMLRefreshesHighlightsAndBadges(t *testing.T) {
	// When switching back to the Package Map tab (already rendered),
	// switchTab must call updatePackageMapHighlights() and
	// updatePackageMapBadges() inside a requestAnimationFrame so that
	// visual indicators (borders + count badges) reflect the current
	// selection state that may have changed while on another tab.

	// The switchTab function must contain the else-if branch for 'pkgmap-html'
	// that handles the already-rendered case (distinct from the initial render).
	assert.Contains(t, interactiveHTMLTemplate,
		`} else if (tab === 'pkgmap-html') {`,
		"switchTab should have an else-if branch for pkgmap-html (already rendered case)")

	// Extract the switchTab function body for focused assertions.
	switchTabIdx := strings.Index(interactiveHTMLTemplate, "function switchTab(tab) {")
	if switchTabIdx < 0 {
		t.Fatal("switchTab function must exist in the template")
	}
	rest := interactiveHTMLTemplate[switchTabIdx+1:]
	nextFnIdx := strings.Index(rest, "\n      function ")
	if nextFnIdx < 0 {
		nextFnIdx = 1000
	}
	switchTabBody := interactiveHTMLTemplate[switchTabIdx : switchTabIdx+1+nextFnIdx]

	// The already-rendered pkgmap-html branch must use requestAnimationFrame
	// and call both updatePackageMapHighlights and updatePackageMapBadges.
	assert.Contains(t, switchTabBody,
		"else if (tab === 'pkgmap-html') {\n          requestAnimationFrame(function() {\n            updatePackageMapHighlights();\n            updatePackageMapBadges();\n          });",
		"pkgmap-html already-rendered branch should call updatePackageMapHighlights and updatePackageMapBadges inside requestAnimationFrame")
}

func TestSwitchTabPkgMapHTMLBranchIsSeparateFromInitialRender(t *testing.T) {
	// The initial render branch checks !pkgMapHtmlRendered and calls
	// layoutTreemap(). The re-visit branch must be a separate else-if
	// that only refreshes highlights and badges, NOT re-layout.

	// Extract the switchTab function body.
	switchTabIdx := strings.Index(interactiveHTMLTemplate, "function switchTab(tab) {")
	if switchTabIdx < 0 {
		t.Fatal("switchTab function must exist in the template")
	}
	rest := interactiveHTMLTemplate[switchTabIdx+1:]
	nextFnIdx := strings.Index(rest, "\n      function ")
	if nextFnIdx < 0 {
		nextFnIdx = 1000
	}
	switchTabBody := interactiveHTMLTemplate[switchTabIdx : switchTabIdx+1+nextFnIdx]

	// The initial render branch must guard with !pkgMapHtmlRendered
	assert.Contains(t, switchTabBody,
		"if (tab === 'pkgmap-html' && !pkgMapHtmlRendered) {",
		"initial render branch should check !pkgMapHtmlRendered")

	// The initial render branch calls layoutTreemap, not the highlight/badge functions
	assert.Contains(t, switchTabBody,
		"layoutTreemap();\n            pkgMapHtmlRendered = true;",
		"initial render branch should call layoutTreemap and set pkgMapHtmlRendered")

	// The re-visit branch must NOT call layoutTreemap
	// Find the else-if branch for pkgmap-html and verify it does not contain layoutTreemap
	elseIfIdx := strings.Index(switchTabBody, "} else if (tab === 'pkgmap-html') {")
	assert.Greater(t, elseIfIdx, 0,
		"else-if pkgmap-html branch must exist after the initial render branch")

	// Get the text from the else-if branch to the next else-if or closing brace
	elseIfRest := switchTabBody[elseIfIdx:]
	nextElseIdx := strings.Index(elseIfRest[1:], "} else if")
	if nextElseIdx < 0 {
		nextElseIdx = len(elseIfRest) - 1
	} else {
		nextElseIdx++ // adjust for the [1:] offset
	}
	elseIfBranch := elseIfRest[:nextElseIdx]

	assert.NotContains(t, elseIfBranch, "layoutTreemap()",
		"re-visit pkgmap-html branch must NOT call layoutTreemap — that is only for initial render")
	assert.NotContains(t, elseIfBranch, "pkgMapHtmlRendered",
		"re-visit pkgmap-html branch must NOT reference pkgMapHtmlRendered")
	assert.Contains(t, elseIfBranch, "updatePackageMapHighlights()",
		"re-visit pkgmap-html branch must call updatePackageMapHighlights")
	assert.Contains(t, elseIfBranch, "updatePackageMapBadges()",
		"re-visit pkgmap-html branch must call updatePackageMapBadges")
}
