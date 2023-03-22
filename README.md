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
