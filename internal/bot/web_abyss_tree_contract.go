package bot

import (
	"net/http"
	"strings"

	"ts3news/internal/content"
)

const abyssTreeLayoutHeader = "X-Abyss-Tree-Layout"

type abyssTreeHandler func(http.ResponseWriter, *http.Request, string)

// requireAbyssTreeLayout rejects an explicitly versioned request from a stale
// browser tab. The header remains optional for compatibility with older API
// clients; the first refreshed page starts sending it on every mutation.
func (s *WebServer) requireAbyssTreeLayout(next abyssTreeHandler) abyssTreeHandler {
	return func(w http.ResponseWriter, r *http.Request, uid string) {
		provided := strings.TrimSpace(r.Header.Get(abyssTreeLayoutHeader))
		current := content.AbyssTree().TopologyHash()
		if provided != "" && provided != current {
			writeJSON(w, map[string]any{
				"ok": false, "error": "skill tree layout changed; refresh before applying changes",
				"layout_hash": current,
			})
			return
		}
		next(w, r, uid)
	}
}
