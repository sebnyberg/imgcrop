# Cropping large images

Exploration into cropping large images.

## Background and goals

The background to this exploration is a need for cropping large (100MB+) images in a fashion that keeps the memory and CPU footprint as small as possible. The hidden proprietary problem driving this exploration is to support high-performance, lossless, concurrent cropping of large images for Data Science purposes.

Areas of interest:

1. Image file format
2. Block- vs pixel-by-pixel copy
3. Seekable compressed stream (zstd)
4. io_uring
5. mmap and madvise

### Image file format 

Let's consider different file formats. Theres JPEG (lossy) and PNG (lossless) which provides low size images. This is great for websites, but not great for cropping. For example, it is not possible to predictably jump to a specific pixel; all pages of the image must be fed from disk to the page cache, to user-space, only to be discarded. Not great.

### Pixel-by-pixel vs block copy

In compressed formats, it is not possible to calculate how many bytes represent a certain number of pixels. To find out, the image must first be decompressed in memory before being cropped.

Having less data is a very big advantage. Depending on the type of image, lossless compression can reduce total size drastically (up to 80%). So whether predictable pixel size is better than compression comes down to image characteristics and I/O vs CPU.

Decompression using e.g. zstd is heavily performance optimized, making use of vectorized instructions such as SVE / AVX. So a minimum requirement to beat a compressed stream would be to not parse pixels in any way, utilizing similar vector instructions for copying data. In theory, it may even be possible to perform these copies in kernel space, since the userspace program does not care about the contents of the bytes, only their offsets in the original file.

### Seekable compressed stream

Depending on the crop placement, skipping the portion of the file that is irrelevant to reading the image may increase performance. An interesting best-of-both worlds (hopefully) approach would be to compress a predictable-pixel-size image such as BMP with zstd and use [ZSTD seekable compression format](https://github.com/facebook/zstd/blob/dev/contrib/seekable_format/zstd_seekable_compression_format.md) implemented by [this excellent package](https://github.com/SaveTheRbtz/zstd-seekable-format-go).

### io_uring

For concurrent cropping performance, IO uring can help with asynchronously reading from many images at once while minimizing byte copying between user and kernel space.

I haven't had a usecase for io_uring yet so looking forward to learning what it can and cannot do.

### mmap and madvise

`mmap(2)` may enhance performance over regular `open(2)`, `lseek(2)` and `read(2)`.

Additionally, `madvise(2)` and in particular `MADV_SEQUENTIAL` can inform the kernel of the sequential nature of reading image contents.

## Downloading an image to test with

ESA has some amazing images of space over at <https://esahubble.org/images/>.

Tests will fail unless you first download one of these and put it at `./testdata/img.tiff`:

```shell
curl -o ./testdata/img.tiff https://esahubble.org/media/archives/images/original/heic0707a.tif
```

## Cutting a bitmap

The goal here is to cut a bitmap from disk without 

Easy. Just use the stdlib.

To get the BMP header, use unsafe and link in the header parser from `x/images/bmp` like a lad:

```shell
//go:linkname decodeConfig golang.org/x/image/bmp.decodeConfig
func decodeConfig(r io.Reader) (config image.Config, bitsPerPixel int, topDown bool, allowAlpha bool, err error)
```

`decodeConfig` reads the header, advancing the reader position. For repeatedly cropping 

