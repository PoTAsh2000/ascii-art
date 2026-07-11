package main

import "math"

// ---- Gaussian blur --------------------------------------------------------
// Tunable here: kernelRadiusSigmas sets how far the kernel reaches (in sigmas).
// The blur itself is separable (two 1-D passes) with edge clamping, so a flat
// field is preserved exactly. This is the building block the DoG filter stacks.

// kernelRadiusSigmas controls kernel width: radius = ceil(kernelRadiusSigmas*sigma).
// 3 captures ~99.7% of the Gaussian; lower is faster/coarser.
const kernelRadiusSigmas = 3.0

// gaussianKernel builds a normalized 1-D Gaussian of radius ceil(3*sigma).
func gaussianKernel(sigma float64) []float64 {
	if sigma <= 0 {
		return []float64{1}
	}
	radius := int(math.Ceil(kernelRadiusSigmas * sigma))
	k := make([]float64, 2*radius+1)
	sum := 0.0
	twoSigma2 := 2 * sigma * sigma
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / twoSigma2)
		k[i+radius] = v
		sum += v
	}
	for i := range k {
		k[i] /= sum
	}
	return k
}

// gaussianBlur applies a separable Gaussian blur with edge clamping. A constant
// field is preserved exactly (the kernel is normalized).
func gaussianBlur(src []float64, w, h int, sigma float64) []float64 {
	k := gaussianKernel(sigma)
	radius := len(k) / 2

	tmp := make([]float64, w*h)
	for y := 0; y < h; y++ { // horizontal pass
		for x := 0; x < w; x++ {
			sum := 0.0
			for i := -radius; i <= radius; i++ {
				sx := clampInt(x+i, 0, w-1)
				sum += src[y*w+sx] * k[i+radius]
			}
			tmp[y*w+x] = sum
		}
	}
	out := make([]float64, w*h)
	for y := 0; y < h; y++ { // vertical pass
		for x := 0; x < w; x++ {
			sum := 0.0
			for i := -radius; i <= radius; i++ {
				sy := clampInt(y+i, 0, h-1)
				sum += tmp[sy*w+x] * k[i+radius]
			}
			out[y*w+x] = sum
		}
	}
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
