package main

import (
	"image"
	"image/color"
	"testing"
)

// defaultEdgeOpts returns the CLI-default edge options, enabled, debug off.
func defaultEdgeOpts() edgeOptions {
	return edgeOptions{
		enabled:      true,
		tile:         defaultTile,
		sigma:        defaultSigma,
		sigmaScale:   defaultSigmaScale,
		tau:          defaultTau,
		dogThreshold: defaultDogThreshold,
		magThreshold: defaultMagThreshold,
		minTileFrac:  defaultMinTileFrac,
	}
}

// halfImage builds a w×h image split into two solid regions so a single strong
// edge runs along the boundary. orient picks the split axis/diagonal.
func halfImage(w, h int, orient string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dark bool
			switch orient {
			case "vertical": // left dark, right light -> vertical edge
				dark = x < w/2
			case "horizontal": // top dark, bottom light -> horizontal edge
				dark = y < h/2
			case "slash": // dark below the / diagonal (bottom-left) -> '/'
				dark = (x + y) < (w+h)/2
			case "back": // dark below the \ diagonal (bottom-right) -> '\'
				dark = (x - y) > 0
			}
			c := color.RGBA{255, 255, 255, 255}
			if dark {
				c = color.RGBA{0, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// countGlyph counts occurrences of r across an edge grid.
func countGlyph(grid [][]rune, r rune) int {
	n := 0
	for _, row := range grid {
		for _, g := range row {
			if g == r {
				n++
			}
		}
	}
	return n
}

func TestEdgeGridVertical(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "vertical"), 20, 20, defaultEdgeOpts())
	if countGlyph(grid, '|') == 0 {
		t.Fatal("vertical edge should produce '|' glyphs, got none")
	}
}

func TestEdgeGridHorizontal(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "horizontal"), 20, 20, defaultEdgeOpts())
	if countGlyph(grid, '_') == 0 {
		t.Fatal("horizontal edge should produce '_' glyphs, got none")
	}
}

func TestEdgeGridSlash(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "slash"), 20, 20, defaultEdgeOpts())
	if countGlyph(grid, '/') == 0 {
		t.Fatal("'/' diagonal should produce '/' glyphs, got none")
	}
}

func TestEdgeGridBack(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "back"), 20, 20, defaultEdgeOpts())
	if countGlyph(grid, '\\') == 0 {
		t.Fatal("'\\' diagonal should produce '\\' glyphs, got none")
	}
}

// TestEdgeGridOnlyFourGlyphs makes sure no stray '-' (or anything else) leaks in.
func TestEdgeGridOnlyFourGlyphs(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "horizontal"), 20, 20, defaultEdgeOpts())
	allowed := map[rune]bool{0: true, '|': true, '/': true, '_': true, '\\': true}
	for _, row := range grid {
		for _, g := range row {
			if !allowed[g] {
				t.Fatalf("unexpected glyph %q in edge grid", g)
			}
		}
	}
}

func TestEdgeGridFlatImageHasNoEdges(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	grid := computeEdgeGrid(img, 20, 20, defaultEdgeOpts())
	for y, row := range grid {
		for x, g := range row {
			if g != 0 {
				t.Fatalf("flat image should have no edges, found %q at (%d,%d)", g, x, y)
			}
		}
	}
}

// TestGaussianBlurPreservesConstant checks the normalized kernel leaves a flat
// field untouched (no darkening/brightening at borders thanks to clamping).
func TestGaussianBlurPreservesConstant(t *testing.T) {
	const w, h = 16, 16
	src := make([]float64, w*h)
	for i := range src {
		src[i] = 0.5
	}
	out := gaussianBlur(src, w, h, 1.5)
	for i, v := range out {
		if diff := v - 0.5; diff > 1e-9 || diff < -1e-9 {
			t.Fatalf("blur changed constant field at %d: %v", i, v)
		}
	}
}

// TestSobelGradientPeaksAtEdge confirms the gradient magnitude is large at a
// step edge and ~0 in the flat interior.
func TestSobelGradientPeaksAtEdge(t *testing.T) {
	const w, h = 9, 9
	e := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x >= w/2 {
				e[y*w+x] = 1
			}
		}
	}
	_, _, mag := sobelGradient(e, w, h)

	edgeMag := mag[(h/2)*w+w/2] // on the step
	flatMag := mag[(h/2)*w+0]   // far-left interior, uniform 0
	if edgeMag <= flatMag {
		t.Fatalf("expected edge magnitude > flat: edge=%v flat=%v", edgeMag, flatMag)
	}
	if flatMag > 1e-9 {
		t.Fatalf("flat region should have ~0 gradient, got %v", flatMag)
	}
}

// TestThinVerticalBandSingleWidth checks a wide vertical edge collapses so each
// grid row has at most one '|'.
func TestThinVerticalBandSingleWidth(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "vertical"), 20, 20, defaultEdgeOpts())
	if countGlyph(grid, '|') == 0 {
		t.Fatal("expected some '|' glyphs")
	}
	for y, row := range grid {
		n := 0
		for _, g := range row {
			if g == '|' {
				n++
			}
		}
		if n > 1 {
			t.Fatalf("row %d has %d '|' glyphs, want <=1 after thinning", y, n)
		}
	}
}

// TestThinHorizontalBandSingleHeight checks a wide horizontal edge collapses so
// each grid column has at most one '_'.
func TestThinHorizontalBandSingleHeight(t *testing.T) {
	grid := computeEdgeGrid(halfImage(120, 120, "horizontal"), 20, 20, defaultEdgeOpts())
	if countGlyph(grid, '_') == 0 {
		t.Fatal("expected some '_' glyphs")
	}
	rows := len(grid)
	cols := len(grid[0])
	for x := 0; x < cols; x++ {
		n := 0
		for y := 0; y < rows; y++ {
			if grid[y][x] == '_' {
				n++
			}
		}
		if n > 1 {
			t.Fatalf("col %d has %d '_' glyphs, want <=1 after thinning", x, n)
		}
	}
}

// TestThinDirectionsRuns exercises thinDirections directly: a horizontal run of
// vertical-ish cells keeps only the strongest; a vertical run of horizontal
// cells keeps only the strongest per column.
func TestThinDirectionsRuns(t *testing.T) {
	dirs := []int{dirVert, dirVert, dirVert, dirVert}
	strength := []float64{1, 2, 3, 4}
	got := thinDirections(dirs, strength, 4, 1)
	want := []int{dirNone, dirNone, dirNone, dirVert}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("horizontal run: got %v, want %v", got, want)
		}
	}

	dirs = []int{dirHoriz, dirHoriz, dirHoriz, dirHoriz}
	strength = []float64{1, 5, 2, 1}
	got = thinDirections(dirs, strength, 1, 4)
	want = []int{dirNone, dirHoriz, dirNone, dirNone}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("vertical run: got %v, want %v", got, want)
		}
	}
}

// TestQuantizeAngle pins the four axis/diagonal gradient directions to codes.
func TestQuantizeAngle(t *testing.T) {
	cases := []struct {
		gx, gy float64
		want   int
	}{
		{1, 0, dirVert},  // horizontal gradient -> vertical edge
		{0, 1, dirHoriz}, // vertical gradient -> horizontal edge
		{1, 1, dirSlash}, // 45° gradient -> '/'
		{1, -1, dirBack}, // -45° gradient -> '\'
	}
	for _, c := range cases {
		if got := quantizeAngle(c.gx, c.gy); got != c.want {
			t.Fatalf("quantizeAngle(%v,%v) = %d, want %d", c.gx, c.gy, got, c.want)
		}
	}
}
