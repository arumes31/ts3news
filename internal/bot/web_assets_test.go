package bot

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssetVerAndURL(t *testing.T) {
	// 1. Check style.css version
	styleVer := AssetVer("webassets/style.css")
	if len(styleVer) != 12 {
		t.Errorf("expected 12-char hex hash for style.css, got %q", styleVer)
	}

	// 2. Check abyss_ui200.css version
	uiVer := AssetVer("webassets/abyss_ui200.css")
	if len(uiVer) != 12 {
		t.Errorf("expected 12-char hex hash for abyss_ui200.css, got %q", uiVer)
	}

	commandVer := AssetVer("webassets/abyss_command.css")
	if len(commandVer) != 12 {
		t.Errorf("expected 12-char hex hash for abyss_command.css, got %q", commandVer)
	}

	// Check abyss_live.css version
	liveVer := AssetVer("webassets/abyss_live.css")
	if len(liveVer) != 12 {
		t.Errorf("expected 12-char hex hash for abyss_live.css, got %q", liveVer)
	}

	// 3. Check favicon and logo
	favVer := AssetVer("webassets/favicon.svg")
	if len(favVer) != 12 {
		t.Errorf("expected 12-char hex hash for favicon.svg, got %q", favVer)
	}

	logoVer := AssetVer("webassets/logo.svg")
	if len(logoVer) != 12 {
		t.Errorf("expected 12-char hex hash for logo.svg, got %q", logoVer)
	}

	// 4. Check composite version "all"
	allVer := AssetVer("all")
	if len(allVer) != 12 {
		t.Errorf("expected 12-char hex hash for composite version 'all', got %q", allVer)
	}

	// 5. Test AssetURL with various paths
	wantStyleURL := "/static/style.css?v=" + styleVer
	if got := AssetURL("/static/style.css"); got != wantStyleURL {
		t.Errorf("AssetURL(/static/style.css) = %q, want %q", got, wantStyleURL)
	}

	wantCommandURL := "/static/abyss_command.css?v=" + commandVer
	if got := AssetURL("/static/abyss_command.css"); got != wantCommandURL {
		t.Errorf("AssetURL(/static/abyss_command.css) = %q, want %q", got, wantCommandURL)
	}

	wantLiveURL := "/static/abyss_live.css?v=" + liveVer
	if got := AssetURL("/static/abyss_live.css"); got != wantLiveURL {
		t.Errorf("AssetURL(/static/abyss_live.css) = %q, want %q", got, wantLiveURL)
	}

	wantFavURL := "/static/favicon.svg?v=" + favVer
	if got := AssetURL("/static/favicon.svg"); got != wantFavURL {
		t.Errorf("AssetURL(/static/favicon.svg) = %q, want %q", got, wantFavURL)
	}

	// AssetURL with empty path
	if got := AssetURL(""); got != "" {
		t.Errorf("AssetURL('') = %q, want empty", got)
	}

	// AssetURL with already existing query params
	if got := AssetURL("/static/style.css?foo=bar"); got != "/static/style.css?foo=bar&v="+styleVer {
		t.Errorf("AssetURL with existing query = %q", got)
	}
}

func TestAbyssSceneLayerRemainsTransparent(t *testing.T) {
	css, err := webAssets.ReadFile("webassets/abyss_command.css")
	if err != nil {
		t.Fatalf("read abyss command theme: %v", err)
	}
	source := string(css)
	start := strings.Index(source, ".abyss-command-page .ab-scene {")
	if start < 0 {
		t.Fatal("abyss scene theme rule is missing")
	}
	end := strings.Index(source[start:], "}")
	if end < 0 {
		t.Fatal("abyss scene theme rule is unterminated")
	}
	rule := source[start : start+end]
	if !strings.Contains(rule, "background: transparent;") {
		t.Fatalf("abyss scene must remain transparent instead of covering the stage: %q", rule)
	}
}

func TestAbyssWorkspaceTabs(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	planner, err := webAssets.ReadFile("webassets/abyss_forge_planner.html")
	if err != nil {
		t.Fatalf("read forge planner: %v", err)
	}

	source := string(page)
	wantMarkers := []string{
		`id="abyssSetBonuses" data-abyss-section="progression"`,
		`id="abyssSanctuaryUpgrades" data-abyss-section="progression"`,
		`id="abyssTokenShop" data-abyss-section="shop"`,
		`id="abyssTokenExchange" data-abyss-section="shop"`,
		`id="abyssWorkshop" data-abyss-section="forge"`,
		`id="abyssForgePanel" data-abyss-section="forge"`,
		`localStorage.setItem('abyss_section_tab', g.key)`,
	}
	for _, marker := range wantMarkers {
		if !strings.Contains(source, marker) {
			t.Errorf("Abyss workspace marker is missing: %s", marker)
		}
	}
	if !strings.Contains(string(planner), `id="forgePlanner" data-abyss-section="forge"`) {
		t.Error("forge planner must participate in the Forge workspace")
	}

	partial := strings.LastIndex(source, `{{if .ForgeWorkbenchEnabled}}{{template "abyss-forge-planner" .}}{{end}}`)
	initTabs := strings.LastIndex(source, "buildAbyssTabs();")
	if partial < 0 || initTabs < partial {
		t.Error("workspace tabs must initialize after the optional forge planner partial")
	}
}

func TestAbyssPixelArtAssets(t *testing.T) {
	tests := []struct {
		name string
		path string
		cols int
		rows int
	}{
		{
			name: "combat sprites",
			path: "webassets/abyss_combat_sprites.png",
			cols: 4,
			rows: 3,
		},
		{
			name: "enemy atlas",
			path: "webassets/abyss_enemy_atlas.png",
			cols: 6,
			rows: 6,
		},
		{
			name: "icon atlas",
			path: "webassets/abyss_icon_atlas.png",
			cols: 8,
			rows: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := webAssets.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read pixel art asset: %v", err)
			}
			if len(asset) < 50_000 {
				t.Fatalf("pixel art asset is unexpectedly small: %d bytes", len(asset))
			}
			if len(asset) < 8 || string(asset[:8]) != "\x89PNG\r\n\x1a\n" {
				t.Fatal("pixel art asset must be a PNG")
			}
			config, _, err := image.DecodeConfig(bytes.NewReader(asset))
			if err != nil {
				t.Fatalf("decode pixel art dimensions: %v", err)
			}
			if config.Width%tt.cols != 0 || config.Height%tt.rows != 0 {
				t.Fatalf(
					"pixel art dimensions %dx%d do not divide into a %dx%d atlas",
					config.Width,
					config.Height,
					tt.cols,
					tt.rows,
				)
			}
			if len(AssetVer(tt.path)) != 12 {
				t.Fatal("pixel art asset must receive a content hash")
			}
		})
	}
}

func TestAbyssPixelCombatTemplates(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatalf("read live combat template: %v", err)
	}
	pixel, err := webAssets.ReadFile("webassets/abyss_pixel.html")
	if err != nil {
		t.Fatalf("read pixel combat template: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_pixel.css")
	if err != nil {
		t.Fatalf("read pixel combat stylesheet: %v", err)
	}
	feedback, err := webAssets.ReadFile("webassets/abyss_combat_feedback.html")
	if err != nil {
		t.Fatalf("read combat feedback template: %v", err)
	}
	feedbackCSS, err := webAssets.ReadFile("webassets/abyss_combat_feedback.css")
	if err != nil {
		t.Fatalf("read combat feedback stylesheet: %v", err)
	}

	markers := []struct {
		name   string
		source string
		want   string
	}{
		{name: "page stylesheet", source: string(page), want: `{{asset "/static/abyss_pixel.css"}}`},
		{name: "page renderer partial", source: string(page), want: `{{template "abyssPixelCombatJS" .}}`},
		{name: "battlefield", source: string(live), want: `id="livePixelStage"`},
		{name: "renderer", source: string(pixel), want: "function renderLivePixelStage(state)"},
		{name: "bestiary classifier", source: string(pixel), want: "function liveEnemyArt(unit)"},
		{name: "deterministic fallback", source: string(pixel), want: "function liveNameHash(name)"},
		{name: "enemy atlas", source: string(page), want: `{{asset "/static/abyss_enemy_atlas.png"}}`},
		{name: "boss tier", source: string(css), want: ".ab-pixel-unit.boss-tier"},
		{name: "icon classifier", source: string(pixel), want: "function liveActionIconCell(option)"},
		{name: "reduced motion", source: string(css), want: "@media (prefers-reduced-motion: reduce)"},
		{name: "feedback stylesheet", source: string(page), want: `{{asset "/static/abyss_combat_feedback.css"}}`},
		{name: "feedback controls", source: string(live), want: `{{template "abyssCombatFeedbackControls" .}}`},
		{name: "feedback script", source: string(page), want: `{{template "abyssCombatFeedbackJS" .}}`},
		{name: "saved audio preference", source: string(feedback), want: "abyssCombatAudio"},
		{name: "gesture-gated audio", source: string(feedback), want: "function enableLiveCombatAudio()"},
		{name: "snapshot feedback", source: string(feedback), want: "function emitLiveCombatFeedback(state,previous,newLogs)"},
		{name: "impact styling", source: string(feedbackCSS), want: ".ab-pixel-stage.impact"},
		{name: "feedback reduced motion", source: string(feedbackCSS), want: "@media (prefers-reduced-motion: reduce)"},
	}
	for _, marker := range markers {
		if !strings.Contains(marker.source, marker.want) {
			t.Errorf("%s marker is missing", marker.name)
		}
	}
}

func TestIconVerAndURL(t *testing.T) {
	bicepsVer := IconVer("biceps")
	if len(bicepsVer) != 12 {
		t.Errorf("expected 12-char hex hash for biceps icon, got %q", bicepsVer)
	}

	wantURL := "/static/icons/biceps.svg?v=" + bicepsVer
	if got := IconURL("biceps"); got != wantURL {
		t.Errorf("IconURL(biceps) = %q, want %q", got, wantURL)
	}

	// Handles with .svg and path prefix
	if got := IconURL("/static/icons/biceps.svg"); got != wantURL {
		t.Errorf("IconURL(/static/icons/biceps.svg) = %q, want %q", got, wantURL)
	}

	if got := IconURL(""); got != "" {
		t.Errorf("IconURL('') = %q, want empty", got)
	}
}

func TestServeAssetHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		ServeAsset(w, r, "webassets/style.css", "text/css; charset=utf-8")
	})
	mux.HandleFunc("/static/icons/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/static/icons/")
		ServeAsset(w, r, "webassets/icons/"+name, "image/svg+xml")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := server.Client()

	// 1. Request versioned asset -> 200 OK with immutable Cache-Control and ETag
	styleVer := AssetVer("webassets/style.css")
	req, err := http.NewRequest("GET", server.URL+"/static/style.css?v="+styleVer, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /static/style.css: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/css; charset=utf-8", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want public, max-age=31536000, immutable", cc)
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Errorf("expected ETag header on static asset response")
	}

	// 2. Conditional request with If-None-Match -> 304 Not Modified
	req304, err := http.NewRequest("GET", server.URL+"/static/style.css?v="+styleVer, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req304.Header.Set("If-None-Match", etag)
	resp304, err := client.Do(req304)
	if err != nil {
		t.Fatalf("GET /static/style.css 304: %v", err)
	}
	defer func() { _ = resp304.Body.Close() }()

	if resp304.StatusCode != http.StatusNotModified {
		t.Errorf("status with matching If-None-Match = %d, want 304", resp304.StatusCode)
	}

	// 3. Unversioned request -> 200 OK with must-revalidate Cache-Control
	reqUnver, err := http.NewRequest("GET", server.URL+"/static/style.css", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	respUnver, err := client.Do(reqUnver)
	if err != nil {
		t.Fatalf("GET unversioned style.css: %v", err)
	}
	defer func() { _ = respUnver.Body.Close() }()

	if respUnver.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", respUnver.StatusCode)
	}
	if cc := respUnver.Header.Get("Cache-Control"); cc != "public, max-age=86400, must-revalidate" {
		t.Errorf("Cache-Control for unversioned request = %q, want public, max-age=86400, must-revalidate", cc)
	}

	// 4. Test SVG icon
	iconVer := IconVer("biceps")
	reqIcon, err := http.NewRequest("GET", server.URL+"/static/icons/biceps.svg?v="+iconVer, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	respIcon, err := client.Do(reqIcon)
	if err != nil {
		t.Fatalf("GET icon: %v", err)
	}
	defer func() { _ = respIcon.Body.Close() }()

	if respIcon.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", respIcon.StatusCode)
	}
	if ct := respIcon.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	if cc := respIcon.Header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want public, max-age=31536000, immutable", cc)
	}

	// 5. Test Non-existent asset -> 404
	req404, err := http.NewRequest("GET", server.URL+"/static/icons/non-existent-icon.svg", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp404, err := client.Do(req404)
	if err != nil {
		t.Fatalf("GET 404: %v", err)
	}
	defer func() { _ = resp404.Body.Close() }()

	if resp404.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp404.StatusCode)
	}
}

func TestHTMLAndAPIAntiCachingHeaders(t *testing.T) {
	ws, err := NewWebServer(&Bot{Cfg: nil})
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}

	// 1. Test HTML page rendering sets anti-caching headers
	rec := httptest.NewRecorder()
	ws.render(rec, "denied", map[string]any{"Title": "Access"})

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("render Content-Type = %q, want text/html", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache, no-store, must-revalidate" {
		t.Errorf("render Cache-Control = %q, want no-cache, no-store, must-revalidate", cc)
	}
	if p := rec.Header().Get("Pragma"); p != "no-cache" {
		t.Errorf("render Pragma = %q, want no-cache", p)
	}
	if exp := rec.Header().Get("Expires"); exp != "0" {
		t.Errorf("render Expires = %q, want 0", exp)
	}

	// 2. Test JSON API writeJSON sets anti-caching headers
	apiRec := httptest.NewRecorder()
	writeJSON(apiRec, map[string]any{"ok": true, "balance": 100})

	if ct := apiRec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("writeJSON Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	if cc := apiRec.Header().Get("Cache-Control"); cc != "no-store, no-cache, must-revalidate" {
		t.Errorf("writeJSON Cache-Control = %q, want no-store, no-cache, must-revalidate", cc)
	}
	if p := apiRec.Header().Get("Pragma"); p != "no-cache" {
		t.Errorf("writeJSON Pragma = %q, want no-cache", p)
	}

	var data map[string]any
	if err := json.Unmarshal(apiRec.Body.Bytes(), &data); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if data["ok"] != true {
		t.Errorf("json ok = %v, want true", data["ok"])
	}
}
