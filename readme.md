# ASCII ART

## How to use
* You need to have [GO](https://go.dev/) installed to be able to run this script
* Install the **term** and **resize** packages:
    * go get golang.org/x/term
    * go get github.com/nfnt/resize
* ▶️ run the program, passing the image path as the only argument: ```go run main.go test_images/master-chief.jpg```

## Recommandations
⚠️ Disclamer: Some images fail to decode. This is something that still needs to be looked at. Most jpg and png files work fine.
* **Set your terminal font size small (≈8px).** The art gets far more detail because a smaller font fits more character cells on screen. This cannot be done by the program: modern Windows terminals (Windows Terminal, VS Code, mintty/Git Bash) run through ConPTY, which ignores the console font/window APIs, so the font must be set manually in your terminal settings.
* Make your terminal as big as possible (maximize / fullscreen) before running — lower resolutions won't look that nice with the ascii characters.
* After the image is drawn the program waits silently — **press any key to exit** (this keeps the art on screen instead of the shell prompt overwriting it).
* Prefered to use more zoomed in items. Resize the image to the terminal size will usualy result in the loss of a lot of detail, so perhabs dont bother using super detailed images

## Todos
* Fix image decoding issue
* Draw edges around areas that have a big contrast difference
* Look into color settings