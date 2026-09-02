package sweep

import (
	"fmt"
	"strings"

	"github.com/openfluke/tide/permute"
	"github.com/openfluke/welvet/core"
	"github.com/openfluke/welvet/quant"
)

// DefaultLRs is the CNN2/bit/mnist learning-rate sweep.
var DefaultLRs = []float64{0.005, 0.05, 0.5}

// MixPatterns assign two parent modes across N cams (alt / block / roundrobin).
var MixPatterns = []string{"alt", "block", "roundrobin"}

// expandMix builds cam-mix cells for cams 2–6: distinct ordered mode pairs × patterns × CamSync.
func expandMix(modes []permute.TrainMode, alphas []float64) []permute.Cell {
	var out []permute.Cell
	for n := 2; n <= 6; n++ {
		arch := permute.ArchForCams(n)
		for _, pat := range MixPatterns {
			for _, m0 := range modes {
				for _, m1 := range modes {
					if m0 == m1 {
						continue
					}
					branch := expandMixPattern(n, m0, m1, pat)
					c := mixCell(n, arch, branch, pat)
					for _, a := range alphas {
						nc := c
						nc.ID = fmt.Sprintf("%s|cs=%s", c.ID, formatCS(a))
						out = append(out, nc)
					}
				}
			}
		}
	}
	return out
}

func mixCell(n int, arch permute.ArchKind, branch []permute.TrainMode, pat string) permute.Cell {
	c := permute.Cell{
		DType:   core.DTypeBinary,
		Format:  quant.FormatNone,
		Mode:    permute.ModeSGD, // parent NormalBP; per-cam modes live in bm=
		Arch:    arch,
		Cams:    n,
		Backend: core.BackendSIMD,
		UseSIMD: true,
	}
	tokens := make([]string, len(branch))
	for i, m := range branch {
		tokens[i] = string(m)
	}
	c.ID = c.String() + "|bm=" + strings.Join(tokens, "+") + "|pat=" + pat
	return c
}

func expandMixPattern(n int, m0, m1 permute.TrainMode, pat string) []permute.TrainMode {
	out := make([]permute.TrainMode, n)
	switch pat {
	case "block":
		half := n / 2
		for i := 0; i < n; i++ {
			if i < half {
				out[i] = m0
			} else {
				out[i] = m1
			}
		}
	case "roundrobin":
		for i := 0; i < n; i++ {
			if i%2 == 0 {
				out[i] = m1
			} else {
				out[i] = m0
			}
		}
	default: // alt
		for i := 0; i < n; i++ {
			if i%2 == 0 {
				out[i] = m0
			} else {
				out[i] = m1
			}
		}
	}
	return out
}

// ParseBM reads per-cam train tokens from bm=m0+m1+… in a cell ID.
func ParseBM(id string) ([]string, bool) {
	for _, p := range strings.Split(id, "|") {
		if strings.HasPrefix(p, "bm=") {
			raw := strings.TrimPrefix(p, "bm=")
			var out []string
			for _, m := range strings.Split(raw, "+") {
				m = strings.TrimSpace(m)
				if m != "" {
					out = append(out, m)
				}
			}
			if len(out) > 0 {
				return out, true
			}
		}
	}
	return nil, false
}
