package idely

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAudioBot_CommandURLAndAuth(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	ab := NewAudioBot(AudioBotConfig{BaseURL: srv.URL, UID: "abc", Token: "tok"})
	if err := ab.Move(0, 5); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if want := "/api/bot/use/0/(/bot/move/5)"; gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("abc:tok"))
	if gotAuth != want {
		t.Fatalf("auth = %q, want %q", gotAuth, want)
	}
}

func TestAudioBot_ConnectParsesId(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"Id":3,"Name":null,"Server":"host","Status":1}`))
	}))
	defer srv.Close()
	ab := NewAudioBot(AudioBotConfig{BaseURL: srv.URL})
	id, err := ab.Connect("217.154.216.239:9987")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if id != 3 {
		t.Fatalf("id = %d, want 3", id)
	}
	if want := "/api/bot/connect/to/217.154.216.239:9987"; gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}
}

func TestAudioBot_TokenMarkerNormalised(t *testing.T) {
	ab := NewAudioBot(AudioBotConfig{BaseURL: "http://x", UID: "abc", Token: "abc:ts3ab:secret"})
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("abc:abc:secret"))
	if ab.authHdr != want {
		t.Fatalf("authHdr = %q, want %q", ab.authHdr, want)
	}
}

func TestAudioBot_ArgsEscaped(t *testing.T) {
	var gotRaw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw = r.URL.EscapedPath()
	}))
	defer srv.Close()
	ab := NewAudioBot(AudioBotConfig{BaseURL: srv.URL})
	if err := ab.Play(2, "lofi tracks/rainy day.wav"); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if want := "/api/bot/use/2/(/play/lofi%20tracks%2Frainy%20day.wav)"; gotRaw != want {
		t.Fatalf("escaped path = %q, want %q", gotRaw, want)
	}
}

func TestAudioBot_HTTPErrorSurfaced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ErrorName":"CommandError"}`))
	}))
	defer srv.Close()
	ab := NewAudioBot(AudioBotConfig{BaseURL: srv.URL})
	if err := ab.Stop(0); err == nil {
		t.Fatal("expected error on HTTP 400")
	}
}
