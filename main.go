package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/nfnt/resize"
	"golang.org/x/term"
)

// cellAspect is the width:height ratio of a terminal character cell (~0.5,
// cells are roughly twice as tall as wide). Used to undo vertical squashing.
const cellAspect = 0.5

// Edge-detection defaults. Contours use the four characters | / _ \ drawn on top
// of the brightness ramp. These seed the matching CLI flags; each filter file
// documents the parameter it consumes.
const (
	// defaultTile is pixels-per-character-cell per axis. 8 matches an 8×8 font
	// tile (the compute-shader group size in the reference), so each cell owns a
	// clean 8×8 block of the /8-downscaled image.
	defaultTile = 8
	// DoG preprocess (filter_dog.go): two blurs at sigma and sigma*scale, their
	// weighted difference thresholded into a binary edge mask.
	defaultSigma        = 2.5
	defaultSigmaScale   = 1.6
	defaultTau          = 1.0
	defaultDogThreshold = 0.015
	// Sobel voting (filter_sobel.go / edges.go): ignore near-zero gradients, and
	// require this fraction of a tile to be edge pixels before drawing a line
	// (0.125 = the reference's 8-of-64 rule).
	defaultMagThreshold = 0.0
	defaultMinTileFrac  = 0.125
)

// toggle is a flag.Value accepting 1/0, true/false, on/off (case-insensitive).
// It is a boolean flag, so a bare --edges means "on" and values use --edges=off.
type toggle bool

func (t *toggle) String() string {
	if t != nil && bool(*t) {
		return "on"
	}
	return "off"
}

func (t *toggle) Set(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "on", "yes":
		*t = true
	case "0", "false", "off", "no":
		*t = false
	default:
		return fmt.Errorf("invalid value %q (use 1/0, true/false, on/off)", s)
	}
	return nil
}

func (t *toggle) IsBoolFlag() bool { return true }

func main() {
	edges := toggle(true) // on by default
	debug := toggle(true) // dump per-filter images by default
	opts := edgeOptions{tile: defaultTile}
	flag.Var(&edges, "edges", "draw edge/contour lines (1/0, true/false, on/off)")
	flag.Var(&debug, "debug", "write a PNG per filter step (1/0, true/false, on/off)")
	flag.StringVar(&opts.debugDir, "debug-dir", "debug", "directory for filter-step images")
	flag.Float64Var(&opts.sigma, "sigma", defaultSigma, "DoG base Gaussian sigma")
	flag.Float64Var(&opts.sigmaScale, "sigma-scale", defaultSigmaScale, "DoG second-Gaussian sigma multiplier")
	flag.Float64Var(&opts.tau, "tau", defaultTau, "DoG difference weight")
	flag.Float64Var(&opts.dogThreshold, "dog-threshold", defaultDogThreshold, "DoG magnitude to count a pixel as an edge")
	flag.Float64Var(&opts.magThreshold, "mag-threshold", defaultMagThreshold, "min Sobel gradient magnitude to vote")
	flag.Float64Var(&opts.minTileFrac, "tile-fill", defaultMinTileFrac, "min edge-pixel fraction of a tile to draw a contour")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ascii-art [flags] <image-path>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	filename := flag.Arg(0)
	opts.enabled = bool(edges)
	opts.debug = bool(debug)

	// Erase screen + move cursor home (once). For future video, redraw each
	// frame with just "\x1b[H" to overwrite in place without flicker.
	fmt.Print("\x1b[2J\x1b[H")

	terminal_width, terminal_height, err := get_terminal_size()
	if err != nil {
		fail(err)
	}

	decoded, err := decodeImage(filename)
	if err != nil {
		fail(err)
	}

	bounds := decoded.Bounds()
	cols, rows := fitDimensions(bounds.Dx(), bounds.Dy(), terminal_width, terminal_height)
	charImg := resize.Resize(uint(cols), uint(rows), decoded, resize.Lanczos3)

	var edgeGrid [][]rune
	if opts.enabled {
		edgeGrid = computeEdgeGrid(decoded, cols, rows, opts)
	}

	os.Stdout.WriteString(render(charImg, edgeGrid))

	// Keep the art on screen: wait for any key before exiting so the shell
	// prompt doesn't immediately clobber the image. No prompt is printed on
	// purpose — the pause is silent.
	wait_for_key()
}

// wait_for_key blocks until a single key is pressed. It puts stdin in raw mode
// so one keystroke returns immediately (no Enter needed) and prints nothing.
// If raw mode isn't available (e.g. stdin isn't a terminal) it falls back to
// reading a line so the program still doesn't exit instantly.
func wait_for_key() {
	fd := int(os.Stdin.Fd())
	old_state, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fscanln(os.Stdin)
		return
	}
	defer term.Restore(fd, old_state)

	var buf [1]byte
	os.Stdin.Read(buf[:])
}

// fail prints a uniform error message to stderr and exits non-zero.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// decodeImage opens and decodes an image file (gif/jpeg/png via the blank
// imports above). Split out so both the character-grid resize and the
// higher-resolution edge buffer can work from a single decode.
func decodeImage(filename string) (image.Image, error) {
	image_file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", filename, err)
	}
	defer image_file.Close()

	decoded_image, _, err := image.Decode(image_file)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", filename, err)
	}
	return decoded_image, nil
}

func resize_image(filename string, terminal_width int, terminal_height int) (image.Image, error) {
	decoded_image, err := decodeImage(filename)
	if err != nil {
		return nil, err
	}

	bounds := decoded_image.Bounds()
	cols, rows := fitDimensions(bounds.Dx(), bounds.Dy(), terminal_width, terminal_height)

	return resize.Resize(uint(cols), uint(rows), decoded_image, resize.Lanczos3), nil
}

// fitDimensions returns the largest (cols, rows) that fits within maxCols×maxRows
// while preserving the source aspect ratio and correcting for the terminal cell
// shape (cellAspect). Output is contain-style: no padding.
func fitDimensions(srcW, srcH, maxCols, maxRows int) (cols, rows int) {
	if srcW <= 0 || srcH <= 0 {
		return 1, 1
	}

	// Rows needed for a given column count so the image keeps its aspect ratio
	// in character space: rows = cols * cellAspect * (srcH/srcW).
	cols = maxCols
	rows = int(float64(cols) * cellAspect * float64(srcH) / float64(srcW))

	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows) * float64(srcW) / (cellAspect * float64(srcH)))
	}

	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

func get_terminal_size() (width int, height int, err error) {
	width, height, err = term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0, fmt.Errorf("get terminal size: %w", err)
	}
	return width, height, nil
}
