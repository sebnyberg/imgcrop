# Cropping large images

Exploration into cropping large images.

## Background and goals

The background to this exploration is a need for cropping large (100MB+) images in a fashion that keeps the memory and CPU footprint as small as possible. The hidden proprietary problem driving this exploration is to support high-performance, lossless, concurrent cropping of large images for Data Science purposes.

Another goal is to support stream cropping as the image is being recieved over e.g. a TCP socket.

Areas of interest:

1. Image file format
2. Block- vs pixel-by-pixel copy
3. Seekable compressed stream (zstd)
4. io_uring
5. mmap and madvise
6. performance of existing libraries

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

## Pre-requisites

To run these benchmarks, a large image and `libvips` is required.

ESA has some amazing images of space over at <https://esahubble.org/images/>.

Tests will fail unless you first download one of these and put it at `./testdata/img.tiff`:

```shell
curl -o ./testdata/img.tiff https://esahubble.org/media/archives/images/original/heic0707a.tif
```

For `libvips` installation instructions, see <https://github.com/davidbyttow/govips>.

## Results

### Image file formats

#### BMP

At first glance, BMP is exactly what is needed. A single image per file, easy format, typically uncompressed.

However, BMP is not widely supported for some reason. For example, BMP is not supported by Go wrappers for libvips, although libvips can sometimes read BMP images given that certain libraries are linked in during compilation. Ugh.

Aside from cutting, libvips is best-in-class for image transformation. Losing vips support is not worth it.

#### TIFF

TIFF (a Thousand Incompatible File Formats) is more like a file system than an image format.

TIFF calls its image descriptors Image File Directories (IFDs) and supports having many different images in the same file.

Support for many files isn't necessarily a bad thing. However, the standard does enforce placement of image descriptors. The IFDs do not have to be placed at the start of the file; they can be placed anywhere. Most sane people would argue that descriptors (headers) should typically come *before* the data it describes. However, the Go stdlib and most libraries seem to put the IFD at the end of the file, making it impossible to crop the images as a stream.

This forced me to write a barebones TIFF-compatible encoder that shuffles the order of IFDs so that the header can be efficiently read as a stream.

