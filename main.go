package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"

	"github.com/nfnt/resize"
	"golang.org/x/term"
)

const (
	redWeight   = 0.2126
	greenWeight = 0.7152
	blueWeight  = 0.0722
)

func main() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()

	terminal_width, terminal_height := get_terminal_size()

	var resized_image_data image.Image = resize_image("test_images/porsche-911-1.jpg", terminal_width, terminal_height)
	process_image(resized_image_data)
}

func process_image(image image.Image) {
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

	bounds := image.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := image.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()

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

			character := ascii_map[luminance]
			fmt.Printf("%s", character)
		}
	}
}

func resize_image(filename string, terminal_width int, terminal_height int) (resized_image image.Image) {
	image_file, err := os.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("an exception occurred while trying to open file. %v", err))
	}
	defer image_file.Close()

	image, _, err := image.Decode(image_file)
	if err != nil {
		panic(fmt.Sprintf("an exception occurred while trying decode the image. %v", err))
	}

	return resize.Resize(uint(terminal_width), uint(terminal_height), image, resize.Lanczos3)
}

func get_terminal_size() (width int, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))

	if err != nil {
		panic(fmt.Sprintf("An exception occurred while trying to get terminal size. %v", err))
	}

	terminal_width := width
	terminal_height := height

	return terminal_width, terminal_height
}
