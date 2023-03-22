# tiffx

Implements a small subset of TIFF for efficient parallelized cropping.

Puts a bunch of requirements on TIFF to make it more predictable and efficient for cropping:

* IFDs are put directly after the header, and all IFDs are set to a specific size.
* Header + IFDS + IFD values is at most 2048 bytes
* Directory entry overflow is placed after IFDs
* One image per file
* Only RGBA (32-bit pixel size)
* Only single strips
* Image always starts at byte position 2048 and continues until the end of the file
* No compression (it would require strips to be efficient)

## Benchmark comparison
