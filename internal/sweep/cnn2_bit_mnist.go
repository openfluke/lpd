// Package sweep builds permutation plans for LPD tests.
package sweep

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openfluke/tide/permute"
	"github.com/openfluke/tide/river"
	"github.com/openfluke/welvet/core"
	"github.com/openfluke/welvet/quant"
)

// CamSyncAlphas is the CamSync pull spectrum (1%, 10%, 50%, 75%, 100%).
var CamSyncAlphas = []float64{0.01, 0.10, 0.50, 0.75, 1.00}

// CamSyncLabels for display.
var CamSyncLabels = []string{"1%", "10%", "50%", "75%", "100%"}

// CNN2BitMNIST is cam 1–3 × CamSync × all uniform train modes × cam-mix pairs × DefaultLRs.
func CNN2BitMNIST(_ float64) ([]permute.Cell, map[string]float64, error) {
	modes := permute.AllModes()
	base := permute.Expand(permute.Config{
		DTypes:  []core.DType{core.DTypeBinary},
		Formats: []quant.Format{quant.FormatNone},
		Modes:   modes,
		Cams:    permute.CamsRange(1, 3),
	})
	uniform := expandCamSync(base, CamSyncAlphas)
	mix := expandMix(modes, CamSyncAlphas)
	cells := append(uniform, mix...)
	cells, cellLR := river.ExpandWithLRs(cells, DefaultLRs)
	return cells, cellLR, nil
}

func expandCamSync(base []permute.Cell, alphas []float64) []permute.Cell {
	out := make([]permute.Cell, 0, len(base)*len(alphas))
	for _, c := range base {
		n := c.Cams
		if n <= 0 {
			n = permute.CamsOf(c.Arch)
		}
		if n < 2 {
			// Cam 1 has no parallel sync — one cell without cs= tag.
			out = append(out, c)
			continue
		}
		for _, a := range alphas {
			nc := c
			nc.ID = fmt.Sprintf("%s|cs=%s", c.ID, formatCS(a))
			out = append(out, nc)
		}
	}
	return out
}

func formatCS(a float64) string {
	return strconv.FormatFloat(a, 'f', -1, 64)
}

// ParseCS reads CamSync α from a cell ID (cs=0.10 → 0.10).
func ParseCS(id string) (float64, bool) {
	for _, p := range strings.Split(id, "|") {
		if strings.HasPrefix(p, "cs=") {
			v, err := strconv.ParseFloat(strings.TrimPrefix(p, "cs="), 64)
			return v, err == nil
		}
	}
	return 0, false
}

// Shard splits cells across machines (shard 0..nShards-1).
func Shard(cells []permute.Cell, shard, nShards int) []permute.Cell {
	if nShards <= 1 {
		return append([]permute.Cell(nil), cells...)
	}
	if shard < 0 || shard >= nShards {
		return nil
	}
	out := make([]permute.Cell, 0, len(cells)/nShards+1)
	for i, c := range cells {
		if i%nShards == shard {
			out = append(out, c)
		}
	}
	return out
}
