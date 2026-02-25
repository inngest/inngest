# MINLZ FORMAT SPECIFICATION V1.0.0

All implementations are requested to state: "This implements the MinLZ specification v1.0"

Furthermore, if a subset of features is supported, it should state this clearly.

A reference decoder is provided. If there is any ambiguity in the specification, 
the behavior of the reference decoder should be followed.

The spec versioning follows [semantic versioning](https://semver.org/).

* Major version numbers indicate breaking changes.
* Minor version numbers indicate added functionality that will not be readable by previous versions.
* Patch version numbers indicate non-breaking additions to the spec.

# BLOCK FORMAT

MinLZ is a LZ77-style compressor with a fixed size, byte-oriented encoding.

All values are encoded with full-byte offsets as an interleaved stream, 
where operations and literals are intermixed, similar to LZ4 and Snappy.

This specification defines the decoding format. 
The encoding of specific content is not defined and may change for any implementation.

The basic structure is similar to Snappy, but most encodings have been adjusted.

## 1.0 MinLZ Indicator

An initial byte of value 0 indicates that this is MinLZ encoded block.

A 0-byte block is allowed, and is encoded as a single byte with a 0 value.

If the first byte is not 0, this can be used to handle seamless 
fallback to Snappy.
Decoders *may* implement fallback to Snappy and provide seamless decoding
of the block if this value is non-zero.

## 1.1 Block Size 

A block starts with the uncompressed length up to a maximum of 2^24,
stored as an unsigned varint.

Maximum uncompressed block size is 8 MiB = 8,388,608 bytes.

If this value is 0, the rest of the block should be emitted as literals.

A compressed block may not be bigger than the decompressed block after reading the header.

## 2 MinLZ Encoding

Each block is encoded as a sequence of literals and copy/repeat commands.

Each element starts with a tag byte, and the lower two bits of this tag 
byte signal what type of element will follow:

| Tag | Meaning                              |
|-----|--------------------------------------|
| 0   | Literal(s) or Repeat previous offset |
| 1   | Copy with 10-bit offset              |
| 2   | Copy with 16-bit offset              |
| 3   | Copy with up to 21-bit offset        |

The interpretation of the upper six bits is tag-dependent.

Values spanning multiple bytes are stored in little-endian order.

### 2.1 Literals/Repeat (tag 00)

The literal length is stored differently depending on the length
of the literal block.

Optionally, instead of literals, a repeat copy can be specified, 
copying from the previous offset, or 1 if at the beginning of block

The value of the tag byte should be interpreted as follows:

| Bits | Meaning | Description                        |
|------|---------|------------------------------------|
| 0-1  | Tag     | Always 0                           |
| 2    | Repeat  | When set do repeat copy            |
| 3-7  | Length  | Length of literal or repeat block. |

Length follows the tag, and is encoded like this: 

| Value | Literals          | 
|-------|-------------------|
| 0-28  | 1 + Value         | 
| 29    | 30 + Read 1 byte  |
| 30    | 30 + Read 2 bytes | 
| 31    | 30 + Read 3 bytes |

Repeats are handled as copies, but with no offset specified. 
See below how offsets are handled. 

Literals follow the tag or extended length field. 

### 2.2 Copies

Copies are references back into previous decompressed data, telling
the decompressor to reuse data it has previously decoded.
They encode two values: The _offset_, saying how many bytes back
from the current position to read, and the _length_, how many bytes
to copy.

Backreferences that go past the end of the block 
(offset > current decompressed position) are not allowed.

As in most LZ77-based compressors, the length can be larger than the offset,
yielding a form of run-length encoding (RLE). For instance,
"xababab" could be encoded as

`<literal: "xab"> <copy: offset=2 length=4>`

There are several different kinds of copy elements, depending on
the number of bytes to be copied (length), and how far back the
data to be copied is (offset).

Furthermore, a "repeat" can be emitted, which will use the last offset for a copy,
but with a new length. Initial repeat offset is 1. So an initial RLE can be encoded as

`<start of block> <literal: "x"> <repeat: length=4>`

This will emit "xxxxx"

Matches shorter than 4 bytes cannot be represented, except for repeats.
Longer offset representations have minimum offsets, mainly to help decompression speed.

#### 2.2.1 Fused Literals+Copy

Copies with 2 or 3 byte offset can contain up to 4 literals in their encoding.
The literals must be emitted before the copy. 

The offset is the offset of the destination *after* the copy, 
so the copy offset is the same as if the literals were emitted separately.

See section 2.5.1 and 2.5.2 for details on fused copy operations.

Encoding should prefer fused literal+copy when tied for size, since the fused operation
will typically decode faster than separate operations. 

### 2.3 Copy1 with 1-byte offset (tag 01)

These elements can encode lengths between [4..273] bytes and offsets
between [1..1024] bytes. 

| Bits | Meaning   | Description                                                  |
|------|-----------|--------------------------------------------------------------|
| 0-1  | Tag       | Always 1                                                     |
| 2-5  | Length    | Length of copy<br/>Values are 0-15. See decoding table below |
| 6-7  | Offset LB | Lower 2 bits of offset                                       |

| Bits | Meaning   | Description       |
|------|-----------|-------------------|
| 0-7  | Offset UB | Offset Upper bits |

Offset is 1 more than stored value. The minimum offset is 1, and the maximum is 1024.

| Value | Output Length    |
|-------|------------------|
| 0-14  | 4 + Value        |
| 15    | 18 + Read 1 byte |

Maximum length is therefore 18 + 255 = 273

The extra length byte is stored *after* the offset byte if present.

Minimum encoded length 2 bytes, max 3 bytes with length 19 -> 273. 
Longer matches should emit a repeat with the extra bytes, or use Copy2.

### 2.4 Copy2 with 2-byte offset (tag 10)

These elements can encode lengths between [4...] bytes and offsets
between [64...65599] bytes.

| Bits | Meaning | Description                                                   |
|------|---------|---------------------------------------------------------------|
| 0-1  | Tag     | Always 2                                                      |
| 2-7  | Length  | Length of copy.<br/>Values are 0-63. See decoding table below |

Offsets are encoded as 2 little-endian bytes following the tag.
The minimum offset is 64 which should be added to the stored value.
The maximum backreference offset is therefore 65,599.

| Bits  | Meaning     | Description               |
|-------|-------------|---------------------------|
| 0-15  | Offset      | Offset + 64 `[64->65599]` |

Lengths are encoded as follows:

| Value | Output            |
|-------|-------------------|
| 0-60  | 4 + Value         |
| 61    | 64 + Read 1 byte  |
| 62    | 64 + Read 2 bytes |
| 63    | 64 + Read 3 bytes |

Minimum encoded length 3 bytes, max 6 bytes.

When both Copy1 and Copy2 have similar encoded length (mainly length 19->64), 
prefer copy2 as decoding will be faster.

### 2.5 Fused Copy2/3 (tag 11)

Tag can contain either a copy with 2 or 3 byte offsets and fused literals.

The third bit indicates if this is a copy2 or copy3.

| Bits | Meaning        | Description                                 |
|------|----------------|---------------------------------------------|
| 0-1  | Tag            | Always 3                                    |
| 2    | Copy3          | 1 when Copy3, 0 when Fused Copy2.           |
| 3-4  | Literal Length | Number of literals to emit before the copy. | 

The literal length is shared as bit 3-4, but Fused Copy2 has a minimum of 1. 

#### 2.5.1 Fused Copy2

Fused Copy2 offers a short 4->11 byte copy with 16 bit offset, preceded by 1-4 literals. 

| Bits | Meaning                          |
|------|----------------------------------|
| 0-1  | Tag. Always 3                    |
| 2    | Copy3. Always 0                  |
| 3-4  | Literal length + 1 `[1->4]`      |
| 5-7  | Copy Length + 4 `[4->11]`        |
| 8-23 | 16 bit offset + 64 `[64->65599]` |
| 24-> | 1-4 Literals                     |

For literal + copy2, a 2-byte offset will follow the tag, then the immediate(s) will follow.

A repeat operation can be used to extend the copy.

Encoded fused copy2 is 3 bytes, with 1-4 additional literals.

### 2.5.2 Copy3 (Optionally Fused)

Offsets are encoded as 21 bits, with 0 -> 3 fused literals. 
The minimum offset is 65,536 and must be added the stored offset.
The maximum backreference offset is therefore 2,162,687 (1<<21 + 65535).

| Bits  | Meaning                            |
|-------|------------------------------------|
| 0-1   | Tag. Always 3                      |
| 2     | Copy3. Always 1.                   |
| 3-4   | Literal length `[0->3]`            |
| 5-10  | Copy Length. 6 bits. (See table)   |
| 11-31 | 21 bit offset + 65536 `[64K->2MB]` |
| 32->x | 0-3 Extended Length Bytes          |
| x->   | 0-3 Literals                       |

Length is encoded as follows:

| Value | Output            |
|-------|-------------------|
| 0-60  | 4 + Value         |
| 61    | 64 + Read 1 byte  |
| 62    | 64 + Read 2 bytes |
| 63    | 64 + Read 3 bytes |

Minimum encoded Copy3 length is 4 bytes. 
Max length is 7 bytes, plus up to 3 literals.

Note that the shortest length Copy3 does not gain anything, 
except possibly avoid a literal tag, if there are fused literals,
or a repeat can be set up for copy.

## 3 DICTIONARY FORMAT

TBD.

# STREAM FORMAT

Follows [Snappy Framing Format](https://github.com/google/snappy/blob/main/framing_format.txt) - 
but with modifications to allow to be easily backwards compatible with Snappy/S2. 

### 1. General structure

The file consists solely of chunks, lying back-to-back with no padding
in between. Each chunk consists first a single byte of chunk identifier,
then a three-byte little-endian length of the chunk in bytes (from 0 to
16,777,215 inclusive), and then the data if any. The four bytes of chunk
header is not counted in the data length.

The different chunk types are listed below. The first chunk must always
be the stream identifier chunk (see section 4.1, below).

### 2. File type identification

The following identifiers for this format are recommended where appropriate.
However, note that none have been registered officially, so this is only to
be taken as a guideline. 

    File extension:         .mz
    MIME type:              application/x-minlz-compressed
    HTTP Content-Encoding:  x-minlz-compressed

Individual blocks contain no corruption detection, so these should not be exchanged.
However, if software produces them they should use the `.mzb` extension. 

### 3. Checksum format

Some chunks have data protected by a checksum (the ones that do will say so
explicitly). The checksums are always masked CRC-32Cs.

A description of CRC-32C can be found in [RFC 3720](https://datatracker.ietf.org/doc/html/rfc3720),
section 12.1, with examples in section B.4.

Checksums are not stored directly, but masked, as checksumming data and
then its own checksum can be problematic. The masking is the same as used
in Apache Hadoop: Rotate the checksum by 15 bits, then add the constant
0xa282ead8 (using wraparound as normal for unsigned integers). This is
equivalent to the following C code:

```
uint32_t mask_checksum(uint32_t x) {
    return ((x >> 15) | (x << 17)) + 0xa282ead8;
}
```

Note that the masking is reversible.

The checksum is always stored as a four-byte long integer, in little-endian.

### 4. Chunk types

The currently supported chunk types are described below. The list may
be extended in the future.

| ID      | Description                   | See Section |
|---------|-------------------------------|-------------|
| 0       | (legacy compressed Data)      | 4.3         |
| 1       | Uncompressed Data             | 4.3         |
| 2, 3    | MinLZ Compressed Block        | 4.4, 4.5    |
| 32      | EOF                           | 4.6         |
| 4-63    | (reserved, non-skippable)     | 4.8         |
| 64      | Stream Index                  | 4.12        |
| 65-127  | (reserved, skippable)         | 4.9         |
| 128-191 | (user defined, skippable)     | 4.10        |
| 192-253 | (user defined, non-skippable) | 4.11        |
| 254     | Padding                       | 4.7         |
| 255     | Stream identifier             | 4.1         |

### 4.1. Stream identifier (chunk type 0xff)

The stream identifier is always the first element in the stream.
It is exactly six bytes long and starts with "MinLz" in ASCII. 
This means that a valid MinLZ framed stream always starts with the bytes:

    (type) (length)        M    i    n    L    z
    0xff   0x06 0x00 0x00  0x4d 0x69 0x6e 0x4c 0x7a <stream info>

The final byte of the identifier is a block size indicator.

| Bits | Description               |
|------|---------------------------|
| 0-3  | Max block size indicator  |
| 4-5  | Reserved, must be ignored |
| 6-7  | Reserved, must be 0       |

#### 4.1.1 Max Block Size

The value is the log2 of the maximum block size minus 10.

So if the value is `0x4`, the maximum block size is 2^(4+10) (16KiB).
The maximum block size is 2^23 (8MiB), so the maximum identifier is 13,
which can also be used if for some reason the maximum block size is not known.
This size only applies to content frames.

Decoders may choose not to decode streams based on the maximum block size.
Decoders *must* reject any value > 13.

To allow concatenation, a stream identifier can follow an EOF chunk. 

### 4.2. Uncompressed data (chunk type 0x01)

Uncompressed data chunks allow a compressor to send uncompressed,
raw data; this is useful if, for instance, incompressible or
near-incompressible data is detected, and faster decompression is desired.

As in the compressed chunks, the data is preceded by its own masked
CRC-32C (see section 3).

An uncompressed data chunk minus the CRC should not exceed the maximum block size
as indicated by the Stream identifier.

### 4.3. Legacy Compressed data (chunk type 0x00) — BACKCOMPAT ONLY

Type 0x00 compressed chunks are not allowed in MinLZ streams.

Instead, use type 0x02, which indicates a MinLZ compressed block.

### 4.4. MinLZ Compressed data (chunk type 0x02)

A MinLZ block *without* the MinLZ identifier (initial 0 byte).

A CRC32C (see section 3) checksum of the *uncompressed* 
data is stored at the beginning of the block.

Chunks with 0 decompressed bytes are not allowed, 
as well as blocks with decompressed size less than compressed size.

### 4.5. MinLZ Compressed data - Compressed CRC (chunk type 0x03).

A MinLZ block *without* the MinLZ identifier (initial 0 byte).

A CRC32C (see section 3) checksum of the *compressed* 
data is stored at the beginning of the block.

Chunks with 0 decompressed bytes are not allowed,
as well as blocks with decompressed size less than compressed size.

If possible prefer type `0x02` over this.

### 4.6 EOF (chunk type 0x20)

The end of the stream is indicated by a chunk with the ID `0x20`.
This allows detection of truncated streams.

The output size of the stream is encoded as an unsigned varint as the only content.
It is allowed to add an empty block to not validate size.
Maximum chunk size is 10 bytes (64 bit varint encoded).

Encoders should always emit this chunk. 
Decoders can optionally reject streams that do not have this chunk.

If this is the first chunk in the stream, the stream is empty.

If there are multiple Stream identifier (chunk type 0xff) chunks in the stream,
each must have a corresponding EOF chunk and the count resets to 0 at each Stream identifier.

### 4.7. Padding (chunk type 0xfe)

Padding chunks allow a compressor to increase the size of the data stream
so that it complies with external demands, e.g. that the total number of
bytes is a multiple of some value.

All bytes of the padding chunk, except the chunk byte itself and the length,
should be zero, but decompressors must not try to interpret or verify the
padding data in any way.

### 4.8. Reserved unskippable chunks (chunk types 0x04-0x3f)

These are reserved for future expansion. A decoder that sees such a chunk
should immediately return an error, as it must assume it cannot decode the
stream correctly.

Future versions of this specification may define meanings for these chunks.

### 4.9. Reserved skippable chunks (chunk types 0x40-0x7f)

These are also reserved for future expansion, but unlike the chunks
described in 4.5, a decoder seeing these must skip them and continue
decoding.

Future versions of this specification may define meanings for these chunks.

### 4.10. User defined skippable chunks (chunk types 0x80-0xbf)

These are allowed for user-defined data. A decoder that does not recognize
the chunk type should skip it and continue decoding.

Users should use additional checks to ensure that the chunk is of the expected type.

Future versions of this specification will not define meanings for these chunks.

### 4.11. User defined non-skippable chunks (chunk types 0xc0-0xfd)

These are allowed for user-defined data. If users do not recognize these chunks,
decoding should stop.

Users should use additional checks to ensure that the chunk is of the expected type.

Future versions of this specification will not define meanings for these chunks.


### 4.12 Index (chunk type 0x40) — OPTIONAL

Each block is structured as a skippable block, with the chunk ID 0x40.

Decoders are free to skip this block.

The block can be read from the front, but contains information
so it can easily be read from the back as well.

Numbers are stored as fixed size little endian values or [zigzag encoded](https://developers.google.com/protocol-buffers/docs/encoding#signed_integers) [base 128 varints](https://developers.google.com/protocol-buffers/docs/encoding),
with un-encoded value length of 64 bits, unless other limits are specified.

| Content                              | Format                                                                                                                        |
|--------------------------------------|-------------------------------------------------------------------------------------------------------------------------------|
| ID, `[1]byte`                        | Always 0x40.                                                                                                                  |
| Data Length, `[3]byte`               | 3 byte little-endian length of the chunk in bytes, following this.                                                            |
| Header `[6]byte`                     | Header, must be `[115, 50, 105, 100, 120, 0]` or in text: "s2idx\x00".                                                        |
| UncompressedSize, Varint             | Total Uncompressed size.                                                                                                      |
| CompressedSize, Varint               | Total Compressed size if known. Should be -1 if unknown.                                                                      |
| EstBlockSize, Varint                 | Block Size, used for guessing uncompressed offsets. Must be >= 0.                                                             |
| Entries, Varint                      | Number of Entries in index, must be < 65536 and >=0.                                                                          |
| HasUncompressedOffsets `byte`        | 0 if no uncompressed offsets are present, 1 if present. Other values are invalid.                                             |
| UncompressedOffsets, [Entries]VarInt | Uncompressed offsets. See below how to decode.                                                                                |
| CompressedOffsets, [Entries]VarInt   | Compressed offsets. See below how to decode.                                                                                  |
| Block Size, `[4]byte`                | Little Endian total encoded size (including header and trailer). Can be used for searching backwards to start of block.       |
| Trailer `[6]byte`                    | Trailer, must be `[0, 120, 100, 105, 50, 115]` or in text: "\x00xdi2s". Can be used for identifying block from end of stream. |

For regular streams the uncompressed offsets are fully predictable,
so `HasUncompressedOffsets` allows to specify that compressed blocks all have
exactly `EstBlockSize` bytes of uncompressed content.

Entries *must* be in order, starting with the lowest offset,
and there *must* be no uncompressed offset duplicates.  
Entries *may* point to the start of a skippable block,
but it is then not allowed to also have an entry for the next block since
that would give an uncompressed offset duplicate.

There is no requirement for all blocks to be represented in the index.
In fact there is a maximum of 65535 block entries in an index.

The writer can use any method to reduce the number of entries.
An implicit block start at 0,0 can be assumed.

### Decoding entries:

```
// Read Uncompressed entries.
// Each assumes EstBlockSize delta from previous.
for each entry {
    uOff = 0
    if HasUncompressedOffsets == 1 {
        uOff = ReadVarInt // Read value from stream
    }
   
    // Except for the first entry, use previous values.
    if entryNum == 0 {
        entry[entryNum].UncompressedOffset = uOff
        continue
    }
    
    // Uncompressed uses previous offset and adds EstBlockSize
    entry[entryNum].UncompressedOffset = entry[entryNum-1].UncompressedOffset + EstBlockSize + uOff
    entryNum++
}


// Guess that the first block will be 50% of uncompressed size.
// Integer truncating division must be used.
CompressGuess := EstBlockSize / 2

// Read Compressed entries.
// Each assumes CompressGuess delta from previous.
// CompressGuess is adjusted for each value.
for each entry {
    cOff = ReadVarInt // Read value from stream
    
    // Except for the first entry, use previous values.
    if entryNum == 0 {
        entry[entryNum].CompressedOffset = cOff
        continue
    }
    
    // Compressed uses previous and our estimate.
    entry[entryNum].CompressedOffset = entry[entryNum-1].CompressedOffset + CompressGuess + cOff
        
     // Adjust compressed offset for next loop, integer truncating division must be used. 
     CompressGuess += cOff/2
     
     entryNum++               
}
```

To decode from any given uncompressed offset `(wantOffset)`:

* Iterate entries until `entry[n].UncompressedOffset > wantOffset`.
* Start decoding from `entry[n-1].CompressedOffset`.
* Discard `entry[n-1].UncompressedOffset - wantOffset` bytes from the decoded stream.

This is similar to S2, except the ID is 0x40 instead of 0x99.

# Implementation Notes

This section contains guidelines for implementation and use. 
None of these are strict requirements. 

MinLZ is designed for a certain speed/size tradeoff.

It is designed to be used in scenarios where encoding and decoding speed is critical.

Unless decompression speed is critical, it is not designed for long-term storage,
and formats like [zstandard](https://facebook.github.io/zstd/) should be considered for better compression.
Formats like xz/bzip2 also offer excellent compression, but these offer even more reduced decompression speed.

## No entropy nor dynamic encoding

MinLZ by design only offers static encoding types and no entropy coding of remainder literals, 
making decoding possible with no tables.

While it was considered, all conditional decoding (decoding based on previous operation or output position)
was avoided to simplify the decoder.

## Independent block streams

A primary design choice is to make blocks on streams fully independent to facilitate
independent compression and decompression.

This will make streams seekable, and even without an index, streams can be 
skipped forwards without decompression, and blocks can be decoded concurrently.

The maximum block size of 8MB is designed to minimize the size impact of this.

## Speed Optimizations

This section describes tricks that can help achieve maximum speed.

Decompression has been designed to make use of modern CPU branch prediction.

### Margin-specific code

For decompression it can be beneficial to have 2 parts of the decompressors:
A primary decoding loop, that runs while there is some input and output margin and another
that deals when at the end of the input/output, which has stricter checks.

### No-overlap Encodings

Minimum offsets to certain encodings mainly exist to avoid overlapping copies and make a 64 byte/loop 
copy safe to do. 

Copy2/Copy3 operations guarantee there are no copies with an offset less than 64 bytes.
This means that these can use a bigger copy loop without the need to worry about these.

This effectively moves this branch from the encoder to the decoder.  

### Safe Fused Literals

Fused literals can always be safely copied with a 4 byte copy, 
since there will always "follow" a match with at least 4 bytes that will "fix up" any extra literals.

Therefore, fused copies are typically faster than a separate literal + copy operation.

### Encoding Tips

 * Prefer fused literals, even when encoded size is equal.
 * Avoid 1 and 2 byte repeats. The encoding exists mostly for flexibility.
 * Prefer Copy2 over Copy1 when encoded size is the same.
