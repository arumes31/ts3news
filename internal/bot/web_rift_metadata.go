package bot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

// riftPublicMetadata is an explicit public allowlist. Never add character or run data.
func riftPublicMetadata(now time.Time) map[string]any {
	return map[string]any{"class_names": riftClassNames(), "class_options": content.AbyssClasses(), "rooms": rift.Rooms, "levels": rift.Campaign(), "bestiary": riftBestiary(now), "rarities": riftRarities()}
}
func handleRiftMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "GET or HEAD only", http.StatusMethodNotAllowed)
		return
	}
	writeRiftMetadata(w, r, riftPublicMetadata(time.Now()))
}
func writeRiftMetadata(w http.ResponseWriter, r *http.Request, metadata map[string]any) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "GET or HEAD only", http.StatusMethodNotAllowed)
		return
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "Metadata unavailable", 500)
		return
	}
	tag := fmt.Sprintf(`"%x"`, sha256.Sum256(body))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	w.Header().Set("ETag", tag)
	for _, candidate := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == tag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}
