package bot

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func FuzzCanonicalForgeParameters(f *testing.F) {
	f.Add([]byte(`{"target":10,"family":"offense"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, input []byte) {
		canonical, err := canonicalForgeParameters(json.RawMessage(input))
		if err != nil {
			return
		}
		second, err := canonicalForgeParameters(canonical)
		if err != nil || !bytes.Equal(canonical, second) {
			t.Fatalf("canonicalization is not idempotent: %q -> %q (%v)", canonical, second, err)
		}
	})
}

func FuzzDecodeBoundedForgeRequest(f *testing.F) {
	f.Add([]byte(`{"operation":"temper","inv_id":1,"parameters":{}}`))
	f.Add([]byte(`{"operation":"corrupt","slot":"main_hand"}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		request := httptest.NewRequest("POST", "/api/abyss/forge/quote", bytes.NewReader(input))
		var decoded abyssForgeQuoteRequest
		_ = decodeBoundedForgeRequest(request, &decoded)
	})
}
