package main

import (
	"image"
	"math"
	"strings"
)

// asciiRamp maps brightness (dark→light) to characters. Luminance is a value in
// [0,1], and mapping it to the matching character is the basis of ASCII art.
// Ten glyphs — one per tenth of the brightness range — kept deliberately short
// so the art stays chunky/cartoonish rather than a smooth photographic gradient.
var asciiRamp = []rune{' ', '.', 'i', 'c', 'o', 'L', 'P', 'O', '#', '█'}

// gamma > 1 brightens midtones before ramp mapping, so shadow detail isn't
// crushed into the darkest few characters. Tune to taste; 0 and 1 stay fixed.
const gamma = 1.8

// render builds the full ASCII frame for an image and returns it as one string.
// If edgeGrid is non-nil, a non-zero rune at edgeGrid[y][x] overrides the fill
// glyph for that cell, drawing a contour line on top of the brightness ramp.
// edgeGrid must be rows×cols matching the image; pass nil for fill-only output.
func render(img image.Image, edgeGrid [][]rune) string {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Build the whole frame, then write once. One syscall instead of thousands
	// and no flicker — this is the fast path video playback will reuse.
	var sb strings.Builder
	sb.Grow(width*height*3 + height) // ramp runes are up to 3 bytes (█)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if edgeGrid != nil && y < len(edgeGrid) && x < len(edgeGrid[y]) && edgeGrid[y][x] != 0 {
				sb.WriteRune(edgeGrid[y][x])
				continue
			}

			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			luminance := math.Pow(luminanceRGBA(r, g, b), 1.0/gamma)

			idx := int(luminance*float64(len(asciiRamp)-1) + 0.5)
			sb.WriteRune(asciiRamp[idx])
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
