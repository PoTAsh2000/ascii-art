package main

import "image"

// Rec.709 luminance weights. Luminance is the perceived brightness of a pixel
// and is the whole basis of the ASCII fill: brightness picks a ramp character.
const (
	redWeight   = 0.2126
	greenWeight = 0.7152
	blueWeight  = 0.0722
)

// luminanceRGBA converts a 16-bit-per-channel RGBA sample (as returned by
// color.Color.RGBA) to a clamped 0..1 Rec.709 luminance. Shared by the fill
// renderer and every filter that works on a grayscale buffer.
func luminanceRGBA(r, g, b uint32) float64 {
	nr := float64(r>>8) / 255.0
	ng := float64(g>>8) / 255.0
	nb := float64(b>>8) / 255.0

	l := redWeight*nr + greenWeight*ng + blueWeight*nb
	if l < 0 {
		l = 0
	}
	if l > 1 {
		l = 1
	}
	return l
}

// grayscaleBuffer flattens an image into a row-major 0..1 luminance slice. This
// is filter step 1 for the edge pipeline (and what every later filter reads).
func grayscaleBuffer(img image.Image, w, h int) []float64 {
	b := img.Bounds()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			out[y*w+x] = luminanceRGBA(r, g, bl)
		}
	}
	return out
}
