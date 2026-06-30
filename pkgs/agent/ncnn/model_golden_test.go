package ncnn

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

const (
	ecapaGoldenCosineMin  = 0.999
	ecapaGoldenMeanAbsMax = 0.30
	ecapaGoldenMaxAbsMax  = 1.25
)

type ecapaGoldenFile struct {
	Model     string            `json:"model"`
	Generator string            `json:"generator"`
	Seed      int64             `json:"seed"`
	Cases     []ecapaGoldenCase `json:"cases"`
}

type ecapaGoldenCase struct {
	Name     string    `json:"name"`
	Frames   int       `json:"frames"`
	NumMels  int       `json:"num_mels"`
	Input    []float32 `json:"input"`
	Expected []float32 `json:"expected"`
}

func TestECAPAModelMatchesPythonGolden(t *testing.T) {
	requireNativeNCNNSupportedRuntime(t)

	golden := loadECAPAGolden(t)
	net, err := LoadModel(ModelSpeakerECAPA)
	if err != nil {
		t.Fatalf("LoadModel(%q): %v", ModelSpeakerECAPA, err)
	}
	defer func() {
		_ = net.Close()
	}()

	for _, tc := range golden.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Frames <= 0 || tc.NumMels <= 0 {
				t.Fatalf("invalid golden case shape: %+v", tc)
			}
			if got, want := len(tc.Input), tc.Frames*tc.NumMels; got != want {
				t.Fatalf("golden input len=%d, want %d", got, want)
			}
			if len(tc.Expected) == 0 {
				t.Fatal("golden expected output is empty")
			}

			mat, err := NewMat2D(tc.NumMels, tc.Frames, tc.Input)
			if err != nil {
				t.Fatalf("NewMat2D(%d,%d): %v", tc.NumMels, tc.Frames, err)
			}
			defer func() {
				_ = mat.Close()
			}()

			ex, err := net.NewExtractor()
			if err != nil {
				t.Fatalf("NewExtractor: %v", err)
			}
			defer func() {
				_ = ex.Close()
			}()

			if err := ex.SetInput("in0", mat); err != nil {
				t.Fatalf("SetInput: %v", err)
			}
			out, err := ex.Extract("out0")
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			defer func() {
				_ = out.Close()
			}()

			got := out.FloatData()
			if len(got) != len(tc.Expected) {
				t.Fatalf("output len=%d, want %d", len(got), len(tc.Expected))
			}

			cos, meanAbs, maxAbs := compareFloat32Vectors(got, tc.Expected)
			if cos < ecapaGoldenCosineMin || meanAbs > ecapaGoldenMeanAbsMax || maxAbs > ecapaGoldenMaxAbsMax {
				t.Fatalf(
					"golden mismatch: cosine=%.6f mean_abs=%.6f max_abs=%.6f (thresholds: cosine>=%.3f mean_abs<=%.2f max_abs<=%.2f)",
					cos,
					meanAbs,
					maxAbs,
					ecapaGoldenCosineMin,
					ecapaGoldenMeanAbsMax,
					ecapaGoldenMaxAbsMax,
				)
			}
		})
	}
}

func loadECAPAGolden(t *testing.T) ecapaGoldenFile {
	t.Helper()

	path := filepath.Join("testdata", "ecapa_golden.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}

	var golden ecapaGoldenFile
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("Unmarshal(%s): %v", path, err)
	}
	if len(golden.Cases) == 0 {
		t.Fatalf("golden file %s has no cases", path)
	}
	return golden
}

func compareFloat32Vectors(got, want []float32) (cosine, meanAbs, maxAbs float64) {
	var (
		dot, normGot, normWant float64
		sumAbs                 float64
	)
	for i := range got {
		gv := float64(got[i])
		wv := float64(want[i])
		diff := math.Abs(gv - wv)
		sumAbs += diff
		if diff > maxAbs {
			maxAbs = diff
		}
		dot += gv * wv
		normGot += gv * gv
		normWant += wv * wv
	}
	if len(got) > 0 {
		meanAbs = sumAbs / float64(len(got))
	}
	if normGot == 0 || normWant == 0 {
		return 0, meanAbs, maxAbs
	}
	cosine = dot / (math.Sqrt(normGot) * math.Sqrt(normWant))
	return cosine, meanAbs, maxAbs
}
