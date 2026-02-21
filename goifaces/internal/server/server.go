package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/olehluchkiv/goifaces/internal/diagram"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>goifaces — Interface Diagram</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      display: flex;
      flex-direction: column;
      align-items: center;
      min-height: 100vh;
      padding: 1rem;
      transition: background-color 0.3s, color 0.3s;
    }

    /* Light mode (default) */
    body {
      background-color: #f8f9fa;
      color: #212529;
    }

    /* Dark mode */
    @media (prefers-color-scheme: dark) {
      body {
        background-color: #1a1a2e;
        color: #e0e0e0;
      }
      .controls button {
        background-color: #2d2d44;
        color: #e0e0e0;
        border-color: #444;
      }
      .controls button:hover {
        background-color: #3d3d5c;
      }
    }

    h1 {
      margin: 1rem 0;
      font-size: 1.4rem;
      font-weight: 600;
    }

    .controls {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 1rem;
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

    .diagram-viewport {
      width: 100%;
      max-width: 100vw;
      overflow: auto;
      flex: 1;
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

    /* Override Mermaid's small default font sizes in class diagrams */
    .mermaid svg { font-size: 18px !important; }
    .mermaid svg g.classGroup text { font-size: 18px !important; }
    .mermaid svg .classTitleText { font-size: 28px !important; }
    .mermaid svg .nodeLabel { font-size: 18px !important; }
    .mermaid svg .edgeLabel { font-size: 16px !important; }
    .mermaid svg .label text { font-size: 18px !important; }

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
  <h1>goifaces — Interface Diagram</h1>

  <div class="controls">
    <button id="zoom-in" title="Zoom In">+ Zoom In</button>
    <button id="zoom-out" title="Zoom Out">- Zoom Out</button>
    <button id="zoom-reset" title="Reset Zoom">Reset</button>
    <button id="copy-src" title="Copy Mermaid Source">Copy Mermaid Source</button>
  </div>

  <div class="diagram-viewport">
    <div class="diagram-container" id="diagram-container">
      <pre class="mermaid">{{.MermaidContent}}</pre>
    </div>
  </div>

  <script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
  <script>
    mermaid.initialize({
      startOnLoad: true,
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
      var scale = 1;
      var step = 0.15;
      var minScale = 0.1;
      var maxScale = 10;
      var container = document.getElementById('diagram-container');

      function applyZoom() {
        container.style.transform = 'scale(' + scale + ')';
      }

      applyZoom();

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
        var src = {{.MermaidRaw}};
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

// Serve starts the HTTP server with the given Mermaid content.
// It blocks until the context is cancelled.
func Serve(ctx context.Context, mermaidContent string, port int, openBrowser bool, logger *slog.Logger) error {
	tmpl, err := template.New("diagram").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}

	data := struct {
		MermaidContent string
		MermaidRaw     string
	}{
		MermaidContent: mermaidContent,
		MermaidRaw:     mermaidContent,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			logger.Error("failed to render template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/mermaid.md", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(mermaidContent))
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	logger.Info("starting HTTP server", "addr", url)

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

	// Block until the context is cancelled or the server fails.
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

// slideEntry holds data for a single slide in the template.
type slideEntry struct {
	Index   int
	Title   string
	Mermaid string
}

// slidesData holds all data passed to the slides HTML template.
type slidesData struct {
	Slides     []slideEntry
	SlideCount int
}

const slidesHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>goifaces — Interface Diagram (Slides)</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      display: flex;
      flex-direction: column;
      align-items: center;
      min-height: 100vh;
      padding: 1rem;
      transition: background-color 0.3s, color 0.3s;
    }

    /* Light mode (default) */
    body {
      background-color: #f8f9fa;
      color: #212529;
    }

    /* Dark mode */
    @media (prefers-color-scheme: dark) {
      body {
        background-color: #1a1a2e;
        color: #e0e0e0;
      }
      .controls button, .controls select {
        background-color: #2d2d44;
        color: #e0e0e0;
        border-color: #444;
      }
      .controls button:hover {
        background-color: #3d3d5c;
      }
    }

    h1 {
      margin: 1rem 0;
      font-size: 1.4rem;
      font-weight: 600;
    }

    .controls {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 1rem;
      flex-wrap: wrap;
      justify-content: center;
      align-items: center;
    }

    .controls button, .controls select {
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

    .controls button:disabled {
      opacity: 0.4;
      cursor: default;
    }

    .slide-counter {
      font-size: 0.9rem;
      font-weight: 500;
      min-width: 4rem;
      text-align: center;
    }

    .slide-title {
      font-size: 1.1rem;
      font-weight: 600;
      margin-bottom: 0.5rem;
    }

    .diagram-viewport {
      width: 100%;
      max-width: 100vw;
      overflow: auto;
      flex: 1;
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

    .slide-panel {
      display: block;
    }
    .slide-panel.ready:not(.active) {
      display: none;
    }

    /* Override Mermaid's small default font sizes in class diagrams */
    .mermaid svg { font-size: 18px !important; }
    .mermaid svg g.classGroup text { font-size: 18px !important; }
    .mermaid svg .classTitleText { font-size: 28px !important; }
    .mermaid svg .nodeLabel { font-size: 18px !important; }
    .mermaid svg .edgeLabel { font-size: 16px !important; }
    .mermaid svg .label text { font-size: 18px !important; }

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
  <h1>goifaces — Interface Diagram</h1>

  <div class="controls">
    <button id="prev-btn" title="Previous Slide">Prev</button>
    <select id="slide-select">
      {{range .Slides}}<option value="{{.Index}}">{{.Title}}</option>
      {{end}}
    </select>
    <button id="next-btn" title="Next Slide">Next</button>
    <span class="slide-counter" id="slide-counter">1 / {{.SlideCount}}</span>
    <button id="zoom-in" title="Zoom In">+ Zoom In</button>
    <button id="zoom-out" title="Zoom Out">- Zoom Out</button>
    <button id="zoom-reset" title="Reset Zoom">Reset</button>
    <button id="copy-src" title="Copy Mermaid Source">Copy Source</button>
  </div>

  <div class="slide-title" id="slide-title"></div>

  <div class="diagram-viewport">
    <div class="diagram-container" id="diagram-container">
      {{range .Slides}}<div class="slide-panel" id="slide-{{.Index}}">
        <pre class="mermaid">{{.Mermaid}}</pre>
      </div>
      {{end}}
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
      var current = 0;
      var total = {{.SlideCount}};
      var titles = [{{range .Slides}}"{{.Title}}",{{end}}];
      var sources = [{{range .Slides}}` + "`" + `{{.Mermaid}}` + "`" + `,{{end}}];

      function showSlide(idx) {
        if (idx < 0) idx = 0;
        if (idx >= total) idx = total - 1;
        var panels = document.querySelectorAll('.slide-panel');
        for (var i = 0; i < panels.length; i++) {
          panels[i].classList.remove('active');
        }
        document.getElementById('slide-' + idx).classList.add('active');
        document.getElementById('slide-counter').textContent = (idx + 1) + ' / ' + total;
        document.getElementById('slide-title').textContent = titles[idx];
        document.getElementById('slide-select').value = idx;
        current = idx;
      }

      // All slides start visible so Mermaid can measure SVG text.
      // After rendering completes, fix SVG widths and add 'ready' class.
      mermaid.run().then(function() {
        // Set each SVG width based on its viewBox and available space.
        // If the SVG fits, render at natural pixel width for crisp text.
        // If it overflows, scale it down to fit the viewport.
        document.querySelectorAll('pre.mermaid svg').forEach(function(svg) {
          var vb = svg.getAttribute('viewBox');
          if (vb) {
            var w = parseFloat(vb.split(/\s+/)[2]);
            if (w > 0) {
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
          }
        });
        document.querySelectorAll('.slide-panel').forEach(function(p) {
          p.classList.add('ready');
        });
      });

      showSlide(0);

      document.getElementById('prev-btn').addEventListener('click', function() {
        showSlide(current - 1);
      });
      document.getElementById('next-btn').addEventListener('click', function() {
        showSlide(current + 1);
      });
      document.getElementById('slide-select').addEventListener('change', function() {
        showSlide(parseInt(this.value, 10));
      });

      document.addEventListener('keydown', function(e) {
        if (e.key === 'ArrowLeft') { showSlide(current - 1); }
        if (e.key === 'ArrowRight') { showSlide(current + 1); }
      });

      // Zoom
      var scale = 1;
      var step = 0.15;
      var minScale = 0.1;
      var maxScale = 10;
      var container = document.getElementById('diagram-container');

      function applyZoom() {
        container.style.transform = 'scale(' + scale + ')';
      }

      applyZoom();

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
        var src = sources[current];
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

// ServeSlides starts the HTTP server with paginated slide navigation.
// It blocks until the context is cancelled.
func ServeSlides(ctx context.Context, slides []diagram.Slide, port int, openBrowser bool, logger *slog.Logger) error {
	tmpl, err := template.New("slides").Parse(slidesHTMLTemplate)
	if err != nil {
		return fmt.Errorf("parsing slides HTML template: %w", err)
	}

	entries := make([]slideEntry, len(slides))
	for i, s := range slides {
		entries[i] = slideEntry{
			Index:   i,
			Title:   s.Title,
			Mermaid: s.Mermaid,
		}
	}

	data := slidesData{
		Slides:     entries,
		SlideCount: len(slides),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			logger.Error("failed to render slides template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/mermaid.md", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		idx := 0
		if q := r.URL.Query().Get("slide"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n >= 0 && n < len(slides) {
				idx = n
			}
		}
		_, _ = w.Write([]byte(slides[idx].Mermaid))
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	logger.Info("starting HTTP server (slides mode)", "addr", url, "slideCount", len(slides))

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

	// Block until the context is cancelled or the server fails.
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
