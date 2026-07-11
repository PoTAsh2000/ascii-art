package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// dumpDebug writes one PNG per pipeline stage into dir so the effect of every
// filter can be inspected visually. Failures are reported but never abort the
// render — debug output is a convenience, not a requirement.
func dumpDebug(dir string, cols, rows, w, h int, gray, dog, mask, mag []float64, dirs []int) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "debug: cannot create dir:", err)
		return
	}

	writeGray01(filepath.Join(dir, "01_grayscale.png"), gray, w, h)
	writeGrayNorm(filepath.Join(dir, "02_dog.png"), dog, w, h)
	writeGray01(filepath.Join(dir, "03_threshold_mask.png"), mask, w, h)
	writeGrayNorm(filepath.Join(dir, "04_sobel_magnitude.png"), mag, w, h)
	writeEdges(filepath.Join(dir, "05_edges.png"), dirs, cols, rows, w/cols)

	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	fmt.Fprintln(os.Stderr, "debug: wrote filter step images to", abs)
}

// writeGray01 renders a 0..1 buffer straight to an 8-bit grayscale PNG.
func writeGray01(path string, buf []float64, w, h int) {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i, v := range buf {
		img.Pix[i] = clamp8(v * 255)
	}
	savePNG(path, img)
}

// writeGrayNorm normalizes a buffer to its own min..max before writing, so
// signed fields (DoG) and unbounded ones (Sobel magnitude) stay visible.
func writeGrayNorm(path string, buf []float64, w, h int) {
	if len(buf) == 0 {
		return
	}
	min, max := buf[0], buf[0]
	for _, v := range buf {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	if span == 0 {
		span = 1
	}
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i, v := range buf {
		img.Pix[i] = clamp8((v - min) / span * 255)
	}
	savePNG(path, img)
}

// writeEdges paints the final per-cell direction grid, one solid tile-sized
// block per character cell, colored by direction so the contour map is legible.
func writeEdges(path string, dirs []int, cols, rows, tile int) {
	if tile < 1 {
		tile = 1
	}
	colors := map[int]color.RGBA{
		dirVert:  {255, 80, 80, 255},  // | red
		dirSlash: {80, 220, 80, 255},  // / green
		dirHoriz: {80, 140, 255, 255}, // _ blue
		dirBack:  {240, 220, 60, 255}, // \ yellow
	}
	img := image.NewRGBA(image.Rect(0, 0, cols*tile, rows*tile))
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			c := color.RGBA{0, 0, 0, 255}
			if dir := dirs[cy*cols+cx]; dir != dirNone {
				c = colors[dir]
			}
			for by := 0; by < tile; by++ {
				for bx := 0; bx < tile; bx++ {
					img.Set(cx*tile+bx, cy*tile+by, c)
				}
			}
		}
	}
	savePNG(path, img)
}

func savePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "debug: cannot write", path, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "debug: encode", path, err)
	}
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
