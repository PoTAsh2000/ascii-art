package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"
	"github.com/nfnt/resize"
	"golang.org/x/term"
)

const (
	redWeight   = 0.2126
	greenWeight = 0.7152
	blueWeight  = 0.0722
)

func main() {
	if len(os.Args) < 2 {
		fail(fmt.Errorf("usage: ascii-art <image-path>"))
	}
	filename := os.Args[1]

	// Erase screen + move cursor home (once). For future video, redraw each
	// frame with just "\x1b[H" to overwrite in place without flicker.
	fmt.Print("\x1b[2J\x1b[H")

	terminal_width, terminal_height, err := get_terminal_size()
	if err != nil {
		fail(err)
	}

	resized_image_data, err := resize_image(filename, terminal_width, terminal_height)
	if err != nil {
		fail(err)
	}

	process_image(resized_image_data)
}

// fail prints a uniform error message to stderr and exits non-zero.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func process_image(img image.Image) {
	os.Stdout.WriteString(render(img))
}

// render builds the full ASCII frame for an image and returns it as one string.
// Kept separate from process_image so it can be unit-tested and so video
// playback can reuse it to build a frame before writing.
func render(img image.Image) string {
	ascii_map := map[float64]string{
		0:   " ",
		0.1: ".",
		0.2: "i",
		0.3: "c",
		0.4: "o",
		0.5: "L",
		0.6: "P",
		0.7: "O",
		0.8: "?",
		0.9: "#",
		1:   "█",
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Build the whole frame, then write once. One syscall instead of thousands
	// and no flicker — this is the fast path video playback will reuse.
	var sb strings.Builder
	sb.Grow(width*height + height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()

			normal_r := float64(r>>8) / 255.0
			normal_g := float64(g>>8) / 255.0
			normal_b := float64(b>>8) / 255.0

			luminance := redWeight*normal_r + greenWeight*normal_g + blueWeight*normal_b
			luminance = math.Floor(luminance*10) / 10.0

			if luminance < 0 {
				luminance = 0
			}
			if luminance > 1 {
				luminance = 1
			}

			sb.WriteString(ascii_map[luminance])
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func resize_image(filename string, terminal_width int, terminal_height int) (image.Image, error) {
	image_file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", filename, err)
	}
	defer image_file.Close()

	decoded_image, _, err := image.Decode(image_file)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", filename, err)
	}

	return resize.Resize(uint(terminal_width), uint(terminal_height), decoded_image, resize.Lanczos3), nil
}

func get_terminal_size() (width int, height int, err error) {
	width, height, err = term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0, fmt.Errorf("get terminal size: %w", err)
	}
	return width, height, nil
}
