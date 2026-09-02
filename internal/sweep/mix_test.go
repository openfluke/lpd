package sweep

import (
	"testing"

	"github.com/openfluke/tide/permute"
)

func TestCNN2BitMNISTSize(t *testing.T) {
	modes := permute.AllModes()
	cells, cellLR, err := CNN2BitMNIST(0)
	if err != nil {
		t.Fatal(err)
	}
	wantUniform := len(modes) * (1 + 2*5) // cam 1 + cams 2–3 × CamSync
	wantMix := len(modes) * (len(modes) - 1) * len(MixPatterns) * 2 * 5
	want := (wantUniform + wantMix) * len(DefaultLRs)
	if len(cells) != want {
		t.Fatalf("cells=%d want %d (uniform %d + mix %d) × %d lrs",
			len(cells), want, wantUniform, wantMix, len(DefaultLRs))
	}
	if len(cellLR) != len(cells) {
		t.Fatalf("cellLR %d != cells %d", len(cellLR), len(cells))
	}
}

func TestExpandMixPatternBlock(t *testing.T) {
	got := expandMixPattern(4, permute.ModeSGD, permute.ModeTween, "block")
	if len(got) != 4 || got[0] != permute.ModeSGD || got[1] != permute.ModeSGD || got[2] != permute.ModeTween || got[3] != permute.ModeTween {
		t.Fatalf("block: %v", got)
	}
}
