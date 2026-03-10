# MinLZ

MinLZ is a LZ77-type compressor with a fixed byte-aligned encoding, in the similar class to Snappy and LZ4.

The goal of MinLZ is to provide a fast, low memory compression algorithm that can be used for fast compression of data, 
where encoding and/or decoding speed is the primary concern. 

MinLZ is designed to operate *faster than IO* for both compression and decompression and be a viable "always on"
option even if some content already is compressed.
If slow compression is acceptable, MinLZ can be configured to produce high compression ratio, 
but retain high decompression speed.

* Best in class compression
* Block or Streaming interfaces
* Very fast decompression, even as pure Go
* AMD64 encoder+decoder assembly
* Adjustable Compression (3 levels)
* Concurrent stream Compression
* Concurrent stream Decompression
* Skip forward in compressed stream via independent blocks
* Random seeking with optional indexes
* Stream EOF validation
* Automatic stream size padding
* Custom encoders for small blocks
* Skippable/Non-skippable user blocks
* Detailed control of memory under decompression
* Fast detection of pre-compressed data
* Powerful commandline utility

This package implements the MinLZ specification v1.0 in Go.

For format specification see the included [SPEC.md](SPEC.md).

# Changelog

* [v1.0.0](https://github.com/minio/minlz/releases/tag/v1.0.0)
  * [Initial Release Blog Post](https://blog.min.io/minlz-compression-algorithm/).

# Usage

[![Go Reference](https://pkg.go.dev/badge/minio/minlz.svg)](https://pkg.go.dev/github.com/minio/minlz?tab=subdirectories)
[![Go](https://github.com/minio/minlz/actions/workflows/go.yml/badge.svg)](https://github.com/minio/minlz/actions/workflows/go.yml)

MinLZ can operate on *blocks* up to 8 MB or *streams* with unlimited length.

Blocks are the simplest, but do not provide any output validation. 
Blocks are mainly useful for small data sizes.

Streams are a collection of independent blocks, which each have checksums and EOF checks, 
which ensures against corruption and truncation.

3 compression levels are provided:

* Level 1, "Fastest": Provides the fastest compression with reasonable compression. 
* Level 2, "Balanced": Provides a good balance between compression and speed. ~50% the speed of the fastest level.
* Level 3, "Smallest": Provides the smallest output possible. Not tuned for speed.

A secondary option to control speed/compression is adjusting the block size.
See "Writer Block Size" section below.

## Blocks

MinLZ provides a block encoding interface with blocks up to 8MB.
Blocks do not perform any data integrity check of the content, 
so additional checksum is recommended.

A basic roundtrip looks like this:

```Go
   compressed, err := minlz.Encode(nil, src, minlz.LevelBalanced)
   if err != nil {
       // Handle error
   }
   
    decompressed, err := minlz.Decode(nil, compressed)
    if err != nil {
        // Handle error 
    }
```

In both cases, a destination buffer can be provided, which will be overwritten. 
If the destination buffer is too small, an appropriately sized buffer will be allocated and returned.

It is possible to get the decompressed buffer size by using `minlz.DecodedLen(block []byte) (int, error)`.

You can use the predefined `LevelFastest`, `LevelBalanced` or `LevelSmallest` which correspond to
levels 1,2 and 3 respectively.

MinLZ does not track the compressed size of buffers and the decode input must match the output exactly.
Extra bytes given to decompression will return an error.

It is possible to use `minlz.TryEncode`, which will only return compressed bytes if the output size 
is strictly less than input.
Use `minlz.AppendEncoded` and `minlz.AppendDecoded` to append to existing slices.

## Streams

Streams provide much more safety and allow for unlimited length encoding, 
as well as seeking and concurrent encoding/decoding.

Generally, you do not need buffering on the input or output side as reads and writes 
are done in rather big blocks. 
Reading and writing data on streams are buffered, 
and only non-concurrent will block for input/output.    

When dealing with many streams, it is recommended to re-use the Readers and Writers.
If you are dealing with short streams, consider limiting the concurrency, so 
`block_size * concurrency` doesn't exceed the expected stream size.

### Encoding

Streams are the recommended way to use MinLZ.
They provide end-to-end validation against corruption and truncation.

```Go
    // Create a new stream encoder.
    // The encoder will write to the provided io.Writer.
    enc := minlz.NewWriter(output)
	
	// We defer a call to Close.
	// This will flush any pending data and indicate we have reached the end of the stream.
	defer enc.Close()

	// Write data to the encoder.
	// The encoder will write the compressed data to the underlying io.Writer.
	js := json.NewEncoder(enc)
	err := js.Encode(data)
```

Encoders can be reused by calling `Reset` on them with another output.
This will reset the encoder to its initial state.

The encoder supports the [io.ReaderFrom](https://pkg.go.dev/io#ReaderFrom) interface, 
which can be used for encoding data from an io.Reader. 
This will typically be faster than writing data to the encoder, since it avoids a memory copy.

If you have a single big buffer to encode, you can use the `EncodeBuffer([]byte) error` 
to encode it. This will encode the buffer with minimal overhead.
If you plan to do multiple writes, use the regular `Write` function.

### Options

There are various options that can be set on the stream encoder.
This can be used to control resource usage on compression and some aspects of decompression.
If invalid options are set, the encoder will return an error when used.

We will cover the most common options here. Refer to the godoc for a complete list.

#### Writer Compression Level 

The `WriterLevel` option controls the compression level of the stream encoder.

You can use the predefined `LevelFastest`, `LevelBalanced` or `LevelSmallest` which correspond to 
levels 1,2 and 3 respectively.

Setting level 0 will disable compression and write the data as an uncompressed stream.

The default level is `LevelBalanced`.

#### Writer Block Size

The `WriterBlockSize` allows to set the maximum size of each block on the stream encoder.
The blocksize - rounded up to a power of 2 - is communicated in the stream, and 
the decoder will use this to allocate memory during decompression.

Smaller blocks will take up less memory on both compression and decompression, 
but will result in a larger output. 

Block size further allows trading off speed vs. size; Here is a sample chart of how 
speed and block size can correlate, using the fastest encoder setting:

| Block Size | Output Size   | E MB/s | Size | E Speed | D Speed |
|------------|---------------|--------|------|---------|---------|
| 8MB        | 840,198,535   | 6419   | 100% | 100%    | 100%    |
| 4MB        | 862,923,396   | 8470   | 103% | 132%    | 124%    |
| 2MB        | 921,750,327   | 9660   | 110% | 150%    | 131%    |
| 1MB        | 950,153,883   | 10407  | 113% | 162%    | 125%    |
| 512KB      | 1,046,061,990 | 11459  | 125% | 179%    | 113%    |

Input is a `3,325,605,752` byte [CSV file](https://files.klauspost.com/compress/nyc-taxi-data-10M.csv.zst)
compressed on a 16 core CPU.

The actual scaling mostly depends on the amount of CPU L2 cache (speed) 
and the nature of the compressed data (size). 

Decompression speed is affected similarly, but less predictably, 
since it is more likely to be limited by memory throughput, 
and larger output also tends to affect it more negatively.

If your software is very sensitive to GC stoppages, also note that with assembly 
single block de/compression cannot be pre-empted, so stop-the-world events may take 
longer on bigger blocks.

The default block size is 2 MB.

#### Writer Concurrency

The `WriterConcurrency` option allows setting the number of concurrent blocks that can be compressed.
Higher concurrency will increase the throughput of the encoder, but will also increase memory usage. 

If `WriterConcurrency(1)` is used no async goroutines will be used and the encoder will run in the calling goroutine.

The default concurrency is `GOMAXPROCS`.

### Decoding

Decoding streams mostly just involves sending the compressed stream to a Reader.

Anything accepting an `io.Reader` as input will then be able to read the decompressed data.

```Go
	// Create a new stream decoder. 
	// The encoder will read from the provided io.Reader.
	dec := minlz.NewReader(input)
	
	// Read decompressed input.
	js := json.NewDecoder(dec)
	err := js.Decode(&data)
```

If you would like the output to be written to an `io.Writer`, the easiest is to use
the `WriteTo` functionality.

```Go
	// Our input and output
	in, _ := os.Create("input.mz")
	out, _ := os.Create("output.txt")
	
	// Create a new stream decoder
	dec := minlz.NewReader(in)

	// Write all decompressed data to output
	n, err := dec.WriteTo(out)
	fmt.Println("Wrote", n, "bytes. Error:", err)
```

The `DecompressConcurrent` has similar functionality to `WriteTo`, but allows specifying the concurrency.
By default `WriteTo` uses `runtime.NumCPU()` or at most 8 concurrent decompressors.
Besides offering higher throughput using `DecompressConcurrent` will also make input reads async when used. 

For memory-sensitive systems, the maximum block size can be set below 8MB. For this use the `ReaderMaxBlockSize(int)`
option.

#### Skipping and Seeking

Streams can be skipped forward by calling `(*Reader).Skip(n int64) error`.
This will skip forward in the stream by `n` bytes. 
Intermediate blocks be read, but will not be decompressed unless the skip ends inside the block.

Full random seeking is supported by using an *index*. An index can be created when the stream is encoded.
The index can either be added to the stream or stored separately. 
For existing streams the `IndexStream(r io.Reader) ([]byte, error)` function can be used to create an index.

To add an index at the end of streams, use the `WriterAddIndex()` option when creating the writer, 
then the index will be added to the stream when it is closed.
To keep the index separate, use the `(*Writer).CloseIndex() ([]byte, error)` method to retrieve 
the index when finishing a stream.

To get a fully seekable reader use `(*Reader).ReadSeeker(index []byte) (*ReadSeeker, error)`.
The returned reader will implement `io.Seeker`, `io.ReaderAt` in addition to the existing `Reader` methods
and can be used to seek to any position in the stream.

If an index is not provided in the call, the reader will attempt to read the index from the end of the stream.
If the input stream does not support `io.Seeker` an error will be returned.

## Custom User Data

Streams can contain user-defined data, that isn't part of the stream. 
Each "chunk" has an ID, which allows for processing of different types.

This data can either be "skippable" - meaning it is ignored if the user hasn't provided a handler for these.
If the chunk is non-skippable, the encoder will error out if this chunk isn't handled by the user.

`MinUserSkippableChunk` is the minimum chunk id with user data and `MaxUserSkippableChunk` is the maximum.

`MinUserNonSkippableChunk` is the minimum ID that will not automatically be skipped if unhandled by the user. 
Finally `MaxUserNonSkippableChunk` is the final ID that can be used for this.

The custom data will not be compressed or modified in any way.

```go
func ExampleWriterAddUserChunk() {
	var buf bytes.Buffer
	w := minlz.NewWriter(&buf)
	// Add a skippable chunk
	w.AddUserChunk(minlz.MinUserSkippableChunk, []byte("Chunk Custom Data"))
	// Write content to stream.
	w.Write([]byte("some data"))
	w.Close()

	// Read back what we wrote.
	r := minlz.NewReader(&buf)
	r.SkippableCB(minlz.MinUserSkippableChunk, func(sr io.Reader) error {
		b, err := io.ReadAll(sr)
		fmt.Println("Callback:", string(b), err)
		return err
	})

	// Read stream data
	b, err := io.ReadAll(r)
	fmt.Println("Stream data:", string(b))

	//OUTPUT:
	//Callback: Chunk Custom Data <nil>
	//Stream data: some data
}
```

The maximum single chunk size is 16MB, but as many chunks as needed can be added. 

## Build Tags

The following build tags can be used to control which speed improvements are used:

* `noasm` disables all assembly.
* `nounsafe` disables all use of unsafe package.
* `purego` disables assembly and unsafe usage.  

Using assembly/non-assembly versions will often produce slightly different output.

We will support 2 releases prior to current Go release version.  

This package has been extensively fuzz tested to ensure that no data input can cause 
crashes or excessive memory usage.

When doing fuzz testing, use `-tags=nounsafe`. Non-assembly functions will also be tested, 
but for completeness also test with `-tags=purego`.

# Performance

## BLOCKS

Individual block benchmarks should be considered carefully - and can be hard to generalize, 
since they tend to over-emphasize specific characteristics of the content.

Therefore, it will be easy to find counter-examples to the benchmarks, where specific patterns suit a 
specific compressor better than others. 
We present a few examples from the [Snappy benchmark set](https://github.com/google/snappy/tree/main/testdata).
As a benchmark this set has an over-emphasis on text files.

Blocks are compressed/decompress using 16 concurrent threads on an AMD Ryzen 9 3950X 16-Core Processor.
Click below to see some sample benchmarks compared to Snappy and LZ4:

### Protobuf Sample


| Compressor   | Size   | Comp MB/s | Decomp MB/s | Reduction % |
|--------------|--------|----------:|-------------|-------------|
| MinLZ 1      | 17,613 |    27,837 |     116,762 |      85.15% |
| MinLZ 1 (Go) | 17,479 |    22,036 |      61,652 |      85.26% |
| MinLZ 2      | 16,345 |    12,797 |     103,100 |      86.22% |
| MinLZ 2 (Go) | 16,345 |     9,732 |      52,964 |      86.22% |
| MinLZ 3      | 14,766 |       210 |     126,385 |      87.55% |
| MinLZ 3 (Go) | 14,766 |           |      68,411 |      87.55% |
| Snappy       | 23,335 |    24,052 |      61,002 |      80.32% |
| Snappy (Go)  | 23,335 |    10,055 |      35,699 |      80.32% |
| LZ4 0        | 18,766 |    12,649 |     137,553 |      84.18% |
| LZ4 0 (Go)   | 18,766 |           |      64,092 |      84.18% |
| LZ4 9        | 15,844 |    12,649 |     139,801 |      86.64% |
| LZ4 9 (Go)   | 15,844 |           |      66,904 |      86.64% |

![Compression vs Size](img/pb-block.png)

Source file: https://github.com/google/snappy/blob/main/testdata/geo.protodata


### HTML Sample

<details>
  <summary>Click To See Data + Charts (102,400 bytes input)</summary>

| Compressor   | Size   | Comp MB/s | Decomp MB/s | Reduction % |
|--------------|--------|----------:|-------------|-------------|
| MinLZ 1      | 20,184 |    17,558 |      82,292 |      80.29% |
| MinLZ 1 (Go) | 19,849 |    15,035 |      32,327 |      80.62% |
| MinLZ 2      | 17,831 |     9,260 |      58,432 |      82.59% |
| MinLZ 2 (Go) | 17,831 |     7,524 |      25,728 |      82.59% |
| MinLZ 3      | 16,025 |       180 |      80,445 |      84.35% |
| MinLZ 3 (Go) | 16,025 |           |      33,382 |      84.35% |
| Snappy       | 22,843 |    17,469 |      44,765 |      77.69% |
| Snappy (Go)  | 22,843 |     8,161 |      21,082 |      77.69% |
| LZ4 0        | 21,216 |     9,452 |     101,490 |      79.28% |
| LZ4 0 (Go)   | 21,216 |           |      40,674 |      79.28% |
| LZ4 9        | 17,139 |     1,407 |      95,706 |      83.26% |
| LZ4 9 (Go)   | 17,139 |           |      39,709 |      83.26% |

![Compression vs Size](img/html-block.png)

Source file: https://github.com/google/snappy/blob/main/testdata/html

</details>

### URL List Sample

<details>
  <summary>Click To See Data + Charts (702,087 bytes input)</summary>

| Compressor   | Size    | Comp MB/s | Decomp MB/s | Reduction % |
|--------------|---------|----------:|-------------|-------------|
| MinLZ 1      | 268,803 |     9,774 |      30,961 |      61.71% |
| MinLZ 1 (Go) | 260,937 |     7,935 |      17,362 |      62.83% |
| MinLZ 2      | 230,280 |     5,197 |      26,871 |      67.20% |
| MinLZ 2 (Go) | 230,280 |     4,280 |      13,926 |      67.20% |
| MinLZ 3      | 207,303 |       226 |      28,716 |      70.47% |
| MinLZ 3 (Go) | 207,303 |           |      15,256 |      70.47% |
| Snappy       | 335,492 |     9,398 |      24,207 |      52.22% |
| Snappy (Go)  | 335,492 |     4,683 |      12,359 |      52.22% |
| LZ4 0        | 299,342 |     4,462 |      51,220 |      57.36% |
| LZ4 0 (Go)   | 299,342 |           |      23,242 |      57.36% |
| LZ4 9        | 252,182 |       638 |      45,295 |      64.08% |
| LZ4 9 (Go)   | 252,182 |           |      16,240 |      64.08% |

![Compression vs Size](img/urls-block.png)

Source file: https://github.com/google/snappy/blob/main/testdata/urls.10K

</details>

### Serialized GEO data Sample

<details>
  <summary>(184,320 bytes input)</summary>

| Compressor   | Size   | Comp MB/s | Decomp MB/s | Reduction % |
|--------------|--------|----------:|-------------|-------------|
| MinLZ 1      | 63,595 |     8,319 |      26,170 |      65.50% |
| MinLZ 1 (Go) | 62,087 |     7,601 |      12,118 |      66.32% |
| MinLZ 2      | 54,688 |     5,932 |      24,688 |      70.33% |
| MinLZ 2 (Go) | 52,752 |     4,690 |      10,566 |      71.38% |
| MinLZ 3      | 46,002 |       230 |      28,083 |      75.04% |
| MinLZ 3 (Go) | 46,002 |           |      12,877 |      75.04% |
| Snappy       | 69,526 |    10,198 |      19,754 |      62.28% |
| Snappy (Go)  | 69,526 |     5,031 |       8,712 |      62.28% |
| LZ4 0        | 66,506 |     5,355 |      45,305 |      63.92% |
| LZ4 0 (Go)   | 66,506 |           |      15,757 |      63.92% |
| LZ4 9        | 50,439 |        88 |      52,877 |      72.64% |
| LZ4 9 (Go)   | 50,439 |           |      18,171 |      72.64% |

![Compression vs Size](img/geo-block.png)

Source file: https://github.com/google/snappy/blob/main/testdata/kppkn.gtb

</details>

In overall terms, we typically observe that:

* The fastest mode typically beats LZ4 both in speed and output size.
* The fastest mode is typically equal to Snappy in speed, but significantly smaller.
* The "balanced" mode typically beats the best possible LZ4 compression, but much faster.
* Without assembler MinLZ is mostly the fastest option for compression.
* LZ4 is decompression speed king.
* Snappy decompression is usually slowest — especially without assembly.

We encourage you to do your own testing with realistic blocks.

You can use `λ mz c -block -bench=10 -verify -cpu=16 -1 file.ext` with our commandline tool to test speed of block encoding/decoding.

## STREAMS

For fair stream comparisons, we run each encoder at its maximum block size
or max 4MB,  while maintaining independent blocks where it is an option.
We use the concurrency offered by the package.

This means there may be further speed/size tradeoffs possible for each, 
so experiment with fine tuning for your needs.

Blocks are compressed/decompress using 16 core AMD Ryzen 9 3950X 16-Core Processor.

### JSON Stream

Input Size: 6,273,951,764 bytes

| Compressor  | Speed MiB/s |          Size | Reduction | Dec MiB/s |
|-------------|------------:|--------------:|----------:|----------:|
| MinLZ 1     |      14,921 |   974,656,419 |    84.47% |     3,204 |
| MinLZ 2     |       8,877 |   901,171,279 |    85.64% |     3,028 |
| MinLZ 3     |         576 |   742,067,802 |    88.17% |     3,835 |
| S2 Default  |      15,501 | 1,041,700,255 |    83.40% |     2,378 |
| S2 Better   |       9,334 |   944,872,699 |    84.94% |     2,300 |
| S2 Best     |         732 |   826,384,742 |    86.83% |     2,572 |
| LZ4 Fastest |       5,860 | 1,274,297,625 |    79.69% |     2,680 |
| LZ4 Best    |       1,772 | 1,091,826,460 |    82.60% |     2,694 |
| Snappy      |         951 | 1,525,176,492 |    75.69% |     1,828 |
| Gzip L5     |         236 |   938,015,731 |    85.05% |       557 |

![Compression vs Size](img/json-v1-comp.png)
![Decompression Speed](img/json-v1-decomp.png)

Source file: https://files.klauspost.com/compress/github-june-2days-2019.json.zst


### CSV Stream

<details>
  <summary>Click To See Data + Charts</summary>

Input Size: 3,325,605,752 bytes

| Compressor | Speed MiB/s | Size          | Reduction |
|------------|-------------|---------------|-----------|
| MinLZ 1    | 9,193       |   937,136,278 |    72.07% |
| MinLZ 2    | 6,158       |   775,823,904 |    77.13% |
| MinLZ 3    | 338         |   657,162,410 |    80.66% |
| S2 Default | 10,679      | 1,093,516,949 |    67.12% |
| S2 Better  | 6,394       |   884,711,436 |    73.40% |
| S2 Best    | 400         |   773,678,211 |    76.74% |
| LZ4 Fast   | 4,835       | 1,066,961,737 |    67.92% |
| LZ4 Best   | 732         |   903,598,068 |    72.83% |
| Snappy     | 553         | 1,316,042,016 |    60.43% |
| Gzip L5    | 128         |   767,340,514 |    76.93% |

![Compression vs Size](img/csv-v1-comp.png)

Source file: https://files.klauspost.com/compress/nyc-taxi-data-10M.csv.zst

</details>

### Log data

<details>
  <summary>Click To See Data + Charts</summary>

Input Size: 2,622,574,440 bytes

| Compressor | Speed MiB/s | Size        | Reduction |
|------------|-------------|-------------|-----------|
| MinLZ 1    | 17,014      | 194,361,157 |    92.59% |
| MinLZ 2    | 12,696      | 174,819,425 |    93.33% |
| MinLZ 3    | 1,351       | 139,449,942 |    94.68% |
| S2 Default | 17,131      | 230,521,260 |    91.21% |
| S2 Better  | 12,632      | 217,884,566 |    91.69% |
| S2 Best    | 1,687       | 185,357,903 |    92.93% |
| LZ4 Fast   | 6,115       | 216,323,995 |    91.75% |
| LZ4 Best   | 2,704       | 169,447,971 |    93.54% |
| Snappy     | 1,987       | 290,116,961 |    88.94% |
| Gzip L5    | 498         | 142,119,985 |    94.58% |

![Compression vs Size](img/logs-v1-comp.png)

Source file: https://files.klauspost.com/compress/apache.log.zst

</details>

### Serialized Data

<details>
  <summary>Click To See Data + Charts</summary>

Input Size: 1,862,623,243 bytes

| Compressor | Speed MiB/s | Size        | Reduction |
|------------|-------------|-------------|-----------|
| MinLZ 1    | 10,701      | 604,315,773 |    67.56% |
| MinLZ 2    | 5,712       | 517,472,464 |    72.22% |
| MinLZ 3    | 250         | 480,707,192 |    74.19% |
| S2 Default | 12,167      | 623,832,101 |    66.51% |
| S2 Better  | 5,712       | 568,441,654 |    69.48% |
| S2 Best    | 324         | 553,965,705 |    70.26% |
| LZ4 Fast   | 5,090       | 618,174,538 |    66.81% |
| LZ4 Best   | 617         | 552,015,243 |    70.36% |
| Snappy     | 929         | 589,837,541 |    68.33% |
| Gzip L5    | 166         | 434,950,800 |    76.65% |

![Compression vs Size](img/msgp-v1-comp.png)

Source file: https://files.klauspost.com/compress/github-ranks-backup.bin.zst

</details>

### Backup (Mixed) Data

<details>
  <summary>Click To See Data + Charts</summary>

Input Size: 10,065,157,632 bytes

| Compressor  | Speed MiB/s | Size          | Reduction |
|-------------|-------------|---------------|-----------|
| MinLZ 1     | 9,356       | 5,859,748,636 |    41.78% |
| MinLZ 2     | 5,321       | 5,256,474,340 |    47.78% |
| MinLZ 3     | 259         | 4,855,930,368 |    51.76% |
| S2 Default  | 10,083      | 5,915,541,066 |    41.23% |
| S2 Better   | 5,731       | 5,455,008,813 |    45.80% |
| S2 Best     | 319         | 5,192,490,222 |    48.41% |
| LZ4 Fastest | 5,065       | 5,850,848,099 |    41.87% |
| LZ4 Best    | 287         | 5,348,127,708 |    46.86% |
| Snappy      | 732         | 6,056,946,612 |    39.82% |
| Gzip L5     | 171         | 4,916,436,115 |    51.15% |

![Compression vs Size](img/10gb-v1-comp.png)

Source file: https://mattmahoney.net/dc/10gb.html

</details>

Our conclusion is that the new compression algorithm provides a good compression increase,
while retaining the ability to saturate pretty much any IO either with compression or
decompression given a moderate amount of CPU cores.


## Why is concurrent block and stream speed so different?

In most cases, MinLZ will be limited by memory bandwidth.

Since streams consist of mostly "unseen" data, it will often mean that memory
reads are outside any CPU cache.

Contrast that to blocks, where data has often just been read/produced and therefore
already is in one of the CPU caches.
Therefore, block (de)compression will more often take place with data read from cache
rather than a stream, where data can be coming from memory.

Even if data is streamed into cache, the "penalty" will still have to paid at some
place in the chain. So streams will mostly appear slower in benchmarks.


# Commandline utility

Official releases can be downloaded from the [releases](https://github.com/minio/minlz/releases) section
with binaries for most platforms.

To install from source execute `go install github.com/minio/minlz/cmd/mz@latest`.

## Usage

```
λ mz
MinLZ compression tool vx.x built at home, (c) 2025 MinIO Inc.
Homepage: https://github.com/minio/minlz

Usage:
Compress:     mz c [options] <input>
Decompress:   mz d [options] <input>
 (cat)    :   mz cat [options] <input>
 (tail)   :   mz tail [options] <input>

Without options 'c' and 'd' can be omitted. Extension decides if decompressing.

Compress file:    mz file.txt
Compress stdin:   mz -
Decompress file:  mz file.txt.mz
Decompress stdin: mz d -
```

Note that all option sizes KB, MB, etc. are base 1024 in the commandline tool.

Speed indications are base 10.

### Compressing

<details>
  <summary>Click To Compression Help</summary>

```
Usage: mz c [options] <input>

Compresses all files supplied as input separately.
Output files are written as 'filename.ext.mz.
By default output files will be overwritten.
Use - as the only file name to read from stdin and write to stdout.

Wildcards are accepted: testdir/*.txt will compress all files in testdir ending with .txt
Directories can be wildcards as well. testdir/*/*.txt will match testdir/subdir/b.txt

File names beginning with 'http://' and 'https://' will be downloaded and compressed.
Only http response code 200 is accepted.

Options:
  -1    Compress faster, but with a minor compression loss
  -2    Default compression speed (default true)
  -3    Compress more, but a lot slower
  -bench int
        Run benchmark n times. No output will be written
  -block
        Use as a single block. Will load content into memory. Max 8MB.
  -bs string
        Max block size. Examples: 64K, 256K, 1M, 8M. Must be power of two and <= 8MB (default "8M")
  -c    Write all output to stdout. Multiple input files will be concatenated
  -cpu int
        Maximum number of threads to use (default 32)
  -help
        Display help
  -index
        Add seek index (default true)
  -o string
        Write output to another file. Single input file only
  -pad string
        Pad size to a multiple of this value, Examples: 500, 64K, 256K, 1M, 4M, etc (default "1")
  -q    Don't write any output to terminal, except errors
  -recomp
        Recompress MinLZ, Snappy or S2 input
  -rm
        Delete source file(s) after success
  -safe
        Do not overwrite output files
  -verify
        Verify files, but do not write output

Example:

λ mz c apache.log
Compressing apache.log -> apache.log.mz 2622574440 -> 170960982 [6.52%]; 4155.2MB/s
```
</details>

## Decompressing

<details>
  <summary>Click To Decompression Help</summary>

```
Usage: mz d [options] <input>

Decompresses all files supplied as input. Input files must end with '.mz', '.s2' or '.sz'.
Output file names have the extension removed. By default output files will be overwritten.
Use - as the only file name to read from stdin and write to stdout.

Wildcards are accepted: testdir/*.txt will decompress all files in testdir ending with .txt
Directories can be wildcards as well. testdir/*/*.txt will match testdir/subdir/b.txt

File names beginning with 'http://' and 'https://' will be downloaded and decompressed.
Extensions on downloaded files are ignored. Only http response code 200 is accepted.

Options:
  -bench int
        Run benchmark n times. No output will be written
  -block
        Decompress single block. Will load content into memory. Max 8MB.
  -block-debug
        Print block encoding
  -c    Write all output to stdout. Multiple input files will be concatenated
  -cpu int
        Maximum number of threads to use (default 32)
  -help
        Display help
  -limit string
        Return at most this much data. Examples: 92, 64K, 256K, 1M, 4M        
  -o string
        Write output to another file. Single input file only
  -offset string
        Start at offset. Examples: 92, 64K, 256K, 1M, 4M. Requires Index
  -q    Don't write any output to terminal, except errors
  -rm
        Delete source file(s) after success
  -safe
        Do not overwrite output files
  -tail string
        Return last of compressed file. Examples: 92, 64K, 256K, 1M, 4M. Requires Index
  -verify
        Verify files, but do not write output

Example:

λ mz d apache.log.mz
Decompressing apache.log.mz -> apache.log 170960982 -> 2622574440 [1534.02%]; 2660.2MB/s
```
</details>

Tail, Offset and Limit can be made to forward to the next newline by adding `+nl`.

For example `mz d -c -offset=50MB+nl -limit=1KB+nl enwik9.mz` will skip 50MB, 
search for the next newline, start outputting data. 
After 1KB, it will stop at the next newline.

Partial files - decoded with tail, offset or limit will have `.part` extension.

# Snappy/S2 Compatibility

MinLZ is designed to be easily upgradable from [Snappy](https://github.com/google/snappy) 
and [S2](https://github.com/klauspost/compress/tree/master/s2#s2-compression).

Both the streaming and block interfaces in the Go port provide seamless
compatibility with existing Snappy and S2 content.
This means that any content encoded with either will be decoded correctly by MinLZ.

Content encoded with MinLZ cannot be decoded by Snappy or S2.  

| Version        | Snappy Decoder | S2 Decoder | MinLZ Decoder |
|----------------|----------------|------------|---------------|
| Snappy Encoder | ✔              | ✔          | ✔ (*)         |
| S2 Encoder     | x              | ✔          | ✔ (*)         |
| MinLZ Encoder  | x              | x          | ✔             |

(*) MinLZ decoders *may* implement fallback to S2/Snappy. 
This is however not required and ports may not support this.

# License

MinLZ is Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Based on code from [snappy-go](https://github.com/golang/snappy) project.

# Ports

Reference code is provided in the `internal/reference` folder.
This provides simplified, but explicit versions of the block de/encoder;
stream and index decoders with minimal dependencies.

Currently, there are no ports of MinLZ to other languages. 
If you are interested in porting MinLZ to another language, open a discussion topic.

If you do a port, feel free to send in a PR for this table:

| Language | Repository Link                                                                         | License    | Block Read | Block Write | Stream Read | Stream Write | Index | Snappy Fallback |
|----------|-----------------------------------------------------------------------------------------|------------|------------|-------------|-------------|--------------|-------|-----------------|
| Go       | [github.com/minio/minlz](https://github.com/minio/minlz)                                | Apache 2.0 | ✅          | ✅           | ✅           | ✅            | ✅     | ✅               |  
| C        | [Experimental GIST](https://gist.github.com/klauspost/5796a5aa116a15eb7341ffa8427bbe7a) | CC0        | ✅          | ✅           |             |              |       |                 |                                                                                                                 


Indicated features must support all parts of each feature as described in the specification.
However, it is up to the implementation to decide the encoding implementation(s).  
