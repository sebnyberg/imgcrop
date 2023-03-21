# Cropping large images

Exploration into cropping large images.

## Background and goals

The background to this exploration is a need for cropping large (100MB+) images in a fashion that keeps the memory and CPU footprint as small as possible. The hidden proprietary problem driving this exploration is to support high-performance, lossless, concurrent sampling (cropping) of large images for Data Science purposes.

A peripheral goal is to support stream (forward-only) cropping as the image is being recieved over e.g. a TCP socket. Stream-based cropping is applicable to cropping of files too; minimizing seeks helps drastically with performance.

Areas of interest:

* Image file formats
* Various performance concerns
* Image manipulation libraries
* Compression & seekable compressed formats
* Kernel I/O optimization
* Benchmarks

## Image file formats

Let's consider image file formats.

There's JPEG (lossy) and PNG (lossless). Both work well for the web. 

If the solution requires sending lossless images, then PNG is the best choice. If the data is too large to send as PNG, JPEG is likely the best choice. There are other nuances, such as the effect JPEG compression has on certain types of images, such as text-on-background versus pictures, but in general this decision holds true.

The problem with both compressed formats such as JPEG and PNG for cropping is that they do not support skipping data to the region that is being cropped. There is no way of knowing which byte corresponds to which pixel unless the image has first been decompressed. 

Interestingly, there used to exist a JPEG container format called JPEG File Interchange Format (JFIF) that stored byte offset markers that could be used to seek through the file quickly. The problem with JFIF is that it isn't used anywhere anymore. It's just a historical artifact. 

BMP does work great for cropping, but as a Windows file format is is not available or interoperable for libraries. For examble, the popular C library `libvips` does not support BMP out of the box.

This brings us to TIFF - the god object of file formats. TIFF embeds images with multiple kinds of compression, sampling and color schemes, and the popular EXIF format is actually just TIFF in disguise. TIFF is both useful and sad.

### TIFF usefulness for cropping

TIFF has a concept called `stripes`. A stripe is a set of (possible one) row(s) that have been written as one bytes chunk. Each stripe has a corresponding `StripeCount` and `StripeOffset`. Note that stripes do not have to be put in sequential order, but they usually are.

Usually, such as in Go's `x/image/tiff` stdlib package, all rows are written as a single stripe. In other words, `StripeCount` is the same as `ImageLength` (length is height in TIFF lingo).

Encoding the entire image as one or many stripes does not matter for uncompressed image lookup - the byte offset is already possible to calculate. However, for compressed, large images, it can be highly beneficial to encode rows as stripes since it allows skipping any pixel rows prior to the crop area.

### TIFF sadness

TIFF is a very complicated format. As mentioned, EXIF uses TIFF. It is easy to accidentally drop or corrupt an image as it is being read or decoded, primarily due to how the TIFF file layout looks on disk.

TIFF places no restrictions on how the image should be laid out on disk. In fact, image pixel stripes can be weaved between image metadata without breaking the spec. Another sad property for stream cropping is that it is common practice to put image metadata at the end of the file, disallowing single-stream cropping. The reason for putting metadata at the end is because it makes it more portable - as users edit image metadata, only the end of the file will be utdated.

For the problem that I'm trying to solve, I decided to use TIFF's flexibility to create an inflexible kind of TIFF with statically defined properties, the header at the top, and striped data for cropping performance. The custom implementation is very unportable and untested, but should work well for high-performance cropping, and supports stream cropping.

## Various performance concerns

### Pixel-by-pixel vs block copy

In compressed formats, it is not possible to calculate how many bytes represent a certain number of pixels. To find out, the image must first be decompressed in memory before being cropped.

Having less data is a very big advantage. Depending on the type of image, lossless compression can reduce total size drastically (up to 80%). So whether predictable pixel size is better than compression comes down to image characteristics and I/O vs CPU.

Decompression using e.g. zstd is heavily performance optimized, making use of vectorized instructions such as SVE / AVX. So a minimum requirement to beat a compressed stream would be to not parse pixels in any way, utilizing similar vector instructions for copying data. In theory, it may even be possible to perform these copies in kernel space, since the userspace program does not care about the contents of the bytes, only their offsets in the original file.

## Image manipulation libraries

### govips

[govips](https://github.com/davidbyttow/govips) provides Go bindings to [libvips](https://github.com/libvips/libvips). Sadly, it serializes file contents into byte slices before passing the data to vips, which offsets any advantage given by using vips to begin with. Cropping a large image is roughly 100'000 times slower than the cropping I implemented Go with e.g. BMP.

<!-- [vipsimage/vips](https://github.com/vipsimage/vips) is much better. It allows for passing the input and output image path as a string rather than serialized byte array, allowing vips to efficiently read from the file. It seems however, perhaps through no fault of this library, that vips just isn't great for cropping. -->

## Compression

### Seekable compressed stream

Depending on the crop placement, skipping the portion of the file that is irrelevant to reading the image may increase performance. An interesting best-of-both worlds (hopefully) approach would be to compress a predictable-pixel-size image such as BMP with zstd and use [ZSTD seekable compression format](https://github.com/facebook/zstd/blob/dev/contrib/seekable_format/zstd_seekable_compression_format.md) implemented by [this excellent package](https://github.com/SaveTheRbtz/zstd-seekable-format-go).

## Kernel I/O optimization

### io_uring

For concurrent cropping performance, IO uring can help with asynchronously reading from many images at once while minimizing byte copying between user and kernel space.

I haven't had a usecase for io_uring yet so looking forward to learning what it can and cannot do.

### mmap and madvise

`mmap(2)` may enhance performance over regular `open(2)`, `lseek(2)` and `read(2)`.

Additionally, `madvise(2)` and in particular `MADV_SEQUENTIAL` can inform the kernel of the sequential nature of reading image contents.

## Benchmarks

### Pre-requisites

To run these benchmarks, a large image and `libvips` is required.

ESA has some amazing images of space over at <https://esahubble.org/images/>.

Tests will fail unless you first download one of these and put it at `./testdata/img.tiff`:

```shell
curl -o ./testdata/img.tiff https://esahubble.org/media/archives/images/original/heic0707a.tif
```

For `libvips` installation instructions, see <https://github.com/davidbyttow/govips>.
