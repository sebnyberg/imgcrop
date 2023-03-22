# Cropping large images

Efficient cropping and resizing of large images; i.e. images that are around 1GiB in uncompressed size and 20'000 x 20'000 pixels.

## Goals and constraints

This library aims to be able to perform parallel cropping and resizing of ~100 images of size 1GiB within 8GB of RAM.

To keep memory footprint small, only transformations that can be efficiently applied with limited-size buffers will be allowed into this library. This means that certain multi-row operations will be allowed, others won't.

For performance reasons, the user must convert images to uncompressed TIFF or BMP before using the library. When applicable, this library will provide low-memory image format transformations to uncompressed TIFF and BMP.

For simplicity reasons, this library makes no guarantees about metadata such as EXIF tags being retained after transformation.

## Image file formats

### JPEG

So far, I don't know whether JPEG is streamable in any fashion, which would be a requirement for integration into this library. Its on my todo to thoroughly read the spec(s) and fork the Go stdlib and explore development of a streaming version.

It's worth noting that I have not yet seen a library that performs efficient JPEG cropping functionality in a library.

Interestingly, there used to exist a container format called JPEG File Interchange Format (JFIF) that stored byte offset markers that could be used to seek through the file quickly. The problem with JFIF is that it is deprecated and unused. It's just a historical artifact. 

### PNG

PNG appears to be streamable given a buffer size of two rows of pixels. For a reasonably large image, say 20'000 pixels wide, this results in a 320KiB row buffer, which is reasonable to manage.

However, png decompression is heavy on CPU use, so PNG should only be allowed as an incoming format and will need to be streamed to an uncompressed target e.g. TIFF before being processed by the library.

The plan is to fork the Go stdlib implementation and emit rows to a writer as they are decoded rather than gathering all bytes in-memory (`image.Image`).

### BMP

BMP does work great for cropping, but as a Windows file format is is not available or interoperable for libraries. For examble, the popular C library `libvips` does not support BMP out of the box.

### TIFF

TIFF - the god object of file formats. TIFF embeds images with multiple kinds of compression, sampling and color schemes, and the popular EXIF format is actually just TIFF in disguise.

TIFF is a very complicated format. As mentioned, EXIF uses TIFF. It is easy to accidentally drop or corrupt an image as it is being read or decoded, primarily due to how the TIFF file layout looks on disk.

Aside for its 8-byte header, TIFF makes no guarantees about layout. In fact, image data can be weaved between image metadata without breaking the spec (using stripes). Another sad property for stream cropping is that it is common practice to put image metadata at the end of the file, disallowing single-stream cropping. The reason for putting metadata at the end is because it makes it more portable - as users edit image metadata, only the end of the file will be utdated.

For this library to make effidient use of TIFF, this library should come up with a strict version of TIFF where the file can be read top-to-bottom, is seekable, uncompressed, and has a predefined header and IDF space for efficient copying.

#### Stripes

TIFF has a concept called `stripes`. A stripe is a set of (possible one) row(s) that have been written as one bytes chunk. Each stripe has a corresponding `StripeCount` and `StripeOffset`. Note that stripes do not have to be put in sequential order, but they usually are.

Usually, such as in Go's `x/image/tiff` stdlib package, all rows are written as a single stripe. In other words, `StripeCount` is the same as `ImageLength` (length is height in TIFF lingo).

Encoding the entire image as one or many stripes does not matter for uncompressed image lookup - the byte offset is already possible to calculate. However, for compressed, large images, it can be highly beneficial to encode rows as stripes since it allows skipping any pixel rows prior to the crop area.

Programs such as Photoshop make efficient use of stripes, but many languages (including Go) lack proper libraries for supporting more complex striping, such as using cJPEG + stripes. 

#### cJPEG

TIFF supports embedding JPEG, both lossy and lossless, through its cJPEG compression marker. However, even the Go stdlib does not support cJPEG (there is an open PR to add support). In theory, striped JPEG data should enable efficient cropping. Due to its lack of support however, I will not be supporting it here.

## Various performance concerns

### Pixel-by-pixel vs block copy

In compressed formats, it is not possible to calculate how many bytes represent a certain number of pixels. To find out, the image must first be decompressed in memory before being cropped.

Having less data is a very big advantage. Depending on the type of image, lossless compression can reduce total size drastically (up to 80%). So whether predictable pixel size is better than compression comes down to image characteristics and I/O vs CPU.

Decompression using e.g. zstd is heavily performance optimized, making use of vectorized instructions such as SVE / AVX. So a minimum requirement to beat a compressed stream would be to not parse pixels in any way, utilizing similar vector instructions for copying data. In theory, it may even be possible to perform these copies in kernel space, since the userspace program does not care about the contents of the bytes, only their offsets in the original file.

### Scan sharing

To limit memory usage when multiple clients request crops from the same image, a sort of [scan sharing](https://www.ibm.com/docs/en/db2/11.1?topic=methods-scan-sharing) could be employed. Either incoming crop requests are batched, or crops jump into ongoing scans in an online fashion. An online delta-interval-based scan is performed over the image, and byte slice references are sent to consumers one by one. AFAIK, the Go library does not manipulate byte arrays handed over to socket writes, so it should be fine to share byte slice references across consumers.

## Image manipulation libraries

### stdlib

It is possible to crop using stdlib's subImage interface. However, this also requires an image.Image object, which is an image serialized in memory, which is not reasonable for hundreds of 1GiB+ images.

### davidbyttow/govips and vipsimage/vips

[govips](https://github.com/davidbyttow/govips) provides Go bindings to [libvips](https://github.com/libvips/libvips). Sadly, it serializes file contents into byte slices before passing the data to vips, which offsets any advantage given by using vips to begin with. Due to this serialization, cropping a large uncompressed image is roughly 100'000 times slower than for this library.

[vipsimage/vips](https://github.com/vipsimage/vips) is more to my liking. It does not require vips startup/stop, and it allows for passing the input and output image path as a string rather than serialized byte array, allowing vips to efficiently read from the file. It seems however, perhaps through no fault of the bindings, that vips just isn't great for cropping.

I believe something was awry when testing `govips`, but here are some benchmarks run for a tiny image (around 200x100 pixels):

```
goos: linux
goarch: amd64
pkg: github.com/sebnyberg/imgcrop
cpu: Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz
BenchmarkTIFF
BenchmarkTIFF/sample
BenchmarkTIFF/sample/8X8-govips
BenchmarkTIFF/sample/8X8-govips-6     	       1	7531801740 ns/op
BenchmarkTIFF/sample/8X8-vipsimage    
BenchmarkTIFF/sample/8X8-vipsimage-6  	     385	   3057453 ns/op
BenchmarkTIFF/sample/8X64-govips      
BenchmarkTIFF/sample/8X64-govips-6    	       1	7479075532 ns/op
BenchmarkTIFF/sample/8X64-vipsimage   
BenchmarkTIFF/sample/8X64-vipsimage-6 	     391	   3034391 ns/op
BenchmarkTIFF/sample/64X8-govips      
BenchmarkTIFF/sample/64X8-govips-6    	       1	7548924592 ns/op
BenchmarkTIFF/sample/64X8-vipsimage   
BenchmarkTIFF/sample/64X8-vipsimage-6 	     404	   2975768 ns/op
BenchmarkTIFF/sample/64X64-govips     
BenchmarkTIFF/sample/64X64-govips-6   	       1	7516983678 ns/op
BenchmarkTIFF/sample/64X64-vipsimage  
BenchmarkTIFF/sample/64X64-vipsimage-6	     384	   3095654 ns/op
```

### vips CLI

For comparison, I ran VIPS via its CLI, extracting a 1000x1000 pixel area out of a 29000x14000 image as TIFF/LZW, TIFF/uncompressed and PNG:

```
$ /usr/bin/time -f 'mem_maxbytes: %M\nseconds: %e' \
    vips extract_area big-uncompressed.tif out.tiff 10000 5000 1000 1000
mem_maxkb: 616952
seconds: 0.66

$ /usr/bin/time -f 'mem_maxbytes: %M\nseconds: %e' \
    vips extract_area big.tif out.tiff 10000 5000 1000 1000
mem_maxkb: 472012
seconds: 3.17

$ /usr/bin/time -f 'mem_maxbytes: %M\nseconds: %e' \
    vips extract_area big.png out.tiff 10000 5000 1000 1000
mem_maxkb: 465784
seconds: 2.89
```

I doubt that a Go wrapper around VIPS can achieve better results than calling VIPS from the command line. Max 600MB RSS memory usage is roughly in line with what can be expected from [VIPS benchmarks](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) where an image takes roughly 100MB for a 10000x10000px image. It's only off by small factor.

Having 100 such images run in parallel would not meet the low memory usage criteria for this library.

## Compression

### LZW

WIP. Is supported by TIFF, and should be fast to decompress, so it is a possible allowed input format.

### ZSTD

ZSTD is an excellent way for this library to receieve an uncompressed image that can be byte addressable as it is being read. From some testing, it seems like Zstd achieves on-par results when applied to an uncompressed TIFF as LZW does when embedded in a TIFF (`big.tif` in the example below).

```
430M big-best.png
433M big-default.png
518M big-fastest.png
502M big.tif
658M big-plain-15.zstd.tif
832M big-plain-3.zstd.tif
1,6G big-plain.tif
```

#### Seekable zstd stream

Depending on the crop placement, skipping the portion of the file that is irrelevant to reading the image may increase performance. An interesting best-of-both worlds (hopefully) approach would be to compress a predictable-pixel-size image such as BMP with zstd and use [ZSTD seekable compression format](https://github.com/facebook/zstd/blob/dev/contrib/seekable_format/zstd_seekable_compression_format.md) implemented by [this excellent package](https://github.com/SaveTheRbtz/zstd-seekable-format-go).

## Kernel I/O optimization

### io_uring

For concurrent cropping performance, IO uring can help with asynchronously reading from many images at once while minimizing byte copying between user and kernel space.

I haven't had a usecase for io_uring yet so looking forward to learning what it can and cannot do.

### mmap and madvise

`mmap(2)` may enhance performance over regular `open(2)`, `lseek(2)` and `read(2)`.

Additionally, `madvise(2)` and in particular `MADV_SEQUENTIAL` can inform the kernel of the sequential nature of reading image contents.
