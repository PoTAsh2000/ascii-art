# ASCII ART

## How to use
* You need to have [GO](https://go.dev/) installed to be able to run this script
* Install the **term** and **resize** packages:
    * go get golang.org/x/term
    * go get github.com/nfnt/resize
* Choose the image you want to convert in the ```func main() { ... }``` function
* ▶️ run the program with ```go run .``` in the src directory

## Recommandations
⚠️ Disclamer: Some images fail to decode. This is something that still needs to be looked at. Most jpg and png files work fine.
* Make you terminal as big as posible, beause lower resolutions wont look that nice with the ascii charaters
* Prefered to use more zoomed in items. Resize the image to the terminal size will usualy result in the loss of a lot of detail, so perhabs dont bother using super detailed images

## Todos
* Fix image decoding issue
* Draw edges around areas that have a big contrast difference
* Look into color settings