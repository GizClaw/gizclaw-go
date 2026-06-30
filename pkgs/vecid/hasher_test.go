package vecid

import (
	"encoding/json"
	"testing"
)

func TestHasherDeterministic(t *testing.T) {
	h := NewHasher(192, 16, 42)

	emb := make([]float32, 192)
	for i := range emb {
		emb[i] = float32(i) * 0.01
	}

	hash1 := h.Hash(emb)
	hash2 := h.Hash(emb)
	if hash1 != hash2 {
		t.Fatalf("same embedding produced different hashes: %q vs %q", hash1, hash2)
	}
	if len(hash1) != 4 {
		t.Fatalf("expected 4 hex chars, got %d: %q", len(hash1), hash1)
	}
}

func TestHasherSeedMatters(t *testing.T) {
	emb := make([]float32, 192)
	for i := range emb {
		emb[i] = float32(i) * 0.01
	}

	h1 := NewHasher(192, 16, 1)
	h2 := NewHasher(192, 16, 2)
	if h1.Hash(emb) == h2.Hash(emb) {
		t.Log("different seeds produced the same 16-bit hash; unlikely but possible")
	}
}

func TestHasherHexFormat(t *testing.T) {
	h := NewHasher(8, 16, 99)
	hash := h.Hash([]float32{1, 2, 3, 4, 5, 6, 7, 8})

	if len(hash) != 4 {
		t.Fatalf("expected length 4, got %d: %q", len(hash), hash)
	}
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			t.Fatalf("non-uppercase-hex char %c in %q", c, hash)
		}
	}
}

func TestHasherMultiPrecision(t *testing.T) {
	h := NewHasher(192, 16, 42)
	emb := make([]float32, 192)
	for i := range emb {
		emb[i] = float32(i) * 0.01
	}

	full := h.Hash(emb)
	if len(full) != 4 {
		t.Fatalf("expected 4 chars, got %d", len(full))
	}
	coarse12 := full[:3]
	coarse8 := full[:2]
	coarse4 := full[:1]
	if coarse12 != full[:3] || coarse8 != full[:2] || coarse4 != full[:1] {
		t.Fatal("prefix truncation broken")
	}
}

func TestHasherPanics(t *testing.T) {
	t.Run("bad_bits", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for bits=3")
			}
		}()
		NewHasher(192, 3, 0)
	})

	t.Run("bad_dim", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for dim=0")
			}
		}()
		NewHasher(0, 16, 0)
	})

	t.Run("dim_mismatch", func(t *testing.T) {
		h := NewHasher(192, 16, 0)
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for wrong embedding dimension")
			}
		}()
		h.Hash([]float32{1, 2, 3})
	})
}

func TestHasherAccessors(t *testing.T) {
	h := NewHasher(192, 16, 0)
	if h.Bits() != 16 {
		t.Fatalf("Bits() = %d, want 16", h.Bits())
	}
	if h.Dim() != 192 {
		t.Fatalf("Dim() = %d, want 192", h.Dim())
	}
}

func TestHasherFromPlanesSameAsSeed(t *testing.T) {
	seed := NewHasher(512, 16, 42)
	loaded := NewHasherFromPlanes(seed.Planes())

	emb := make([]float32, 512)
	for i := range emb {
		emb[i] = float32(i) * 0.01
	}

	hash1 := seed.Hash(emb)
	hash2 := loaded.Hash(emb)
	if hash1 != hash2 {
		t.Fatalf("seed hash %q != planes hash %q", hash1, hash2)
	}
	if hash1 != "82A9" {
		t.Fatalf("expected reference hash 82A9, got %q", hash1)
	}
}

func TestHasherFromJSON(t *testing.T) {
	seed := NewHasher(512, 16, 42)

	emb := make([]float32, 512)
	for i := range emb {
		emb[i] = float32(i) * 0.01
	}
	expectedHash := seed.Hash(emb)

	data, err := json.Marshal(PlanesFile{
		Dim:    seed.Dim(),
		Bits:   seed.Bits(),
		Seed:   42,
		Planes: seed.Planes(),
	})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := NewHasherFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if hash := loaded.Hash(emb); hash != expectedHash {
		t.Fatalf("JSON-loaded hash %q != expected %q", hash, expectedHash)
	}
}

func TestHasherFromJSONRejectsEmptyPlanes(t *testing.T) {
	_, err := NewHasherFromJSON([]byte(`{"planes":[]}`))
	if err == nil {
		t.Fatal("expected error for empty planes")
	}
}

func TestHasherFromJSONRejectsInvalidJSON(t *testing.T) {
	_, err := NewHasherFromJSON([]byte(`{`))
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestHasherFromPlanesPanics(t *testing.T) {
	t.Run("bad_bits", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for zero planes")
			}
		}()
		NewHasherFromPlanes(nil)
	})

	t.Run("inconsistent_dimensions", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for inconsistent plane dimensions")
			}
		}()
		NewHasherFromPlanes([][]float32{{1, 2}, {3}})
	})
}
