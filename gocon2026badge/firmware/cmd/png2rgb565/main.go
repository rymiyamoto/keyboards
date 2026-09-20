package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
)

func main() {
	in := flag.String("in", "", "input PNG path")
	out := flag.String("out", "", "output RGB565BE path")
	flag.Parse()

	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: png2rgb565 -in input.png -out output.rgb565")
		os.Exit(2)
	}

	input, err := os.Open(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer input.Close()

	img, err := png.Decode(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode png: %v\n", err)
		os.Exit(1)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 240 || bounds.Dy() != 240 {
		fmt.Fprintf(os.Stderr, "expected 240x240 PNG, got %dx%d\n", bounds.Dx(), bounds.Dy())
		os.Exit(1)
	}

	data := make([]byte, 0, bounds.Dx()*bounds.Dy()*2)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			data = appendRGB565BE(data, img, x, y)
		}
	}

	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}

func appendRGB565BE(data []byte, img image.Image, x, y int) []byte {
	r16, g16, b16, _ := img.At(x, y).RGBA()
	r := uint16(r16 >> 8)
	g := uint16(g16 >> 8)
	b := uint16(b16 >> 8)
	v := ((r & 0xF8) << 8) | ((g & 0xFC) << 3) | (b >> 3)
	return append(data, byte(v>>8), byte(v))
}
