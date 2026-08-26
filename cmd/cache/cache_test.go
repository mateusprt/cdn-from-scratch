// internal/cache/cache_test.go
package cache

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New()
	entry := Entry{
		Body:       []byte(`{"ok":true}`),
		StatusCode: 200,
		ExpiresAt:  time.Now().Add(1 * time.Minute),
	}

	c.Set("/foo.json", entry)

	got, ok := c.Get("/foo.json")
	if !ok {
		t.Fatal("esperava encontrar a entrada, não encontrou")
	}
	if string(got.Body) != string(entry.Body) {
		t.Errorf("body diferente: got %q, want %q", got.Body, entry.Body)
	}
	if got.StatusCode != entry.StatusCode {
		t.Errorf("status code diferente: got %d, want %d", got.StatusCode, entry.StatusCode)
	}
}

func TestGetMissingKey(t *testing.T) {
	c := New()

	_, ok := c.Get("/nao-existe.json")
	if ok {
		t.Fatal("esperava ok=false para chave inexistente")
	}
}

func TestGetExpiredEntry(t *testing.T) {
	c := New()
	c.Set("/foo.json", Entry{
		Body:      []byte("velho"),
		ExpiresAt: time.Now().Add(-1 * time.Second), // já expirado
	})

	_, ok := c.Get("/foo.json")
	if ok {
		t.Fatal("esperava ok=false para entrada expirada")
	}

	// além de retornar false, Get deveria ter limpado a entrada expirada
	if c.Len() != 0 {
		t.Errorf("esperava que a entrada expirada fosse removida, Len()=%d", c.Len())
	}
}

func TestSetOverwritesExistingEntry(t *testing.T) {
	c := New()
	c.Set("/foo.json", Entry{Body: []byte("v1"), ExpiresAt: time.Now().Add(time.Minute)})
	c.Set("/foo.json", Entry{Body: []byte("v2"), ExpiresAt: time.Now().Add(time.Minute)})

	got, ok := c.Get("/foo.json")
	if !ok {
		t.Fatal("esperava encontrar a entrada")
	}
	if string(got.Body) != "v2" {
		t.Errorf("esperava a versão mais nova (v2), veio %q", got.Body)
	}
}

func TestDelete(t *testing.T) {
	c := New()
	c.Set("/foo.json", Entry{Body: []byte("x"), ExpiresAt: time.Now().Add(time.Minute)})

	c.Delete("/foo.json")

	_, ok := c.Get("/foo.json")
	if ok {
		t.Fatal("esperava que a entrada tivesse sido removida")
	}
}

func TestLen(t *testing.T) {
	c := New()
	if c.Len() != 0 {
		t.Fatalf("cache novo deveria ter Len()=0, veio %d", c.Len())
	}

	c.Set("/a", Entry{ExpiresAt: time.Now().Add(time.Minute)})
	c.Set("/b", Entry{ExpiresAt: time.Now().Add(time.Minute)})

	if c.Len() != 2 {
		t.Errorf("esperava Len()=2, veio %d", c.Len())
	}
}

// TestConcurrentAccess roda com -race para garantir que não há data race
// entre goroutines lendo e escrevendo ao mesmo tempo.
func TestConcurrentAccess(t *testing.T) {
	c := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.Set("/key", Entry{
				Body:      []byte("x"),
				ExpiresAt: time.Now().Add(time.Minute),
			})
		}(i)
		go func() {
			defer wg.Done()
			c.Get("/key")
		}()
	}

	wg.Wait()
}

func TestTTLFromHeaders_MaxAge(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "public, max-age=42")

	got := TTLFromHeaders(h)
	want := 42 * time.Second

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTTLFromHeaders_NoHeader(t *testing.T) {
	h := http.Header{}

	got := TTLFromHeaders(h)
	want := 10 * time.Second // defaultTTL

	if got != want {
		t.Errorf("sem Cache-Control, esperava default %v, veio %v", want, got)
	}
}

func TestTTLFromHeaders_NoStore(t *testing.T) {
	cases := []string{"no-store", "no-cache", "public, no-store"}

	for _, cc := range cases {
		h := http.Header{}
		h.Set("Cache-Control", cc)

		got := TTLFromHeaders(h)
		if got != 0 {
			t.Errorf("Cache-Control=%q deveria dar TTL 0, veio %v", cc, got)
		}
	}
}

func TestTTLFromHeaders_InvalidMaxAge(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "max-age=not-a-number")

	got := TTLFromHeaders(h)
	want := 10 * time.Second // cai no default por não conseguir parsear

	if got != want {
		t.Errorf("max-age inválido deveria cair no default %v, veio %v", want, got)
	}
}

func TestTTLFromHeaders_NegativeMaxAge(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "max-age=-5")

	got := TTLFromHeaders(h)
	want := 10 * time.Second

	if got != want {
		t.Errorf("max-age negativo deveria cair no default %v, veio %v", want, got)
	}
}

func TestTTLFromHeaders_CaseInsensitive(t *testing.T) {
	h := http.Header{}
	h.Set("Cache-Control", "PUBLIC, MAX-AGE=30")

	got := TTLFromHeaders(h)
	want := 30 * time.Second

	if got != want {
		t.Errorf("deveria ser case-insensitive, got %v, want %v", got, want)
	}
}
