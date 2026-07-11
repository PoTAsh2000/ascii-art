package main

// ---- Difference of Gaussians (DoG) ----------------------------------------
// Runs *before* Sobel. Two Gaussian blurs at sigma and sigma*sigmaScale are
// subtracted (weighted by tau) to extract high-frequency detail — the edges.
// A threshold turns that continuous response into a clean binary edge mask, so
// Sobel then measures direction only where a real edge lives.
//
// Tune the look with edgeOptions.sigma / .sigmaScale / .tau / .dogThreshold.

// differenceOfGaussians returns the raw DoG field d = blur(sigma) - tau*blur(k*sigma).
// It is ~0 in flat regions and swings away from 0 across edges.
func differenceOfGaussians(gray []float64, w, h int, opts edgeOptions) []float64 {
	g1 := gaussianBlur(gray, w, h, opts.sigma)
	g2 := gaussianBlur(gray, w, h, opts.sigma*opts.sigmaScale)

	d := make([]float64, w*h)
	for i := range d {
		d[i] = g1[i] - opts.tau*g2[i]
	}
	return d
}

// thresholdMask turns the DoG field into a binary edge mask: 1 where the DoG
// magnitude clears dogThreshold, else 0. This is the "make it an actual edge
// detector" step — everything below the threshold is treated as flat.
func thresholdMask(d []float64, opts edgeOptions) []float64 {
	mask := make([]float64, len(d))
	for i, v := range d {
		if v >= opts.dogThreshold || v <= -opts.dogThreshold {
			mask[i] = 1
		}
	}
	return mask
}
