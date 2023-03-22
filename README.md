# imgcrop

Library for efficient cropping of very large images.

See [Design Doc](./docs/DESIGN.md) for background and design considerations.

## Stability

:warning: This package is under development and in flux. :warning:

## Installation

```shell
go get github.com/sebnyberg/imgcrop
```

## Usage

```go
src, err := os.Open("big.bmp")
if err != nil {
    return err
}

dst, err := os.Open("cropped.bmp")
if err != nil {
    return err
}

offx := 5500
offy := 7500
width := 1500
height := 2000
region := image.Rect(offx, offy, width, height)
err = bmpx.Crop(src, dest, region)
if err != nil {
    return err
}
```

## Performance

Benchmark that crops different sizes from a 1.2GiB 29566x14321 px image and stores the result in an output file, randomizing x- and y-offset with each crop:


```
go test -test.v -test.run=NONE -test.bench='^\QBenchmarkBMP\E$'
goos: linux
goarch: amd64
pkg: github.com/sebnyberg/imgcrop/bmpx
cpu: Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz
BenchmarkBMP
BenchmarkBMP/100X100
BenchmarkBMP/100X100-6         	    1748	    613716 ns/op
BenchmarkBMP/100X800
BenchmarkBMP/100X800-6         	     261	   4610897 ns/op
BenchmarkBMP/100X6400
BenchmarkBMP/100X6400-6        	      32	  37329677 ns/op
BenchmarkBMP/800X100
BenchmarkBMP/800X100-6         	    1609	    747651 ns/op
BenchmarkBMP/800X800
BenchmarkBMP/800X800-6         	     202	   6291474 ns/op
BenchmarkBMP/800X6400
BenchmarkBMP/800X6400-6        	      22	  47321119 ns/op
BenchmarkBMP/6400X100
BenchmarkBMP/6400X100-6        	     636	   1881620 ns/op
BenchmarkBMP/6400X800
BenchmarkBMP/6400X800-6        	      72	  15344632 ns/op
BenchmarkBMP/6400X6400
BenchmarkBMP/6400X6400-6       	       9	 130090939 ns/op
PASS
ok  	github.com/sebnyberg/imgcrop/bmpx	15.100s
```

`

## Examples

* [Cropping PNG (terrible perf)](./examples/png.go)

## Testing

This library has been tested by a friend externally, trust me. :eyes:

