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
      .entity-list label:hover {
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
      overflow-y: auto;
      border: 1px solid #ccc;
      border-radius: 6px;
      background-color: #fff;
      padding: 0.5rem;
    }

    .entity-list label {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.3rem 0.4rem;
      border-radius: 4px;
      cursor: pointer;
      font-size: 0.85rem;
      line-height: 1.3;
    }

    .entity-list label:hover {
      background-color: #f0f0f0;
    }

    .entity-list input[type="checkbox"] {
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

    .entity-list .pkg-name {
      color: #888;
      font-size: 0.75rem;
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
      transition: border-color 0.15s;
      min-height: 36px;
      min-width: 56px;
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

    @media (prefers-color-scheme: dark) {
      .treemap-node {
        border-color: rgba(255,255,255,0.15);
        color: #222 !important;
      }
      .treemap-node:hover {
        border-color: rgba(255,255,255,0.5);
      }
      .treemap-group {
        border-color: rgba(255,255,255,0.2);
      }
      .treemap-group-label {
        background: rgba(255,255,255,0.08);
        color: #e0e0e0;
      }
    }
  </style>
</head>
<body>
  <h1>goifaces — {{.RepoAddress}}</h1>

  <div class="tab-bar">
    <button class="tab-btn active" data-tab="pkgmap">Package Map</button>
    <button class="tab-btn" data-tab="pkgmap-html">Package Map (html)</button>
    <button class="tab-btn" data-tab="impls">Implementations</button>
    <button class="tab-btn" data-tab="ifaces">Interfaces</button>
  </div>

  <div class="controls">
    <button id="zoom-in" title="Zoom In">+ Zoom In</button>
    <button id="zoom-out" title="Zoom Out">- Zoom Out</button>
    <button id="zoom-reset" title="Reset Zoom">Reset</button>
    <button id="copy-src" title="Copy Mermaid Source">Copy Source</button>
  </div>

  <!-- Package Map tab -->
  <div class="tab-panel active full-width" id="panel-pkgmap">
    <div class="diagram-viewport">
      <div class="diagram-container" id="pkgmap-container">
        <pre class="mermaid" id="pkgmap-mermaid">{{.PackageMapMermaid}}</pre>
      </div>
    </div>
  </div>

  <!-- Package Map HTML tab -->
  <div class="tab-panel full-width" id="panel-pkgmap-html">
    <div class="treemap-viewport" id="pkgmap-html-viewport">
      <div class="treemap-container" id="pkgmap-html-container"></div>
    </div>
  </div>

  <div class="treemap-tooltip" id="treemap-tooltip"></div>

  <!-- Implementations tab -->
  <div class="tab-panel" id="panel-impls">
    <div class="entity-list" id="impls-list">
      <div class="entity-list-actions">
        <button id="impls-all" title="Select all implementations">All</button>
        <button id="impls-clear" title="Deselect all implementations">Clear</button>
      </div>
    </div>
    <div class="diagram-viewport">
      <div class="diagram-container" id="impls-diagram-container">
        <div class="placeholder-msg" id="impls-placeholder">Select items from the list to view their relationships</div>
        <pre class="mermaid" id="impls-mermaid" style="display:none;"></pre>
      </div>
    </div>
  </div>

  <!-- Interfaces tab -->
  <div class="tab-panel" id="panel-ifaces">
    <div class="entity-list" id="ifaces-list">
      <div class="entity-list-actions">
        <button id="ifaces-all" title="Select all interfaces">All</button>
        <button id="ifaces-clear" title="Deselect all interfaces">Clear</button>
      </div>
    </div>
    <div class="diagram-viewport">
      <div class="diagram-container" id="ifaces-diagram-container">
        <div class="placeholder-msg" id="ifaces-placeholder">Select items from the list to view their relationships</div>
        <pre class="mermaid" id="ifaces-mermaid" style="display:none;"></pre>
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
      var currentTab = 'pkgmap';
      var currentMermaidSource = '';
      var pkgMapRendered = false;
      var pkgMapHtmlRendered = false;

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

      var tooltip = document.getElementById('treemap-tooltip');
      var TREEMAP_GAP = 4;

      function renderTreemap(container, nodes, rect, depth, colorIdx) {
        if (!nodes || nodes.length === 0) {
          if (depth === 0) {
            container.innerHTML = '<div class="placeholder-msg">No packages found</div>';
          }
          return colorIdx;
        }

        var positioned = squarify(nodes, rect);
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
            var innerPad = 3;
            var innerRect = {x: innerPad, y: headerH, w: Math.max(0, gw - 2 * innerPad), h: Math.max(0, gh - headerH - innerPad)};

            // If this node is also a package, add a self node
            if (d.interfaces > 0 || d.types > 0) {
              var selfValue = d.interfaces + d.types;
              if (selfValue < 1) selfValue = 1;
              var childrenValue = 0;
              for (var k = 0; k < d.children.length; k++) childrenValue += d.children[k].value;
              var selfFraction = selfValue / (selfValue + childrenValue);
              var selfH = innerRect.h * selfFraction;

              var selfNode = document.createElement('div');
              selfNode.className = 'treemap-node';
              selfNode.style.left = innerRect.x + 'px';
              selfNode.style.top = innerRect.y + 'px';
              selfNode.style.width = innerRect.w + 'px';
              selfNode.style.height = Math.max(0, selfH) + 'px';
              selfNode.style.background = treemapPalette[(ci + 1) % treemapPalette.length].fill;
              selfNode.style.color = color.text;

              var sn = document.createElement('div');
              sn.className = 'tm-name';
              sn.textContent = d.relPath || d.name;
              selfNode.appendChild(sn);
              var ss = document.createElement('div');
              ss.className = 'tm-stats';
              ss.textContent = statsText(d);
              selfNode.appendChild(ss);
              attachTooltip(selfNode, d);
              group.appendChild(selfNode);

              innerRect = {x: innerRect.x, y: innerRect.y + selfH, w: innerRect.w, h: Math.max(0, innerRect.h - selfH)};
            }

            colorIdx = renderTreemap(group, d.children, innerRect, depth + 1, ci + 1);
            container.appendChild(group);
          } else {
            // Leaf node
            var node = document.createElement('div');
            node.className = 'treemap-node';
            node.style.left = (p.x + TREEMAP_GAP) + 'px';
            node.style.top = (p.y + TREEMAP_GAP) + 'px';
            node.style.width = Math.max(0, p.w - 2 * TREEMAP_GAP) + 'px';
            node.style.height = Math.max(0, p.h - 2 * TREEMAP_GAP) + 'px';
            node.style.background = color.fill;
            node.style.color = color.text;

            var nameEl = document.createElement('div');
            nameEl.className = 'tm-name';
            nameEl.textContent = d.relPath || d.name;
            node.appendChild(nameEl);
            var statsEl = document.createElement('div');
            statsEl.className = 'tm-stats';
            statsEl.textContent = statsText(d);
            node.appendChild(statsEl);
            attachTooltip(node, d);
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

      function positionTooltip(e) {
        tooltip.style.left = (e.clientX + 12) + 'px';
        tooltip.style.top = (e.clientY + 12) + 'px';
      }

      var resizeTimer = null;
      function layoutTreemap() {
        var viewport = document.getElementById('pkgmap-html-viewport');
        var container = document.getElementById('pkgmap-html-container');
        container.innerHTML = '';
        var w = viewport.clientWidth - 16;
        var h = viewport.clientHeight - 16;
        if (w <= 0 || h <= 0) return;
        var nodes = flattenTree(pkgMapData, 3);
        renderTreemap(container, nodes, {x: 0, y: 0, w: w, h: h}, 0, 0);
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

        if (tab === 'pkgmap' && !pkgMapRendered) {
          renderPackageMap();
        }
        if (tab === 'pkgmap-html' && !pkgMapHtmlRendered) {
          requestAnimationFrame(function() {
            layoutTreemap();
            pkgMapHtmlRendered = true;
          });
        }
      }

      function renderPackageMap() {
        var pre = document.getElementById('pkgmap-mermaid');
        mermaid.run({ nodes: [pre] }).then(function() {
          fixSvgWidth(pre);
          pkgMapRendered = true;
          currentMermaidSource = pre.getAttribute('data-original') || '';
        });
        // Save original source before mermaid replaces it
        pre.setAttribute('data-original', pre.textContent);
      }

      // Initial render of package map (deferred to let the page paint first)
      var pkgPre = document.getElementById('pkgmap-mermaid');
      pkgPre.setAttribute('data-original', pkgPre.textContent);
      pkgPre.textContent = 'Loading diagram...';
      requestAnimationFrame(function() {
        setTimeout(function() {
          pkgPre.textContent = pkgPre.getAttribute('data-original');
          mermaid.run({ nodes: [pkgPre] }).then(function() {
            fixSvgWidth(pkgPre);
            pkgMapRendered = true;
            currentMermaidSource = pkgPre.getAttribute('data-original');
          });
        }, 0);
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

      function onSelectionChange() {
        var selectedTypeIDs = [];
        document.querySelectorAll('.impl-cb:checked').forEach(function(cb) {
          selectedTypeIDs.push(cb.value);
        });
        var selectedIfaceIDs = [];
        document.querySelectorAll('.iface-cb:checked').forEach(function(cb) {
          selectedIfaceIDs.push(cb.value);
        });

        if (selectedTypeIDs.length === 0 && selectedIfaceIDs.length === 0) {
          showPlaceholder();
          currentMermaidSource = '';
          return;
        }

        var mermaidSrc = buildMermaid(selectedTypeIDs, selectedIfaceIDs);
        currentMermaidSource = mermaidSrc;
        renderSelectionDiagram(mermaidSrc);
      }

      function showPlaceholder() {
        ['impls', 'ifaces'].forEach(function(tab) {
          document.getElementById(tab + '-placeholder').style.display = 'block';
          document.getElementById(tab + '-mermaid').style.display = 'none';
        });
      }

      function renderSelectionDiagram(src) {
        ['impls', 'ifaces'].forEach(function(tab) {
          var placeholder = document.getElementById(tab + '-placeholder');
          var pre = document.getElementById(tab + '-mermaid');
          placeholder.style.display = 'none';
          // Reset the pre element for mermaid re-render
          pre.removeAttribute('data-processed');
          pre.innerHTML = '';
          pre.textContent = src;
          pre.style.display = 'block';
        });

        // Render the diagram in the currently visible tab
        var activePreId = currentTab === 'ifaces' ? 'ifaces-mermaid' : 'impls-mermaid';
        var activePre = document.getElementById(activePreId);
        try {
          mermaid.run({ nodes: [activePre] }).then(function() {
            fixSvgWidth(activePre);
            // Copy rendered SVG to the other tab's pre
            var otherPreId = activePreId === 'impls-mermaid' ? 'ifaces-mermaid' : 'impls-mermaid';
            var otherPre = document.getElementById(otherPreId);
            otherPre.innerHTML = activePre.innerHTML;
          }).catch(function(err) {
            activePre.textContent = src;
            activePre.style.whiteSpace = 'pre-wrap';
          });
        } catch(err) {
          activePre.textContent = src;
          activePre.style.whiteSpace = 'pre-wrap';
        }
      }

      function buildMermaid(selectedTypeIDs, selectedIfaceIDs) {
        var typeSet = {};
        selectedTypeIDs.forEach(function(id) { typeSet[id] = true; });
        var ifaceSet = {};
        selectedIfaceIDs.forEach(function(id) { ifaceSet[id] = true; });

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
        if (currentTab === 'pkgmap') return document.getElementById('pkgmap-container');
        if (currentTab === 'pkgmap-html') return document.getElementById('pkgmap-html-container');
        if (currentTab === 'ifaces') return document.getElementById('ifaces-diagram-container');
        return document.getElementById('impls-diagram-container');
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
        if (currentTab === 'pkgmap') {
          src = document.getElementById('pkgmap-mermaid').getAttribute('data-original') || '';
        } else if (currentTab === 'pkgmap-html') {
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
	PackageMapMermaid string
	DataJSON          template.JS
	PackageMapJSON    template.JS
	RepoAddress       string
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
		PackageMapMermaid: data.PackageMapMermaid,
		DataJSON:          template.JS(jsonBytes),   //nolint:gosec // JSON is generated from trusted internal data, not user input
		PackageMapJSON:    template.JS(pkgMapBytes), //nolint:gosec // JSON is generated from trusted internal data, not user input
		RepoAddress:       data.RepoAddress,
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

	mux.HandleFunc("/mermaid.md", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(data.PackageMapMermaid))
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
