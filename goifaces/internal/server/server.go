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
  </style>
</head>
<body>
  <h1>goifaces — {{.RepoAddress}}</h1>

  <div class="tab-bar">
    <button class="tab-btn active" data-tab="pkgmap">Package Map</button>
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
      var currentTab = 'pkgmap';
      var currentMermaidSource = '';
      var pkgMapRendered = false;

      // Build checkbox lists
      var implsList = document.getElementById('impls-list');
      var ifacesList = document.getElementById('ifaces-list');

      data.types.forEach(function(t) {
        var label = document.createElement('label');
        var cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.value = t.id;
        cb.className = 'impl-cb';
        cb.addEventListener('change', onSelectionChange);
        var span = document.createElement('span');
        span.innerHTML = t.name + ' <span class="pkg-name">' + t.pkgName + '</span>';
        label.appendChild(cb);
        label.appendChild(span);
        implsList.appendChild(label);
      });

      data.interfaces.forEach(function(iface) {
        var label = document.createElement('label');
        var cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.value = iface.id;
        cb.className = 'iface-cb';
        cb.addEventListener('change', onSelectionChange);
        var span = document.createElement('span');
        span.innerHTML = iface.name + ' <span class="pkg-name">' + iface.pkgName + '</span>';
        label.appendChild(cb);
        label.appendChild(span);
        ifacesList.appendChild(label);
      });

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

      // Initial render of package map
      var pkgPre = document.getElementById('pkgmap-mermaid');
      pkgPre.setAttribute('data-original', pkgPre.textContent);
      mermaid.run({ nodes: [pkgPre] }).then(function() {
        fixSvgWidth(pkgPre);
        pkgMapRendered = true;
        currentMermaidSource = pkgPre.getAttribute('data-original');
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
    })();
  </script>
</body>
</html>
`

// interactiveData holds all data passed to the interactive HTML template.
type interactiveData struct {
	PackageMapMermaid string
	DataJSON          template.JS
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

	templateData := interactiveData{
		PackageMapMermaid: data.PackageMapMermaid,
		DataJSON:          template.JS(jsonBytes), //nolint:gosec // JSON is generated from trusted internal data, not user input
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
