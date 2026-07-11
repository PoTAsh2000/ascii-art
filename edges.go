package main

import (
	"image"
	"sync"

	"github.com/nfnt/resize"
)

// edgeOptions bundles every tunable of the contour pipeline. Each filter file
// documents the parameters it consumes; the CLI wires these to flags.
type edgeOptions struct {
	enabled bool

	tile int // pixels per character cell per axis (the 8×8 font tile)

	// DoG (see filter_dog.go / filter_gaussian.go)
	sigma        float64
	sigmaScale   float64
	tau          float64
	dogThreshold float64

	// Sobel + tile voting (see filter_sobel.go)
	magThreshold float64 // ignore gradients weaker than this when voting
	minTileFrac  float64 // share of a tile that must be edge pixels to draw a line

	// Debug: dump one PNG per processing step so the pipeline is visible.
	debug    bool
	debugDir string
}

// computeEdgeGrid runs the full contour pipeline and returns a rows×cols grid of
// runes. A zero rune means "no edge here" (keep the fill glyph); a non-zero rune
// is one of the four contour characters | / _ \ for that character cell.
//
// Pipeline (mirrors Acerola's ASCII shader, recreated on the CPU):
//
//	source -> /tile downscale -> grayscale -> DoG -> threshold mask
//	       -> Sobel gradient -> per-tile direction vote -> coherence -> thinning
//
// A PNG snapshot of each intermediate buffer is written when opts.debug is set.
func computeEdgeGrid(src image.Image, cols, rows int, opts edgeOptions) [][]rune {
	if cols < 1 || rows < 1 {
		return nil
	}
	t := opts.tile
	if t < 1 {
		t = 1
	}
	w, h := cols*t, rows*t

	// Downscale the source so each character cell owns a t×t (8×8) block, then
	// run every filter at that resolution before collapsing back to the grid.
	sub := resize.Resize(uint(w), uint(h), src, resize.Lanczos3)
	gray := grayscaleBuffer(sub, w, h)

	d := differenceOfGaussians(gray, w, h, opts)
	mask := thresholdMask(d, opts)
	gx, gy, mag := sobelGradient(d, w, h)

	dirs, strength := voteTiles(gx, gy, mag, mask, cols, rows, t, opts)
	dirs = coherencePass(dirs, cols, rows)
	dirs = thinDirections(dirs, strength, cols, rows)

	if opts.debug {
		dumpDebug(opts.debugDir, cols, rows, w, h, gray, d, mask, mag, dirs)
	}

	grid := make([][]rune, rows)
	for y := 0; y < rows; y++ {
		grid[y] = make([]rune, cols)
		for x := 0; x < cols; x++ {
			if dir := dirs[y*cols+x]; dir != dirNone {
				grid[y][x] = edgeGlyph(dir)
			}
		}
	}
	return grid
}

// voteTiles emulates the GPU compute shader from the video: the image is carved
// into t×t tiles (one per character cell, matching the 8×8 font), and each tile
// independently decides whether it is an edge and, if so, which direction wins.
// Tiles are independent, so we dispatch one goroutine per tile-row — the CPU
// analogue of dispatching thread groups. A tile becomes an edge only if at least
// minTileFrac of its pixels are edge pixels (Acerola's 8/64 rule), then votes
// the magnitude-weighted dominant of the four quantized directions.
func voteTiles(gx, gy, mag, mask []float64, cols, rows, t int, opts edgeOptions) (dirs []int, strength []float64) {
	dirs = make([]int, cols*rows)
	strength = make([]float64, cols*rows)
	w := cols * t
	minCount := opts.minTileFrac * float64(t*t)

	var wg sync.WaitGroup
	for cy := 0; cy < rows; cy++ {
		wg.Add(1)
		go func(cy int) {
			defer wg.Done()
			for cx := 0; cx < cols; cx++ {
				var votes [4]float64
				edgeCount := 0
				for by := 0; by < t; by++ {
					for bx := 0; bx < t; bx++ {
						px := cx*t + bx
						py := cy*t + by
						i := py*w + px
						if mask[i] == 0 || mag[i] <= opts.magThreshold {
							continue
						}
						edgeCount++
						votes[quantizeAngle(gx[i], gy[i])] += mag[i]
					}
				}

				idx := cy*cols + cx
				if float64(edgeCount) < minCount {
					dirs[idx] = dirNone
					continue
				}
				best, bestVotes := 0, votes[0]
				for dir := 1; dir < 4; dir++ {
					if votes[dir] > bestVotes {
						best, bestVotes = dir, votes[dir]
					}
				}
				dirs[idx] = best
				strength[idx] = bestVotes
			}
		}(cy)
	}
	wg.Wait()
	return dirs, strength
}

// coherencePass smooths the per-tile direction grid so lines read as solid and
// continuous: each edge cell adopts the majority direction of itself plus its 8
// neighbors, and isolated edge cells (no edge neighbor) are dropped as noise.
// Reads from a snapshot so updates don't cascade within one pass.
func coherencePass(dirs []int, cols, rows int) []int {
	out := make([]int, len(dirs))
	copy(out, dirs)
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			idx := cy*cols + cx
			if dirs[idx] == dirNone {
				continue
			}
			var counts [4]int
			neighbors := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := cx+dx, cy+dy
					if nx < 0 || nx >= cols || ny < 0 || ny >= rows {
						continue
					}
					dir := dirs[ny*cols+nx]
					if dir == dirNone {
						continue
					}
					counts[dir]++
					if dx != 0 || dy != 0 {
						neighbors++
					}
				}
			}
			if neighbors == 0 {
				out[idx] = dirNone // stray speck
				continue
			}
			best, bestCount := 0, counts[0]
			for dir := 1; dir < 4; dir++ {
				if counts[dir] > bestCount {
					best, bestCount = dir, counts[dir]
				}
			}
			out[idx] = best
		}
	}
	return out
}

// isVerticalish reports whether a direction draws a mostly-upright stroke
// (| / \) that should be thinned along the horizontal axis. Horizontal edges
// (_) are the complement and thin vertically.
func isVerticalish(dir int) bool {
	return dir == dirVert || dir == dirSlash || dir == dirBack
}

// thinDirections is a non-maximum-suppression pass that collapses a thick band
// of edge cells to a single-cell-wide line. Perpendicular to each line's
// orientation, only the strongest cell in a run of the same direction survives.
// Neighbors of a different direction (or none) count as strength 0, so distinct
// lines never suppress each other. Ties keep exactly one cell per run via the
// asymmetric > / >= test. Reads from snapshots so suppression doesn't cascade.
func thinDirections(dirs []int, strength []float64, cols, rows int) []int {
	out := make([]int, len(dirs))
	copy(out, dirs)

	sameStrength := func(nx, ny, dir int) float64 {
		if nx < 0 || nx >= cols || ny < 0 || ny >= rows {
			return 0
		}
		nIdx := ny*cols + nx
		if dirs[nIdx] != dir {
			return 0
		}
		return strength[nIdx]
	}

	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			idx := cy*cols + cx
			dir := dirs[idx]
			if dir == dirNone {
				continue
			}
			st := strength[idx]

			var a, b float64
			if isVerticalish(dir) {
				a = sameStrength(cx-1, cy, dir) // left
				b = sameStrength(cx+1, cy, dir) // right
			} else {
				a = sameStrength(cx, cy-1, dir) // up
				b = sameStrength(cx, cy+1, dir) // down
			}
			if !(st > a && st >= b) {
				out[idx] = dirNone
			}
		}
	}
	return out
}

// edgeGlyph maps a direction code to one of the four contour runes.
func edgeGlyph(dir int) rune {
	switch dir {
	case dirVert:
		return '|'
	case dirSlash:
		return '/'
	case dirHoriz:
		return '_'
	case dirBack:
		return '\\'
	default:
		return 0
	}
}
