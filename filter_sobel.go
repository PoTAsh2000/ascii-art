package main

import "math"

// ---- Sobel gradient + direction quantization ------------------------------
// Sobel is chosen because a single pass yields both edge strength (magnitude)
// AND the edge's orientation (the gradient angle) — exactly what we need to pick
// a directional glyph. It runs on the DoG field, so it only reacts to real
// edges. The continuous gradient angle is then quantized down to four values,
// one per available contour character.

// Edge direction codes. The gradient points across the edge; folding its angle
// into quarters of pi lands each code on the right glyph (see edgeGlyph):
//
//	theta ~ 0    -> vertical edge    |   (intensity changes horizontally)
//	theta ~ pi/4 -> forward slash    /
//	theta ~ pi/2 -> horizontal edge  _   (intensity changes vertically)
//	theta ~ 3pi/4-> back slash       \
const (
	dirNone  = -1
	dirVert  = 0 // |
	dirSlash = 1 // /
	dirHoriz = 2 // _
	dirBack  = 3 // \
)

// Standard Sobel kernels.
//
//	Gx = [-1 0 1 ; -2 0 2 ; -1 0 1]   Gy = [-1 -2 -1 ; 0 0 0 ; 1 2 1]
var (
	sobelX = [3][3]float64{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY = [3][3]float64{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}
)

// sobelGradient convolves the field with the Sobel kernels, returning per-pixel
// gx, gy and magnitude sqrt(gx²+gy²). Borders are clamped.
func sobelGradient(field []float64, w, h int) (gx, gy, mag []float64) {
	gx = make([]float64, w*h)
	gy = make([]float64, w*h)
	mag = make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sx, sy float64
			for j := -1; j <= 1; j++ {
				for i := -1; i <= 1; i++ {
					px := clampInt(x+i, 0, w-1)
					py := clampInt(y+j, 0, h-1)
					v := field[py*w+px]
					sx += v * sobelX[j+1][i+1]
					sy += v * sobelY[j+1][i+1]
				}
			}
			gx[y*w+x] = sx
			gy[y*w+x] = sy
			mag[y*w+x] = math.Hypot(sx, sy)
		}
	}
	return gx, gy, mag
}

// quantizeAngle downscales a continuous gradient angle to one of four direction
// codes: fold theta into [0,pi), then round to the nearest multiple of pi/4
// (4 wraps back to 0). Four buckets = four contour characters.
func quantizeAngle(gx, gy float64) int {
	t := math.Atan2(gy, gx)
	if t < 0 {
		t += math.Pi // fold to [0, pi)
	}
	return int(t/(math.Pi/4)+0.5) % 4
}
