# ASCII ART

## How to use
* You need to have [GO](https://go.dev/) installed to be able to run this script
* Install the **term** and **resize** packages:
    * go get golang.org/x/term
    * go get github.com/nfnt/resize
* ▶️ run the program, passing the image path as the only argument: ```go run . test_images/master-chief.jpg```

## Edge lines
Contour lines are drawn on top of the brightness ramp using four characters — `| / _ \` — and
are **on by default**. The pipeline recreates the compute-shader approach from
[Acerola's video](https://www.youtube.com/watch?v=gg40RWiaHRY) / [AcerolaFX](https://github.com/GarrettGunnell/AcerolaFX)
on the CPU:

1. **Grayscale** — Rec.709 luminance (`luminance.go`)
2. **/8 downscale** — each character cell owns an 8×8 pixel tile (the font/compute-group size)
3. **Difference of Gaussians** — two blurs subtracted to pull out high-frequency detail (`filter_dog.go`, `filter_gaussian.go`)
4. **Threshold** — turns the DoG response into a clean binary edge mask
5. **Sobel** — one pass gives edge strength *and* direction (`filter_sobel.go`)
6. **Quantize** — the gradient angle is reduced to 4 values, one per contour character
7. **Tile vote** — each 8×8 tile becomes an edge only if enough of it is edge pixels, then votes a direction (`edges.go`); tiles run in parallel goroutines (the CPU analogue of dispatching thread groups)
8. **Coherence + thinning** — smooth lines solid, then non-max thin to single-cell width

Every filter lives in its own `filter_*.go` file so parameters are easy to find and tweak.

* Toggle: `--edges=off` (also accepts `0`/`false`, and `1`/`true`/`on`)
* Tune: `--sigma`, `--sigma-scale`, `--tau`, `--dog-threshold` (raise to drop texture noise), `--mag-threshold`, `--tile-fill`
* **Debug images**: on by default, one PNG per filter step is written to `./debug/` so you can watch what each stage does. Turn off with `--debug=off` or point elsewhere with `--debug-dir`.

Example: ```go run . --dog-threshold 0.05 --edges=on image.jpg```

### Tuning guide
The filters are analog, but the output is quantized four times after them (binary threshold →
4-direction bucket → 8×8 tile vote → single-width thinning). Small tweaks often snap back to the
same glyphs, so **judge changes on `debug/02_dog.png` and `debug/03_threshold_mask.png`** (before the
quantization) rather than the ASCII alone.

| Flag | Useful range (sweet spot) | Changes these images the most | If you move it, also move |
|------|---------------------------|-------------------------------|---------------------------|
| `--sigma` | 0.8 – 4.0 (1.5 – 2.5) | busy / fine-texture photos (big sigma erases fine detail); flat images barely react | up ⇒ lower `--dog-threshold` (bigger blur = weaker DoG) |
| `--sigma-scale` | 1.1 – 2.0 (1.4 – 1.8) | images mixing sharp and soft edges | toward 1.0 ⇒ lower `--dog-threshold` a lot (response nears 0) |
| `--tau` | 0.90 – 1.00 (0.95 – 1.0) | low-contrast images | lower tau ⇒ lower `--dog-threshold` |
| `--dog-threshold` | 0.005 – 0.08 (0.02 – 0.05) | **textured / noisy backgrounds** (dead zone on high-contrast subject over flat bg) | raise a lot ⇒ lower `--tile-fill` to keep lines |
| `--mag-threshold` | 0.0 – 0.3 (0.0 – 0.1) | speckly / jittery direction; little effect (mask already gates) | alternative to `--dog-threshold` |
| `--tile-fill` | 0.06 – 0.35 (0.10 – 0.20) | medium edge-density images; **step by ≥ 1/64 ≈ 0.016** or nothing changes | raise `--dog-threshold` ⇒ lower this |

Two constants live in the files (no flag): `defaultTile` (`main.go`, 4–16, coarser = fewer chars) and
`kernelRadiusSigmas` (`filter_gaussian.go`, 2–4).

## Recommandations
⚠️ Disclamer: Some images fail to decode. This is something that still needs to be looked at. Most jpg and png files work fine.
* **Set your terminal font size small (≈8px).** The art gets far more detail because a smaller font fits more character cells on screen. This cannot be done by the program: modern Windows terminals (Windows Terminal, VS Code, mintty/Git Bash) run through ConPTY, which ignores the console font/window APIs, so the font must be set manually in your terminal settings.
* Make your terminal as big as possible (maximize / fullscreen) before running — lower resolutions won't look that nice with the ascii characters.
* After the image is drawn the program waits silently — **press any key to exit** (this keeps the art on screen instead of the shell prompt overwriting it).
* Prefered to use more zoomed in items. Resize the image to the terminal size will usualy result in the loss of a lot of detail, so perhabs dont bother using super detailed images

## Todos
* Fix image decoding issue
* ~~Draw edges around areas that have a big contrast difference~~ ✅ `--edges` (DoG + threshold + Sobel, `| / _ \`)
* Look into color settings