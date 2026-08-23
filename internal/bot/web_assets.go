package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
	"sync"
)

var (
	assetMu       sync.RWMutex
	assetHashes   = make(map[string]string)
	allAssetsVer  string
	assetsInitOnce sync.Once
)

// initAssetHashes scans the embedded webAssets file system and computes
// a content hash (SHA-256 truncated to 12 hex chars) for every asset.
func initAssetHashes() {
	assetsInitOnce.Do(func() {
		hasher := sha256.New()
		var paths []string

		_ = fs.WalkDir(webAssets, "webassets", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			b, err := webAssets.ReadFile(p)
			if err != nil {
				return nil
			}
			sum := sha256.Sum256(b)
			hashStr := hex.EncodeToString(sum[:6]) // 12 hex chars

			// Store normalized keys
			cleanPath := path.Clean(p)
			assetHashes[cleanPath] = hashStr

			// Also store without "webassets/" prefix and with "/static/" prefix for convenience
			rel := strings.TrimPrefix(cleanPath, "webassets/")
			assetHashes[rel] = hashStr
			assetHashes["/static/"+rel] = hashStr

			paths = append(paths, cleanPath)
			return nil
		})

		// Sort paths deterministically to compute composite asset version
		sort.Strings(paths)
		for _, p := range paths {
			hasher.Write([]byte(p + ":" + assetHashes[p] + ";"))
		}
		composite := sha256.Sum256(hasher.Sum(nil))
		allAssetsVer = hex.EncodeToString(composite[:6])
	})
}

func normalizeAssetPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Split(p, "?")[0] // strip existing query
	p = strings.Split(p, "#")[0] // strip fragment
	p = strings.TrimPrefix(p, "/")
	if !strings.HasPrefix(p, "webassets/") {
		if strings.HasPrefix(p, "static/") {
			p = "webassets/" + strings.TrimPrefix(p, "static/")
		}
	}
	return path.Clean(p)
}

// AssetVer returns the content hash for a given asset path or subpath.
// If path is "all" or empty, returns the composite asset version.
func AssetVer(p string) string {
	initAssetHashes()
	if p == "" || p == "all" {
		return allAssetsVer
	}
	norm := normalizeAssetPath(p)

	assetMu.RLock()
	h, ok := assetHashes[norm]
	if !ok {
		// Try without webassets/
		rel := strings.TrimPrefix(norm, "webassets/")
		h, ok = assetHashes[rel]
	}
	assetMu.RUnlock()

	if ok && h != "" {
		return h
	}

	// Dynamic fallback: read directly from webAssets if not indexed
	b, err := webAssets.ReadFile(norm)
	if err == nil {
		sum := sha256.Sum256(b)
		h = hex.EncodeToString(sum[:6])
		assetMu.Lock()
		assetHashes[norm] = h
		assetMu.Unlock()
		return h
	}

	return allAssetsVer
}

// AssetURL appends a ?v=<hash> cache-busting query parameter to an asset URL.
func AssetURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}
	v := AssetVer(urlStr)
	if v == "" {
		return urlStr
	}
	if strings.Contains(urlStr, "?") {
		return urlStr + "&v=" + v
	}
	return urlStr + "?v=" + v
}

// IconVer returns the content hash for a game icon by name (e.g. "biceps" or "biceps.svg").
func IconVer(name string) string {
	clean := strings.TrimSpace(name)
	clean = strings.TrimPrefix(clean, "/static/icons/")
	clean = strings.TrimPrefix(clean, "icons/")
	clean = strings.TrimSuffix(clean, ".svg")
	if clean == "" {
		return AssetVer("all")
	}
	return AssetVer("webassets/icons/" + clean + ".svg")
}

// IconURL returns a versioned URL for a game icon: /static/icons/<name>.svg?v=<hash>.
func IconURL(name string) string {
	clean := strings.TrimSpace(name)
	clean = strings.TrimPrefix(clean, "/static/icons/")
	clean = strings.TrimPrefix(clean, "icons/")
	clean = strings.TrimSuffix(clean, ".svg")
	if clean == "" {
		return ""
	}
	v := IconVer(clean)
	return fmt.Sprintf("/static/icons/%s.svg?v=%s", clean, v)
}

// ServeAsset serves an embedded static asset from webAssets with RFC-compliant
// ETag, If-None-Match, and Cache-Control headers.
func ServeAsset(w http.ResponseWriter, r *http.Request, embedPath string, contentType string) {
	initAssetHashes()
	b, err := webAssets.ReadFile(embedPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	hash := AssetVer(embedPath)
	etag := fmt.Sprintf("%q", hash)

	w.Header().Set("ETag", etag)
	w.Header().Set("Vary", "Accept-Encoding")

	// Check conditional requests
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch != "" {
		for _, part := range strings.Split(ifNoneMatch, ",") {
			part = strings.TrimSpace(part)
			if part == etag || part == "*" || (len(part) >= 2 && strings.Trim(part, `"`) == hash) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	w.Header().Set("Content-Type", contentType)

	// Long-lived immutable cache if version parameter is present, else revalidating cache
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400, must-revalidate")
	}

	_, _ = w.Write(b)
}
