# imgcrop

Library for very efficient image manipulation.

See [Design Doc](./docs/DESIGN.md) for design philosophy and background.

## Installation

```shell
go get github.com/sebnyberg/imgcrop
```

## Introduction

This library performs memory efficient image manipulation. Currently it only supports cropping.

Note that this library uses BMPs for internal storage. Certain image metadata such as EXIF tags will be lost in the process.

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

## Examples

* [Cropping PNG (terrible perf)](./examples/png.go)
