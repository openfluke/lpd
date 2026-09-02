// Package mnist loads MNIST into downloads/mnist (shared cache for all LPD tests).
package mnist

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
)

const (
	baseURL    = "https://storage.googleapis.com/cvdf-datasets/mnist/"
	SampleSeed = uint64(0x4D4E4953) // "MNIS" — same subset as ok MNIST hosts
)

// CacheDir is where IDX gzip files are stored (under repo downloads/).
func CacheDir() string {
	return filepath.Join("downloads", "mnist")
}

// Image is a 28×28 grayscale sample.
type Image struct {
	Pixels []float32
	Label  int
}

// Split is an 80/20 partition of the official training set.
type Split struct {
	Train []Image
	Val   []Image
	Test  []Image
}

// Load downloads (if needed) and returns the train/val/test split.
func Load(dir string) (*Split, error) {
	if dir == "" {
		dir = CacheDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	for _, f := range []string{
		"train-images-idx3-ubyte.gz",
		"train-labels-idx1-ubyte.gz",
		"t10k-images-idx3-ubyte.gz",
		"t10k-labels-idx1-ubyte.gz",
	} {
		if err := ensure(dir, f); err != nil {
			return nil, err
		}
	}
	trainX, err := readImages(filepath.Join(dir, "train-images-idx3-ubyte.gz"))
	if err != nil {
		return nil, err
	}
	trainY, err := readLabels(filepath.Join(dir, "train-labels-idx1-ubyte.gz"))
	if err != nil {
		return nil, err
	}
	testX, err := readImages(filepath.Join(dir, "t10k-images-idx3-ubyte.gz"))
	if err != nil {
		return nil, err
	}
	testY, err := readLabels(filepath.Join(dir, "t10k-labels-idx1-ubyte.gz"))
	if err != nil {
		return nil, err
	}
	all := make([]Image, len(trainX))
	for i := range trainX {
		all[i] = Image{Pixels: trainX[i], Label: int(trainY[i])}
	}
	train, val := make([]Image, 0, len(all)*4/5), make([]Image, 0, len(all)/5)
	for i, im := range all {
		if i%5 == 0 {
			val = append(val, im)
		} else {
			train = append(train, im)
		}
	}
	test := make([]Image, len(testX))
	for i := range testX {
		test[i] = Image{Pixels: testX[i], Label: int(testY[i])}
	}
	return &Split{Train: train, Val: val, Test: test}, nil
}

// TakeBalanced returns a class-balanced subset (deterministic for seed).
func TakeBalanced(in []Image, n int, seed uint64) []Image {
	const nClass = 10
	if n <= 0 || n >= len(in) {
		return in
	}
	by := make([][]int, nClass)
	for i, im := range in {
		if im.Label >= 0 && im.Label < nClass {
			by[im.Label] = append(by[im.Label], i)
		}
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x51B5E7))
	for c := range by {
		rng.Shuffle(len(by[c]), func(i, j int) { by[c][i], by[c][j] = by[c][j], by[c][i] })
	}
	per := n / nClass
	extra := n % nClass
	out := make([]Image, 0, n)
	for c := 0; c < nClass; c++ {
		take := per
		if c < extra {
			take++
		}
		if take > len(by[c]) {
			take = len(by[c])
		}
		for i := 0; i < take; i++ {
			out = append(out, in[by[c][i]])
		}
	}
	return out
}

func ensure(dir, name string) error {
	path := filepath.Join(dir, name)
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return nil
	}
	resp, err := http.Get(baseURL + name)
	if err != nil {
		return fmt.Errorf("mnist download %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("mnist download %s: HTTP %s", name, resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func readImages(path string) ([][]float32, error) {
	r, err := openGZ(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var magic, n, rows, cols int32
	if err := binary.Read(r, binary.BigEndian, &magic); err != nil {
		return nil, err
	}
	if magic != 0x00000803 {
		return nil, fmt.Errorf("mnist: bad image magic %x", magic)
	}
	for _, v := range []*int32{&n, &rows, &cols} {
		if err := binary.Read(r, binary.BigEndian, v); err != nil {
			return nil, err
		}
	}
	pix := int(rows * cols)
	out := make([][]float32, n)
	buf := make([]byte, pix)
	for i := 0; i < int(n); i++ {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		im := make([]float32, pix)
		for j, b := range buf {
			im[j] = float32(b) / 255
		}
		out[i] = im
	}
	return out, nil
}

func readLabels(path string) ([]byte, error) {
	r, err := openGZ(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var magic, n int32
	if err := binary.Read(r, binary.BigEndian, &magic); err != nil {
		return nil, err
	}
	if magic != 0x00000801 {
		return nil, fmt.Errorf("mnist: bad label magic %x", magic)
	}
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	out := make([]byte, n)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}
	return out, nil
}

func openGZ(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &gzFile{gz: gz, f: f}, nil
}

type gzFile struct {
	gz *gzip.Reader
	f  *os.File
}

func (g *gzFile) Read(p []byte) (int, error) { return g.gz.Read(p) }
func (g *gzFile) Close() error {
	err1 := g.gz.Close()
	err2 := g.f.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
