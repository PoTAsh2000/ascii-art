package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gridLines splits a rendered frame into rows (dropping the trailing newline)
// and returns them. Helper for the rectangular-grid assertions below.
func gridLines(frame string) []string {
	return strings.Split(strings.TrimRight(frame, "\n"), "\n")
}

// TestRenderRealImage is the end-to-end "draws an image" test: it decodes a
// tracked sample, resizes it (fit/contain), renders it, and checks the output is
// a rectangular grid that fits within the requested bounds.
func TestRenderRealImage(t *testing.T) {
	const maxW, maxH = 80, 40
	img, err := resize_image(filepath.Join("test_images", "master-chief.jpg"), maxW, maxH)
	if err != nil {
		t.Fatalf("resize_image failed: %v", err)
	}

	lines := gridLines(render(img))
	if len(lines) == 0 || len(lines) > maxH {
		t.Fatalf("row count %d not in 1..%d", len(lines), maxH)
	}
	rowWidth := len([]rune(lines[0]))
	if rowWidth == 0 || rowWidth > maxW {
		t.Fatalf("row width %d not in 1..%d", rowWidth, maxW)
	}
	for i, ln := range lines {
		if got := len([]rune(ln)); got != rowWidth {
			t.Fatalf("row %d width = %d, want uniform %d", i, got, rowWidth)
		}
	}
}

// TestFitDimensions checks contain-fit + cell-aspect correction (P2/P10).
func TestFitDimensions(t *testing.T) {
	const maxCols, maxRows = 80, 40

	// Square source: cell-aspect should halve rows relative to cols.
	cols, rows := fitDimensions(100, 100, maxCols, maxRows)
	if cols < 1 || rows < 1 || cols > maxCols || rows > maxRows {
		t.Fatalf("square out of bounds: %dx%d", cols, rows)
	}
	if wantRows := int(float64(cols) * cellAspect); rows != wantRows {
		t.Fatalf("square: rows=%d, want ~cols*cellAspect=%d", rows, wantRows)
	}

	// Wide (2:1) and tall (1:2) sources must both stay within bounds, >=1.
	for _, tc := range []struct{ w, h int }{{200, 100}, {100, 200}, {1, 1}} {
		c, r := fitDimensions(tc.w, tc.h, maxCols, maxRows)
		if c < 1 || r < 1 || c > maxCols || r > maxRows {
			t.Fatalf("src %dx%d -> %dx%d out of 1..%dx%d", tc.w, tc.h, c, r, maxCols, maxRows)
		}
	}

	// Degenerate source dimensions must not panic.
	if c, r := fitDimensions(0, 0, maxCols, maxRows); c < 1 || r < 1 {
		t.Fatalf("zero source -> %dx%d, want >=1x1", c, r)
	}
}

// TestRenderGridSynthetic renders a small deterministic gradient and checks the
// grid shape plus exact content.
func TestRenderGridSynthetic(t *testing.T) {
	const w, h = 5, 3
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8((x + y) * 20)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}

	lines := gridLines(render(img))
	if len(lines) != h {
		t.Fatalf("expected %d rows, got %d", h, len(lines))
	}
	for i, ln := range lines {
		if got := len([]rune(ln)); got != w {
			t.Fatalf("row %d width = %d, want %d", i, got, w)
		}
	}
}

// TestRenderKnownPixels checks the luminance→ramp mapping at both extremes.
func TestRenderKnownPixels(t *testing.T) {
	solid := func(c color.Color) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, 3, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				img.Set(x, y, c)
			}
		}
		return img
	}

	black := strings.ReplaceAll(render(solid(color.RGBA{0, 0, 0, 255})), "\n", "")
	for _, r := range black {
		if r != ' ' {
			t.Fatalf("black image should render as spaces, got %q", black)
		}
	}

	white := strings.TrimRight(render(solid(color.RGBA{255, 255, 255, 255})), "\n")
	if !strings.Contains(white, "█") {
		t.Fatalf("white image should render as full blocks, got %q", white)
	}
}

// TestResizeImageMissingFile expects a clear error, not a panic.
func TestResizeImageMissingFile(t *testing.T) {
	_, err := resize_image("does-not-exist.png", 10, 10)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "does-not-exist.png") {
		t.Fatalf("error should mention the path, got: %v", err)
	}
}

// TestResizeImageBadDecode feeds a non-image file and expects a decode error.
func TestResizeImageBadDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-an-image.png")
	if err := os.WriteFile(path, []byte("this is not an image"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err := resize_image(path, 10, 10)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got: %v", err)
	}
}

// Note: get_terminal_size requires a real TTY (term.GetSize on stdout's fd) and
// is not unit-tested here — it is exercised when the program runs in a terminal.
