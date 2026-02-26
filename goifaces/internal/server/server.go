package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/olehluchkiv/goifaces/internal/diagram"
)

const interactiveHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>goifaces — {{.RepoAddress}}</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      display: flex;
      flex-direction: column;
      min-height: 100vh;
      padding: 1rem;
      transition: background-color 0.3s, color 0.3s;
      background-color: #f8f9fa;
      color: #212529;
    }

    @media (prefers-color-scheme: dark) {
      body {
        background-color: #1a1a2e;
        color: #e0e0e0;
      }
      .tab-bar button {
        background-color: #2d2d44;
        color: #e0e0e0;
        border-color: #444;
      }
      .tab-bar button:hover {
        background-color: #3d3d5c;
      }
      .tab-bar button.active {
        background-color: #3d3d5c;
        border-bottom-color: #7c8dff;
      }
      .controls button {
        background-color: #2d2d44;
        color: #e0e0e0;
        border-color: #444;
      }
      .controls button:hover {
        background-color: #3d3d5c;
      }
      .entity-list {
        background-color: #2d2d44;
        border-color: #444;
      }
      .entity-list label:hover,
      .sidebar-section-body label:hover {
        background-color: #3d3d5c;
      }
      .placeholder-msg {
        color: #888;
      }
      .entity-list-actions {
        border-bottom-color: #444;
        background-color: #2d2d44;
      }
      .entity-list-actions button {
        background-color: #333;
        color: #e0e0e0;
        border-color: #555;
      }
      .entity-list-actions button:hover {
        background-color: #444;
      }
      .sidebar-section {
        border-color: #444;
        background-color: #2d2d44;
      }
      .sidebar-section-actions button {
        background-color: #333;
        color: #e0e0e0;
        border-color: #555;
      }
      .sidebar-section-actions button:hover {
        background-color: #444;
      }
    }

    h1 {
      margin: 0.5rem 0;
      font-size: 1.4rem;
      font-weight: 600;
      text-align: center;
    }

    .tab-bar {
      display: flex;
      gap: 0.25rem;
      justify-content: center;
      margin: 0.5rem 0;
    }

    .tab-bar button {
      padding: 0.5rem 1.2rem;
      font-size: 0.9rem;
      border: 1px solid #ccc;
      border-bottom: 3px solid transparent;
      border-radius: 6px 6px 0 0;
      background-color: #ffffff;
      color: #212529;
      cursor: pointer;
      transition: background-color 0.15s, border-bottom-color 0.15s;
    }

    .tab-bar button:hover {
      background-color: #e9ecef;
    }

    .tab-bar button.active {
      background-color: #e9ecef;
      border-bottom-color: #4a9c6d;
      font-weight: 600;
    }

    .controls {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 0.5rem;
      flex-wrap: wrap;
      justify-content: center;
    }

    .controls button {
      padding: 0.4rem 0.9rem;
      font-size: 0.9rem;
      border: 1px solid #ccc;
      border-radius: 6px;
      background-color: #ffffff;
      color: #212529;
      cursor: pointer;
      transition: background-color 0.15s;
    }

    .controls button:hover {
      background-color: #e9ecef;
    }

    .tab-panel {
      display: none;
      flex: 1;
    }

    .tab-panel.active {
      display: flex;
      flex-direction: row;
      gap: 1rem;
    }

    /* Package Map tab has no sidebar */
    .tab-panel.active.full-width {
      flex-direction: column;
      align-items: center;
    }

    .entity-list {
      width: 260px;
      min-width: 260px;
      max-height: calc(100vh - 200px);
      align-self: flex-start;
      overflow-y: auto;
      border: 1px solid #ccc;
      border-radius: 6px;
      background-color: #fff;
      padding: 0.5rem;
    }

    .sidebar-col {
      width: 260px;
      min-width: 260px;
      max-height: calc(100vh - 200px);
      align-self: flex-start;
      display: flex;
      flex-direction: column;
      gap: 0.4rem;
    }

    .entity-list label,
    .sidebar-section-body label {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.3rem 0.4rem;
      border-radius: 4px;
      cursor: pointer;
      font-size: 0.85rem;
      line-height: 1.3;
    }

    .entity-list label:hover,
    .sidebar-section-body label:hover {
      background-color: #f0f0f0;
    }

    .entity-list input[type="checkbox"],
    .sidebar-section-body input[type="checkbox"] {
      flex-shrink: 0;
    }

    .entity-list-actions {
      display: flex;
      gap: 0.25rem;
      margin-bottom: 0.5rem;
      padding-bottom: 0.5rem;
      border-bottom: 1px solid #e0e0e0;
      position: sticky;
      top: 0;
      background-color: #fff;
      z-index: 1;
    }

    .entity-list-actions button {
      flex: 1;
      padding: 0.3rem 0.5rem;
      font-size: 0.75rem;
      border: 1px solid #ccc;
      border-radius: 4px;
      background-color: #f8f9fa;
      color: #212529;
      cursor: pointer;
      transition: background-color 0.15s;
    }

    .entity-list-actions button:hover {
      background-color: #e9ecef;
    }

    .entity-list .pkg-name,
    .sidebar-section-body .pkg-name {
      color: #888;
      font-size: 0.75rem;
    }

    .sidebar-section {
      border: 1px solid #ccc;
      border-radius: 6px;
      background-color: #fff;
      padding: 0.3rem 0.5rem;
    }
    .sidebar-section[open] {
      overflow-y: auto;
      flex: 1;
      min-height: 0;
    }
    .sidebar-section-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0.3rem 0.1rem;
      font-size: 0.85rem;
      font-weight: 600;
      cursor: pointer;
      user-select: none;
    }
    .sidebar-section-header::-webkit-details-marker {
      margin-right: 0.3rem;
    }
    .sidebar-section-actions {
      display: flex;
      gap: 0.25rem;
    }
    .sidebar-section-actions button {
      padding: 0.15rem 0.4rem;
      font-size: 0.7rem;
      border: 1px solid #ccc;
      border-radius: 4px;
      background-color: #f8f9fa;
      color: #212529;
      cursor: pointer;
      transition: background-color 0.15s;
    }
    .sidebar-section-actions button:hover {
      background-color: #e9ecef;
    }
    .sidebar-section-body {
      padding: 0 0 0.3rem 0;
    }

    .diagram-viewport {
      flex: 1;
      overflow: auto;
      display: flex;
      justify-content: center;
      align-items: flex-start;
      padding: 1rem;
    }

    .diagram-container {
      width: 100%;
      transform-origin: top center;
      transition: transform 0.2s ease;
    }

    .placeholder-msg {
      color: #666;
      font-size: 1rem;
      text-align: center;
      padding: 3rem;
    }

    /* Override Mermaid's small default font sizes in class diagrams */
    .mermaid svg { font-size: 18px !important; }
    .mermaid svg g.classGroup text { font-size: 18px !important; }
    .mermaid svg .classTitleText { font-size: 28px !important; }
    .mermaid svg .nodeLabel { font-size: 18px !important; }
    .mermaid svg .edgeLabel { font-size: 16px !important; }
    .mermaid svg .label text { font-size: 18px !important; }

    /* Left-align interface methods in class diagram nodes */
    .mermaid svg .methods-group foreignObject div {
      text-align: left !important;
    }

    /* Color coding: interface blocks (blue) */
    .mermaid svg g.node.interfaceStyle > g:first-child > path:first-child {
      fill: #2374ab !important;
    }
    .mermaid svg g.node.interfaceStyle > g:first-child > path:nth-child(2) {
      stroke: #1a5a8a !important;
      stroke-width: 2px !important;
    }
    .mermaid svg g.node.interfaceStyle .nodeLabel {
      color: #fff !important;
    }

    /* Color coding: implementation blocks (green) */
    .mermaid svg g.node.implStyle > g:first-child > path:first-child {
      fill: #4a9c6d !important;
    }
    .mermaid svg g.node.implStyle > g:first-child > path:nth-child(2) {
      stroke: #357a50 !important;
      stroke-width: 2px !important;
    }
    .mermaid svg g.node.implStyle .nodeLabel {
      color: #fff !important;
    }

    /* Treemap styles */
    .treemap-viewport {
      flex: 1;
      overflow: hidden;
      padding: 0.5rem;
      position: relative;
      width: 100%;
      height: calc(100vh - 200px);
    }

    .treemap-container {
      width: 100%;
      height: 100%;
      position: relative;
    }

    .treemap-node {
      position: absolute;
      overflow: hidden;
      border: 1px solid rgba(0,0,0,0.15);
      border-radius: 3px;
      display: flex;
      flex-direction: column;
      justify-content: center;
      align-items: center;
      text-align: center;
      cursor: default;
      transition: border-color 0.15s, border-width 0.15s, box-shadow 0.15s;
      min-height: 36px;
      min-width: 80px;
    }

    .treemap-node:hover {
      border-color: rgba(0,0,0,0.5);
      z-index: 10;
      overflow: visible;
    }

    .treemap-node .tm-name {
      font-weight: 600;
      font-size: 0.85rem;
      line-height: 1.2;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      max-width: 95%;
      flex-shrink: 0;
    }

    .treemap-node .tm-stats {
      font-size: 0.7rem;
      opacity: 0.7;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      max-width: 95%;
      flex-shrink: 0;
    }

    .treemap-group {
      position: absolute;
      overflow: visible;
      border: 2px solid rgba(0,0,0,0.2);
      border-radius: 4px;
      min-height: 24px;
      min-width: 50px;
    }

    .treemap-group-label {
      position: absolute;
      top: 0; left: 0; right: 0;
      padding: 2px 6px;
      font-size: 0.7rem;
      font-weight: 600;
      background: rgba(0,0,0,0.06);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      z-index: 1;
    }

    .treemap-tooltip {
      position: fixed;
      padding: 6px 10px;
      background: rgba(0,0,0,0.85);
      color: #fff;
      font-size: 0.8rem;
      border-radius: 4px;
      pointer-events: none;
      z-index: 100;
      white-space: nowrap;
      display: none;
    }

    .treemap-node[data-clickable] {
      cursor: pointer;
    }

    .treemap-node.tm-selected {
      border: 2px solid #1976d2;
      box-shadow: 0 0 0 2px rgba(25,118,210,0.3);
    }

    .treemap-node.tm-has-selection {
      border: 3px solid #1976d2;
      box-shadow: 0 0 0 2px rgba(25,118,210,0.25);
    }

    .treemap-node.tm-selected.tm-has-selection {
      border: 2px solid #1976d2;
      box-shadow: 0 0 0 2px rgba(25,118,210,0.3);
    }

    .treemap-overlay {
      position: absolute;
      background: #fff;
      border: 1px solid #ccc;
      border-radius: 6px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.15);
      max-height: 300px;
      overflow-y: auto;
      min-width: 200px;
      z-index: 50;
      padding: 8px 0;
    }

    .treemap-overlay-header {
      padding: 4px 12px 6px;
      font-weight: 600;
      font-size: 0.85rem;
      border-bottom: 1px solid #eee;
      margin-bottom: 4px;
    }

    .treemap-overlay-section {
      padding: 4px 12px 2px;
      font-size: 0.7rem;
      font-weight: 600;
      color: #888;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .treemap-overlay-item {
      padding: 3px 12px;
      font-size: 0.8rem;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 6px;
    }

    .treemap-overlay-item input[type="checkbox"] {
      margin: 0;
      flex-shrink: 0;
    }

    .treemap-overlay-item:hover {
      background-color: #f0f0f0;
    }

    @media (prefers-color-scheme: dark) {
      .treemap-node {
        border-color: rgba(255,255,255,0.15);
        color: #222 !important;
      }
      .treemap-node:hover {
        border-color: rgba(255,255,255,0.5);
      }
      .treemap-node.tm-selected {
        border-color: #7c8dff;
        box-shadow: 0 0 0 2px rgba(124,141,255,0.3);
      }
      .treemap-node.tm-has-selection {
        border-width: 3px;
        border-color: #7c8dff;
        box-shadow: 0 0 0 2px rgba(124,141,255,0.25);
      }
      .treemap-node.tm-selected.tm-has-selection {
        border-width: 2px;
        border-color: #7c8dff;
        box-shadow: 0 0 0 2px rgba(124,141,255,0.3);
      }
      .treemap-group {
        border-color: rgba(255,255,255,0.2);
      }
      .treemap-group-label {
        background: rgba(255,255,255,0.08);
        color: #e0e0e0;
      }
      .treemap-overlay {
        background: #2d2d44;
        border-color: #444;
        box-shadow: 0 4px 12px rgba(0,0,0,0.4);
      }
      .treemap-overlay-header {
        color: #e0e0e0;
        border-bottom-color: #444;
      }
      .treemap-overlay-section {
        color: #999;
      }
      .treemap-overlay-item {
        color: #e0e0e0;
      }
      .treemap-overlay-item:hover {
        background-color: #3d3d5c;
      }
    }
  </style>
</head>
<body>
  <h1>goifaces — {{.RepoAddress}}</h1>

  <div class="tab-bar">
    <button class="tab-btn active" data-tab="pkgmap-html">Package Map</button>
    <button class="tab-btn" data-tab="structures">Structures</button>
  </div>

  <div class="controls">
    <button id="zoom-in" title="Zoom In">+ Zoom In</button>
    <button id="zoom-out" title="Zoom Out">- Zoom Out</button>
    <button id="zoom-reset" title="Reset Zoom">Reset</button>
    <button id="copy-src" title="Copy Source">Copy Source</button>
  </div>

  <!-- Package Map tab -->
  <div class="tab-panel active full-width" id="panel-pkgmap-html">
    <div class="treemap-viewport" id="pkgmap-html-viewport">
      <div class="treemap-container" id="pkgmap-html-container"></div>
    </div>
  </div>

  <div class="treemap-tooltip" id="treemap-tooltip"></div>

  <!-- Structures tab -->
  <div class="tab-panel" id="panel-structures">
    <div class="sidebar-col" id="structures-list">
      <details class="sidebar-section" open style="order:1">
        <summary class="sidebar-section-header">
          Implementations
          <span class="sidebar-section-actions">
            <button id="impls-all" title="Select all implementations">All</button>
            <button id="impls-clear" title="Deselect all implementations">Clear</button>
          </span>
        </summary>
        <div class="sidebar-section-body" id="impls-list"></div>
      </details>
      <details class="sidebar-section" style="order:0">
        <summary class="sidebar-section-header">
          Interfaces
          <span class="sidebar-section-actions">
            <button id="ifaces-all" title="Select all interfaces">All</button>
            <button id="ifaces-clear" title="Deselect all interfaces">Clear</button>
          </span>
        </summary>
        <div class="sidebar-section-body" id="ifaces-list"></div>
      </details>
    </div>
    <div class="diagram-viewport">
      <div class="diagram-container" id="structures-diagram-container">
        <div class="placeholder-msg" id="structures-placeholder">Select items from the list to view their relationships</div>
        <pre class="mermaid" id="structures-mermaid" style="display:none;"></pre>
      </div>
    </div>
  </div>

  <script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
  <script>
    mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      themeVariables: {
        primaryColor: '#ffffff',
        primaryBorderColor: '#cccccc',
        primaryTextColor: '#000000',
        lineColor: '#555555',
        fontSize: '16px'
      }
    });

    (function() {
      var data = {{.DataJSON}};
      var pkgMapData = {{.PackageMapJSON}};
      var currentTab = 'pkgmap-html';
      var currentMermaidSource = '';
      var pkgMapHtmlRendered = false;

      // Shared selection state (module-level, drives both overlay and sidebar)
      var selectedTypeIDs = {};   // { [id]: true }
      var selectedIfaceIDs = {};  // { [id]: true }
      var updatingUI = false;     // re-entrancy guard for updateSelectionUI

      // Pastel palette matching Go-side colors
      var treemapPalette = [
        {fill: '#e8f4fd', stroke: '#b8d4e8', text: '#333333'},
        {fill: '#e8f5e9', stroke: '#b8d8ba', text: '#333333'},
        {fill: '#fff3e0', stroke: '#e8c9a0', text: '#333333'},
        {fill: '#f3e5f5', stroke: '#d1b3d8', text: '#333333'},
        {fill: '#fce4ec', stroke: '#e8b0bf', text: '#333333'},
        {fill: '#e0f2f1', stroke: '#b0d4d1', text: '#333333'},
        {fill: '#fff9c4', stroke: '#e8dea0', text: '#333333'},
        {fill: '#e8eaf6', stroke: '#b8bce8', text: '#333333'},
        {fill: '#efebe9', stroke: '#c8b8ad', text: '#333333'},
        {fill: '#f1f8e9', stroke: '#c4dba0', text: '#333333'}
      ];

      // Squarified treemap algorithm
      function squarify(nodes, rect) {
        if (!nodes || nodes.length === 0) return [];
        var total = 0;
        for (var i = 0; i < nodes.length; i++) total += nodes[i].value;
        if (total <= 0) return [];

        var results = [];
        var remaining = nodes.slice().sort(function(a, b) { return b.value - a.value; });
        var r = {x: rect.x, y: rect.y, w: rect.w, h: rect.h};
        var remainingTotal = total;

        while (remaining.length > 0) {
          var short = Math.min(r.w, r.h);
          if (short <= 0) break;
          var row = [remaining[0]];
          remaining.splice(0, 1);
          var rowSum = row[0].value;

          var worst = worstRatio(row, rowSum, short, remainingTotal, r);

          while (remaining.length > 0) {
            var candidate = remaining[0];
            var newRow = row.concat([candidate]);
            var newSum = rowSum + candidate.value;
            var newWorst = worstRatio(newRow, newSum, short, remainingTotal, r);
            if (newWorst <= worst) {
              row.push(candidate);
              remaining.splice(0, 1);
              rowSum = newSum;
              worst = newWorst;
            } else {
              break;
            }
          }

          // Layout this row
          var rowArea = (rowSum / remainingTotal) * r.w * r.h;
          var horizontal = r.w >= r.h;
          var rowLen = horizontal ? rowArea / r.h : rowArea / r.w;
          if (!isFinite(rowLen) || rowLen <= 0) rowLen = 0;

          var offset = 0;
          for (var j = 0; j < row.length; j++) {
            var fraction = rowSum > 0 ? row[j].value / rowSum : 1 / row.length;
            var span = horizontal ? r.h * fraction : r.w * fraction;
            var item = {
              data: row[j],
              x: horizontal ? r.x : r.x + offset,
              y: horizontal ? r.y + offset : r.y,
              w: horizontal ? rowLen : span,
              h: horizontal ? span : rowLen
            };
            results.push(item);
            offset += span;
          }

          remainingTotal -= rowSum;
          if (horizontal) {
            r = {x: r.x + rowLen, y: r.y, w: r.w - rowLen, h: r.h};
          } else {
            r = {x: r.x, y: r.y + rowLen, w: r.w, h: r.h - rowLen};
          }
        }
        return results;
      }

      function worstRatio(row, rowSum, short, total, rect) {
        var area = (rowSum / total) * rect.w * rect.h;
        var rowLen = short > 0 ? area / short : 0;
        var worst = 0;
        for (var i = 0; i < row.length; i++) {
          var fraction = rowSum > 0 ? row[i].value / rowSum : 1 / row.length;
          var span = short * fraction;
          var ratio = rowLen > span ? rowLen / (span > 0 ? span : 1) : span / (rowLen > 0 ? rowLen : 1);
          if (ratio > worst) worst = ratio;
        }
        return worst;
      }

      function verticalStack(nodes, rect) {
        var total = 0;
        for (var i = 0; i < nodes.length; i++) total += nodes[i].value;
        if (total <= 0) return [];

        var minChildH = 2 * TREEMAP_GAP + 36;
        var heights = [];
        var totalH = 0;
        for (var i = 0; i < nodes.length; i++) {
          var h = Math.max(minChildH, rect.h * (nodes[i].value / total));
          heights.push(h);
          totalH += h;
        }

        // Scale down proportionally if total exceeds available height
        var scale = totalH > rect.h && totalH > 0 ? rect.h / totalH : 1;

        var results = [];
        var y = rect.y;
        for (var i = 0; i < nodes.length; i++) {
          var h = heights[i] * scale;
          results.push({
            data: nodes[i],
            x: rect.x,
            y: y,
            w: rect.w,
            h: h
          });
          y += h;
        }
        return results;
      }

      // Flatten deep nesting: cap at maxDepth levels.
      // Applies sqrt scaling to compress the value range so large packages
      // don't dominate the layout and small packages remain readable.
      function flattenTree(nodes, maxDepth) {
        if (!nodes) return [];
        return nodes.map(function(n) {
          var clone = {name: n.name, relPath: n.relPath, pkgPath: n.pkgPath, interfaces: n.interfaces, types: n.types, value: n.value};
          if (n.children && n.children.length > 0) {
            if (maxDepth <= 1) {
              clone.children = null;
              clone.value = Math.max(1, Math.ceil(Math.sqrt(n.value)));
            } else {
              clone.children = flattenTree(n.children, maxDepth - 1);
              var sum = 0;
              for (var i = 0; i < clone.children.length; i++) sum += clone.children[i].value;
              var own = (n.interfaces || 0) + (n.types || 0);
              if (own > 0) sum += Math.max(1, Math.ceil(Math.sqrt(own)));
              clone.value = sum;
            }
          } else {
            clone.value = Math.max(1, Math.ceil(Math.sqrt(n.value)));
          }
          return clone;
        });
      }

      // Build package→interfaces/types lookup maps for overlay
      var pkgInterfaces = {};
      var pkgTypes = {};
      data.interfaces.forEach(function(iface) {
        if (!iface.pkgPath) return;
        if (!pkgInterfaces[iface.pkgPath]) pkgInterfaces[iface.pkgPath] = [];
        pkgInterfaces[iface.pkgPath].push(iface);
      });
      data.types.forEach(function(t) {
        if (!t.pkgPath) return;
        if (!pkgTypes[t.pkgPath]) pkgTypes[t.pkgPath] = [];
        pkgTypes[t.pkgPath].push(t);
      });

      // Overlay state
      var activeOverlay = null;
      var selectedNode = null;

      function showPackageOverlay(nodeEl, d) {
        dismissOverlay();
        var ifaces = pkgInterfaces[d.pkgPath] || [];
        var types = pkgTypes[d.pkgPath] || [];
        if (ifaces.length === 0 && types.length === 0) return;

        var overlay = document.createElement('div');
        overlay.className = 'treemap-overlay';

        var header = document.createElement('div');
        header.className = 'treemap-overlay-header';
        header.textContent = d.relPath ? d.relPath : d.name;
        overlay.appendChild(header);

        if (ifaces.length > 0) {
          var sec = document.createElement('div');
          sec.className = 'treemap-overlay-section';
          sec.textContent = 'Interfaces';
          overlay.appendChild(sec);
          ifaces.forEach(function(iface) {
            var itemLabel = document.createElement('label');
            itemLabel.className = 'treemap-overlay-item';
            var cb = document.createElement('input');
            cb.type = 'checkbox';
            cb.checked = !!selectedIfaceIDs[iface.id];
            cb.setAttribute('data-id', iface.id);
            cb.setAttribute('data-kind', 'iface');
            cb.addEventListener('change', function() {
              if (cb.checked) {
                selectedIfaceIDs[iface.id] = true;
              } else {
                delete selectedIfaceIDs[iface.id];
              }
              updateSelectionUI();
            });
            var nameSpan = document.createElement('span');
            nameSpan.textContent = iface.name.indexOf('.') >= 0 ? iface.name.split('.').pop() : iface.name;
            itemLabel.appendChild(cb);
            itemLabel.appendChild(nameSpan);
            overlay.appendChild(itemLabel);
          });
        }

        if (types.length > 0) {
          var sec2 = document.createElement('div');
          sec2.className = 'treemap-overlay-section';
          sec2.textContent = 'Types';
          overlay.appendChild(sec2);
          types.forEach(function(t) {
            var itemLabel = document.createElement('label');
            itemLabel.className = 'treemap-overlay-item';
            var cb = document.createElement('input');
            cb.type = 'checkbox';
            cb.checked = !!selectedTypeIDs[t.id];
            cb.setAttribute('data-id', t.id);
            cb.setAttribute('data-kind', 'type');
            cb.addEventListener('change', function() {
              if (cb.checked) {
                selectedTypeIDs[t.id] = true;
              } else {
                delete selectedTypeIDs[t.id];
              }
              updateSelectionUI();
            });
            var nameSpan = document.createElement('span');
            nameSpan.textContent = t.name.indexOf('.') >= 0 ? t.name.split('.').pop() : t.name;
            itemLabel.appendChild(cb);
            itemLabel.appendChild(nameSpan);
            overlay.appendChild(itemLabel);
          });
        }

        // Position overlay near the clicked block
        var viewport = document.getElementById('pkgmap-html-viewport');
        var vpRect = viewport.getBoundingClientRect();
        var nodeRect = nodeEl.getBoundingClientRect();

        // Position overlay below the clicked block, left-aligned
        var left = nodeRect.left - vpRect.left + viewport.scrollLeft;
        var top = nodeRect.bottom - vpRect.top + viewport.scrollTop + 4;

        // Set width to match the clicked box (min 200px)
        overlay.style.width = Math.max(200, nodeRect.width) + 'px';

        // Clamp max-height so overlay doesn't overflow viewport bottom
        var spaceBelow = vpRect.height - (nodeRect.bottom - vpRect.top) - 8;
        if (spaceBelow <= 0) {
          // No room below — position above the node
          top = nodeRect.top - vpRect.top + viewport.scrollTop - 4;
          viewport.appendChild(overlay);
          var oh = overlay.offsetHeight;
          top = top - oh;
          if (top < 0) {
            overlay.style.maxHeight = (oh + top) + 'px';
            top = 0;
          }
        } else {
          if (spaceBelow < 300) {
            overlay.style.maxHeight = Math.max(80, spaceBelow) + 'px';
          }
          viewport.appendChild(overlay);
        }

        overlay.style.left = left + 'px';
        overlay.style.top = top + 'px';
        nodeEl.classList.add('tm-selected');
        activeOverlay = overlay;
        selectedNode = nodeEl;
      }

      function dismissOverlay() {
        if (activeOverlay) {
          activeOverlay.remove();
          activeOverlay = null;
        }
        if (selectedNode) {
          selectedNode.classList.remove('tm-selected');
          selectedNode = null;
        }
      }

      // Click outside overlay to dismiss
      document.getElementById('pkgmap-html-viewport').addEventListener('click', function(e) {
        if (activeOverlay && !activeOverlay.contains(e.target) && (!e.target.hasAttribute || !e.target.hasAttribute('data-clickable'))) {
          dismissOverlay();
        }
      });

      var tooltip = document.getElementById('treemap-tooltip');
      var TREEMAP_GAP = 12;
      var MAX_BLOCK_HEIGHT = 120;
      var MIN_NODE_WIDTH = 80;

      function renderTreemap(container, nodes, rect, depth, colorIdx) {
        if (!nodes || nodes.length === 0) {
          if (depth === 0) {
            container.innerHTML = '<div class="placeholder-msg">No packages found</div>';
          }
          return colorIdx;
        }

        var positioned = squarify(nodes, rect);

        // If any child would be narrower than MIN_NODE_WIDTH, fall back to vertical stacking
        var needsVerticalStack = false;
        for (var i = 0; i < positioned.length; i++) {
          if (positioned[i].w - 2 * TREEMAP_GAP < MIN_NODE_WIDTH) {
            needsVerticalStack = true;
            break;
          }
        }
        if (needsVerticalStack) {
          positioned = verticalStack(nodes, rect);
        }

        for (var i = 0; i < positioned.length; i++) {
          var p = positioned[i];
          var d = p.data;
          var ci = (colorIdx + i) % treemapPalette.length;
          var color = treemapPalette[ci];

          if (d.children && d.children.length > 0) {
            // Group node
            var group = document.createElement('div');
            group.className = 'treemap-group';
            group.style.left = (p.x + TREEMAP_GAP) + 'px';
            group.style.top = (p.y + TREEMAP_GAP) + 'px';
            group.style.width = Math.max(0, p.w - 2 * TREEMAP_GAP) + 'px';
            group.style.height = Math.max(0, p.h - 2 * TREEMAP_GAP) + 'px';
            group.style.background = color.fill;

            var label = document.createElement('div');
            label.className = 'treemap-group-label';
            label.textContent = d.name;
            label.style.color = color.text;
            group.appendChild(label);

            var headerH = 20;
            var gw = Math.max(0, p.w - 2 * TREEMAP_GAP);
            var gh = Math.max(0, p.h - 2 * TREEMAP_GAP);
            var innerPad = 9;
            var innerRect = {x: innerPad, y: headerH, w: Math.max(0, gw - 2 * innerPad), h: Math.max(0, gh - headerH - innerPad)};

            // If this node is also a package, add a self node
            if (d.interfaces > 0 || d.types > 0) {
              var selfValue = d.interfaces + d.types;
              if (selfValue < 1) selfValue = 1;
              var childrenValue = 0;
              for (var k = 0; k < d.children.length; k++) childrenValue += d.children[k].value;
              var selfFraction = selfValue / (selfValue + childrenValue);
              var selfH = innerRect.h * selfFraction;

              var renderedSelfH = Math.min(MAX_BLOCK_HEIGHT, Math.max(0, selfH));
              var selfNode = document.createElement('div');
              selfNode.className = 'treemap-node';
              if (d.pkgPath) selfNode.setAttribute('data-pkgpath', d.pkgPath);
              selfNode.style.left = innerRect.x + 'px';
              selfNode.style.top = innerRect.y + 'px';
              selfNode.style.width = innerRect.w + 'px';
              selfNode.style.height = renderedSelfH + 'px';
              selfNode.style.background = treemapPalette[(ci + 1) % treemapPalette.length].fill;
              selfNode.style.color = color.text;

              var sn = document.createElement('div');
              sn.className = 'tm-name';
              sn.textContent = depth > 0 ? d.name : (d.relPath || d.name);
              selfNode.appendChild(sn);
              var ss = document.createElement('div');
              ss.className = 'tm-stats';
              ss.textContent = statsText(d);
              selfNode.appendChild(ss);
              attachTooltip(selfNode, d);
              attachClickHandler(selfNode, d);
              group.appendChild(selfNode);

              innerRect = {x: innerRect.x, y: innerRect.y + renderedSelfH, w: innerRect.w, h: Math.max(0, innerRect.h - renderedSelfH)};
            }

            colorIdx = renderTreemap(group, d.children, innerRect, depth + 1, ci + 1);
            container.appendChild(group);
          } else {
            // Leaf node
            var node = document.createElement('div');
            node.className = 'treemap-node';
            if (d.pkgPath) node.setAttribute('data-pkgpath', d.pkgPath);
            node.style.left = (p.x + TREEMAP_GAP) + 'px';
            node.style.top = (p.y + TREEMAP_GAP) + 'px';
            node.style.width = Math.max(0, p.w - 2 * TREEMAP_GAP) + 'px';
            node.style.height = Math.min(MAX_BLOCK_HEIGHT, Math.max(0, p.h - 2 * TREEMAP_GAP)) + 'px';
            node.style.background = color.fill;
            node.style.color = color.text;

            var nameEl = document.createElement('div');
            nameEl.className = 'tm-name';
            nameEl.textContent = depth > 0 ? d.name : (d.relPath || d.name);
            node.appendChild(nameEl);
            var statsEl = document.createElement('div');
            statsEl.className = 'tm-stats';
            statsEl.textContent = statsText(d);
            node.appendChild(statsEl);
            attachTooltip(node, d);
            attachClickHandler(node, d);
            container.appendChild(node);
          }
        }
        return colorIdx + positioned.length;
      }

      function statsText(d) {
        var parts = [];
        if (d.interfaces > 0) parts.push(d.interfaces + ' iface' + (d.interfaces > 1 ? 's' : ''));
        if (d.types > 0) parts.push(d.types + ' type' + (d.types > 1 ? 's' : ''));
        return parts.join(', ') || '(empty)';
      }

      function attachTooltip(el, d) {
        el.addEventListener('mouseenter', function(e) {
          var text = (d.relPath || d.name) + ': ' + statsText(d);
          if (d.pkgPath) text = d.pkgPath + '\n' + statsText(d);
          tooltip.textContent = text;
          tooltip.style.whiteSpace = d.pkgPath ? 'pre' : 'nowrap';
          tooltip.style.display = 'block';
          positionTooltip(e);
        });
        el.addEventListener('mousemove', positionTooltip);
        el.addEventListener('mouseleave', function() {
          tooltip.style.display = 'none';
        });
      }

      function attachClickHandler(el, d) {
        if (!d.pkgPath) return;
        var ifaces = pkgInterfaces[d.pkgPath] || [];
        var types = pkgTypes[d.pkgPath] || [];
        if (ifaces.length === 0 && types.length === 0) return;
        el.setAttribute('data-clickable', 'true');
        el.addEventListener('click', function(e) {
          e.stopPropagation();
          if (selectedNode === el) {
            dismissOverlay();
          } else {
            showPackageOverlay(el, d);
          }
        });
      }

      function positionTooltip(e) {
        tooltip.style.left = (e.clientX + 12) + 'px';
        tooltip.style.top = (e.clientY + 12) + 'px';
      }

      var resizeTimer = null;
      function layoutTreemap() {
        dismissOverlay();
        var viewport = document.getElementById('pkgmap-html-viewport');
        var container = document.getElementById('pkgmap-html-container');
        container.innerHTML = '';
        var w = viewport.clientWidth - 16;
        var h = viewport.clientHeight - 16;
        if (w <= 0 || h <= 0) return;
        var nodes = flattenTree(pkgMapData, 3);
        renderTreemap(container, nodes, {x: 0, y: 0, w: w, h: h}, 0, 0);
        updatePackageMapHighlights();
      }

      // Build checkbox lists (deferred to avoid blocking initial paint)
      var implsList = document.getElementById('impls-list');
      var ifacesList = document.getElementById('ifaces-list');

      setTimeout(function() {
        var implsFrag = document.createDocumentFragment();
        data.types.forEach(function(t) {
          var label = document.createElement('label');
          var cb = document.createElement('input');
          cb.type = 'checkbox';
          cb.value = t.id;
          cb.className = 'impl-cb';
          cb.addEventListener('change', onSelectionChange);
          var span = document.createElement('span');
          span.appendChild(document.createTextNode(t.name + ' '));
          var pkg = document.createElement('span');
          pkg.className = 'pkg-name';
          pkg.textContent = t.pkgName;
          span.appendChild(pkg);
          label.appendChild(cb);
          label.appendChild(span);
          implsFrag.appendChild(label);
        });
        implsList.appendChild(implsFrag);

        var ifacesFrag = document.createDocumentFragment();
        data.interfaces.forEach(function(iface) {
          var label = document.createElement('label');
          var cb = document.createElement('input');
          cb.type = 'checkbox';
          cb.value = iface.id;
          cb.className = 'iface-cb';
          cb.addEventListener('change', onSelectionChange);
          var span = document.createElement('span');
          span.appendChild(document.createTextNode(iface.name + ' '));
          var pkg = document.createElement('span');
          pkg.className = 'pkg-name';
          pkg.textContent = iface.pkgName;
          span.appendChild(pkg);
          label.appendChild(cb);
          label.appendChild(span);
          ifacesFrag.appendChild(label);
        });
        ifacesList.appendChild(ifacesFrag);
      }, 0);

      // Bulk selection: Implementations
      document.getElementById('impls-all').addEventListener('click', function() {
        document.querySelectorAll('.impl-cb').forEach(function(cb) { cb.checked = true; });
        onSelectionChange();
      });
      document.getElementById('impls-clear').addEventListener('click', function() {
        document.querySelectorAll('.impl-cb').forEach(function(cb) { cb.checked = false; });
        onSelectionChange();
      });

      // Bulk selection: Interfaces
      document.getElementById('ifaces-all').addEventListener('click', function() {
        document.querySelectorAll('.iface-cb').forEach(function(cb) { cb.checked = true; });
        onSelectionChange();
      });
      document.getElementById('ifaces-clear').addEventListener('click', function() {
        document.querySelectorAll('.iface-cb').forEach(function(cb) { cb.checked = false; });
        onSelectionChange();
      });

      // Accordion: only one sidebar section open at a time, collapsed on top
      document.querySelectorAll('.sidebar-section').forEach(function(details) {
        details.addEventListener('toggle', function() {
          if (this.open) {
            this.style.order = '1';
            document.querySelectorAll('.sidebar-section').forEach(function(other) {
              if (other !== details) {
                other.removeAttribute('open');
                other.style.order = '0';
              }
            });
          }
        });
      });

      // Tab switching
      document.querySelectorAll('.tab-btn').forEach(function(btn) {
        btn.addEventListener('click', function() {
          var tab = this.getAttribute('data-tab');
          switchTab(tab);
        });
      });

      function switchTab(tab) {
        currentTab = tab;
        document.querySelectorAll('.tab-btn').forEach(function(b) { b.classList.remove('active'); });
        document.querySelector('[data-tab="' + tab + '"]').classList.add('active');
        document.querySelectorAll('.tab-panel').forEach(function(p) { p.classList.remove('active'); });
        document.getElementById('panel-' + tab).classList.add('active');

        if (tab === 'pkgmap-html' && !pkgMapHtmlRendered) {
          requestAnimationFrame(function() {
            layoutTreemap();
            pkgMapHtmlRendered = true;
          });
        }
      }

      // Initial render of treemap on page load
      requestAnimationFrame(function() {
        layoutTreemap();
        pkgMapHtmlRendered = true;
      });

      function fixSvgWidth(pre) {
        var svg = pre.querySelector('svg');
        if (!svg) return;
        var vb = svg.getAttribute('viewBox');
        if (!vb) return;
        var w = parseFloat(vb.split(/\s+/)[2]);
        if (w <= 0) return;
        var viewport = svg.closest('.diagram-viewport');
        if (!viewport) return;
        var available = viewport.clientWidth - 32;
        if (available <= 0) return;
        if (w > available) {
          svg.style.width = '100%';
          svg.style.maxWidth = '100%';
        } else {
          svg.style.width = w + 'px';
          svg.style.maxWidth = 'none';
        }
      }

      function triggerDiagramUpdate() {
        var typeIDs = Object.keys(selectedTypeIDs);
        var ifaceIDs = Object.keys(selectedIfaceIDs);

        if (typeIDs.length === 0 && ifaceIDs.length === 0) {
          showPlaceholder();
          currentMermaidSource = '';
          return;
        }

        var mermaidSrc = buildMermaid(typeIDs, ifaceIDs);
        currentMermaidSource = mermaidSrc;
        renderSelectionDiagram(mermaidSrc);
      }

      function updatePackageMapHighlights() {
        // Build set of pkgPaths that contain at least one selected item
        var activePkgs = {};
        for (var pkg in pkgInterfaces) {
          for (var i = 0; i < pkgInterfaces[pkg].length; i++) {
            if (selectedIfaceIDs[pkgInterfaces[pkg][i].id]) {
              activePkgs[pkg] = true;
              break;
            }
          }
        }
        for (var pkg in pkgTypes) {
          if (activePkgs[pkg]) continue;
          for (var i = 0; i < pkgTypes[pkg].length; i++) {
            if (selectedTypeIDs[pkgTypes[pkg][i].id]) {
              activePkgs[pkg] = true;
              break;
            }
          }
        }
        // Toggle class on all treemap nodes
        document.querySelectorAll('.treemap-node[data-pkgpath]').forEach(function(el) {
          var pkg = el.getAttribute('data-pkgpath');
          if (activePkgs[pkg]) {
            el.classList.add('tm-has-selection');
          } else {
            el.classList.remove('tm-has-selection');
          }
        });
      }

      function updateSelectionUI() {
        updatingUI = true;

        // Sync Structures sidebar checkboxes
        document.querySelectorAll('.impl-cb').forEach(function(cb) {
          cb.checked = !!selectedTypeIDs[cb.value];
        });
        document.querySelectorAll('.iface-cb').forEach(function(cb) {
          cb.checked = !!selectedIfaceIDs[cb.value];
        });

        // Sync overlay checkboxes (if overlay is open)
        if (activeOverlay) {
          activeOverlay.querySelectorAll('input[type="checkbox"]').forEach(function(cb) {
            var id = cb.getAttribute('data-id');
            var kind = cb.getAttribute('data-kind');
            if (kind === 'iface') {
              cb.checked = !!selectedIfaceIDs[id];
            } else {
              cb.checked = !!selectedTypeIDs[id];
            }
          });
        }

        updatePackageMapHighlights();

        updatingUI = false;
        triggerDiagramUpdate();
      }

      function onSelectionChange() {
        if (updatingUI) return;
        // Rebuild shared state from sidebar checkboxes
        selectedTypeIDs = {};
        document.querySelectorAll('.impl-cb:checked').forEach(function(cb) {
          selectedTypeIDs[cb.value] = true;
        });
        selectedIfaceIDs = {};
        document.querySelectorAll('.iface-cb:checked').forEach(function(cb) {
          selectedIfaceIDs[cb.value] = true;
        });
        updateSelectionUI();
      }

      function showPlaceholder() {
        document.getElementById('structures-placeholder').style.display = 'block';
        document.getElementById('structures-mermaid').style.display = 'none';
      }

      function renderSelectionDiagram(src) {
        var placeholder = document.getElementById('structures-placeholder');
        var pre = document.getElementById('structures-mermaid');
        placeholder.style.display = 'none';
        pre.removeAttribute('data-processed');
        pre.innerHTML = '';
        pre.textContent = src;
        pre.style.display = 'block';

        try {
          mermaid.run({ nodes: [pre] }).then(function() {
            fixSvgWidth(pre);
          }).catch(function(err) {
            pre.textContent = src;
            pre.style.whiteSpace = 'pre-wrap';
          });
        } catch(err) {
          pre.textContent = src;
          pre.style.whiteSpace = 'pre-wrap';
        }
      }

      function buildMermaid(typeIDList, ifaceIDList) {
        var typeSet = {};
        typeIDList.forEach(function(id) { typeSet[id] = true; });
        var ifaceSet = {};
        ifaceIDList.forEach(function(id) { ifaceSet[id] = true; });

        // Find matching relations
        var relatedTypeIDs = {};
        var relatedIfaceIDs = {};
        var filteredRels = [];

        data.relations.forEach(function(rel) {
          if (typeSet[rel.typeId] || ifaceSet[rel.interfaceId]) {
            filteredRels.push(rel);
            relatedTypeIDs[rel.typeId] = true;
            relatedIfaceIDs[rel.interfaceId] = true;
          }
        });

        // Build lookup maps
        var ifaceMap = {};
        data.interfaces.forEach(function(iface) { ifaceMap[iface.id] = iface; });
        var typeMap = {};
        data.types.forEach(function(t) { typeMap[t.id] = t; });

        // Collect included items
        var includedIfaces = [];
        var includedTypes = [];

        data.interfaces.forEach(function(iface) {
          if (ifaceSet[iface.id] || relatedIfaceIDs[iface.id]) {
            includedIfaces.push(iface);
          }
        });

        data.types.forEach(function(t) {
          if (typeSet[t.id] || relatedTypeIDs[t.id]) {
            includedTypes.push(t);
          }
        });

        // Build Mermaid classDiagram
        var lines = ['classDiagram'];
        if (includedIfaces.length > 0 || includedTypes.length > 0) {
          lines.push('    direction LR');
          lines.push('    classDef interfaceStyle fill:#2374ab,stroke:#1a5a8a,color:#fff,stroke-width:2px,font-weight:bold');
          lines.push('    classDef implStyle fill:#4a9c6d,stroke:#357a50,color:#fff,stroke-width:2px');
        }

        // Interface blocks
        includedIfaces.forEach(function(iface) {
          lines.push('');
          lines.push('    class ' + iface.id + ' {');
          lines.push('        <<interface>>');
          if (iface.sourceFile) {
            lines.push('        %% file: ' + iface.sourceFile);
          }
          if (iface.methods) {
            iface.methods.forEach(function(m) {
              lines.push('        +' + m);
            });
          }
          lines.push('    }');
        });

        // Type blocks
        if (includedIfaces.length > 0 && includedTypes.length > 0) {
          lines.push('');
        }
        includedTypes.forEach(function(t) {
          lines.push('');
          lines.push('    class ' + t.id + ' {');
          if (t.sourceFile) {
            lines.push('        %% file: ' + t.sourceFile);
          }
          lines.push('    }');
        });

        // Relations
        if ((includedIfaces.length > 0 || includedTypes.length > 0) && filteredRels.length > 0) {
          lines.push('');
        }
        filteredRels.forEach(function(rel) {
          lines.push('');
          lines.push('    ' + rel.typeId + ' --|> ' + rel.interfaceId);
        });

        // CSS class assignments
        if (includedIfaces.length > 0 || includedTypes.length > 0) {
          lines.push('');
          includedIfaces.forEach(function(iface) {
            lines.push('    cssClass "' + iface.id + '" interfaceStyle');
          });
          includedTypes.forEach(function(t) {
            lines.push('    cssClass "' + t.id + '" implStyle');
          });
        }

        return lines.join('\n');
      }

      // Zoom
      var scale = 1;
      var step = 0.15;
      var minScale = 0.1;
      var maxScale = 10;

      function getActiveContainer() {
        if (currentTab === 'pkgmap-html') return document.getElementById('pkgmap-html-container');
        return document.getElementById('structures-diagram-container');
      }

      function applyZoom() {
        getActiveContainer().style.transform = 'scale(' + scale + ')';
      }

      document.getElementById('zoom-in').addEventListener('click', function() {
        scale = Math.min(maxScale, scale + step);
        applyZoom();
      });
      document.getElementById('zoom-out').addEventListener('click', function() {
        scale = Math.max(minScale, scale - step);
        applyZoom();
      });
      document.getElementById('zoom-reset').addEventListener('click', function() {
        scale = 1;
        applyZoom();
      });

      document.getElementById('copy-src').addEventListener('click', function() {
        var src = '';
        if (currentTab === 'pkgmap-html') {
          src = buildTreemapText(pkgMapData, '');
        } else {
          src = currentMermaidSource;
        }
        if (!src) return;
        navigator.clipboard.writeText(src).then(function() {
          var btn = document.getElementById('copy-src');
          var orig = btn.textContent;
          btn.textContent = 'Copied!';
          setTimeout(function() { btn.textContent = orig; }, 1500);
        });
      });
      function buildTreemapText(nodes, indent) {
        if (!nodes) return '';
        var lines = [];
        for (var i = 0; i < nodes.length; i++) {
          var n = nodes[i];
          lines.push(indent + (n.relPath || n.name) + ': ' + statsText(n));
          if (n.children) {
            lines.push(buildTreemapText(n.children, indent + '  '));
          }
        }
        return lines.join('\n');
      }

      // ResizeObserver for treemap recalculation
      var resizeObs = new ResizeObserver(function() {
        if (!pkgMapHtmlRendered) return;
        if (resizeTimer) clearTimeout(resizeTimer);
        resizeTimer = setTimeout(function() {
          layoutTreemap();
        }, 100);
      });
      var vp = document.getElementById('pkgmap-html-viewport');
      if (vp) resizeObs.observe(vp);
    })();
  </script>
</body>
</html>
`

// interactiveData holds all data passed to the interactive HTML template.
type interactiveData struct {
	DataJSON       template.JS
	PackageMapJSON template.JS
	RepoAddress    string
}

// ServeInteractive starts the HTTP server with interactive tabbed UI.
// It blocks until the context is cancelled.
func ServeInteractive(ctx context.Context, data diagram.InteractiveData, port int, openBrowser bool, logger *slog.Logger) error {
	logger = logger.With("component", "server")
	tmpl, err := template.New("interactive").Parse(interactiveHTMLTemplate)
	if err != nil {
		return fmt.Errorf("parsing interactive HTML template: %w", err)
	}

	jsonBytes, err := json.Marshal(struct {
		Interfaces []diagram.InteractiveInterface `json:"interfaces"`
		Types      []diagram.InteractiveType      `json:"types"`
		Relations  []diagram.InteractiveRelation  `json:"relations"`
	}{
		Interfaces: data.Interfaces,
		Types:      data.Types,
		Relations:  data.Relations,
	})
	if err != nil {
		return fmt.Errorf("marshaling interactive data to JSON: %w", err)
	}

	pkgMapBytes, err := json.Marshal(data.PackageMapNodes)
	if err != nil {
		return fmt.Errorf("marshaling package map data to JSON: %w", err)
	}

	templateData := interactiveData{
		DataJSON:       template.JS(jsonBytes),   //nolint:gosec // JSON is generated from trusted internal data, not user input
		PackageMapJSON: template.JS(pkgMapBytes), //nolint:gosec // JSON is generated from trusted internal data, not user input
		RepoAddress:    data.RepoAddress,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, templateData); err != nil {
			logger.Error("failed to render interactive template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	logger.Info("starting HTTP server (interactive mode)", "addr", url)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
		close(errCh)
	}()

	if openBrowser {
		openInBrowser(url, logger)
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
		return nil
	}
}

// openInBrowser opens the given URL in the default system browser.
func openInBrowser(url string, logger *slog.Logger) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		logger.Warn("unsupported platform for opening browser", "os", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Warn("failed to open browser", "error", err)
	}
}
