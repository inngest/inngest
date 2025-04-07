# UULID 

https://godoc.org/github.com/sudhirj/uulid.go

This package is a bridge between the UUID and the ULID specifications for creating unique identifiers. 

Both specs specify a large number, 16 bytes / 128 bits long as being the generated identifier. 

In the UUID spec, as defined in [RFC4122](https://tools.ietf.org/html/rfc4122), the entire number (all 128 bits) are completely random, and the representation is 36 hexadecimal (base 16) characters encoded in the following format: `XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX`. This is very commonly supported spec, and most database systems have native handling for this representation that stores it as an efficient [16]byte array under the covers. Most primary keys index are also B-Tree indexes that allow sorting and range queries, but because of the completely random nature of the identifier these capabilities are mostly wasted. 

A [ULID spec](https://github.com/ulid/spec) has since been developed that provides an alternative to the generation and representation that has benefits for modern applications - namely dedicating the first 48 bits to a millisecond precision unix timestamp, and the remaining 80 bits to randomness. This provides all the randomness (possibly even more) guarantees that the UUID spec provides, while encoding time data into the ID. This is especially useful in identifying data objects that are naturally chronological, like events, inserts, updates, news, or any feed of actions. 

The ULID also has a more efficient 26 character string representation (in Base 32) that is sortable. This is important when using the ID in NoSQL systems as sort keys, or even in regular RDBMS systems where the primar key can now allow chronological queries, which are often the most common type of query for immutable data. 

This package represents a [16]byte array as a type called UULID, and provides convenience methods to seamlessly switch between the two generations and representations. This is especially useful when you want to use a ULID in a data system that natively supports only UUIDs or vice versa. 

## Usage

Generate an identifier using the methods available
```go
id := NowUULID() // uses time.Now() with crypto/rand randomness
idWithTime := NewTimedUULID(someTime) // uses the given time with crypto/rand randomness
idWithContent := NewContentUULID(time.Now(), contentReader) // uses the SHA1 of the given content to provide "randomness" - the ID will no longer be random, but idempotent to the given time and content.
```

Once you've created the IDs, you can either get the ULID or the UUID representations based on your needs

```go
id.UUIDString() // 016f1873-dff7-ed3a-6745-04e60ea72957
id.ULIDString() // 01DWC77QZQXMX6EH84WR7AEAAQ 
```

This `UUIDString` is what you'd want to use on a system like PostgreSQL that has native handling for the UUID type. If you're a on a NoSQL system like DynamoDB or Mongo, the `ULIDString` might be a better way to go. 

To switch between the two, you can use the parsing convenience methods:
```go
MustParseUUID("016f1873-dff7-ed3a-6745-04e60ea72957").ULIDString() // 01DWC77QZQXMX6EH84WR7AEAAQ
MustParseULID("01DWC77QZQXMX6EH84WR7AEAAQ").UUIDString() // 016f1873-dff7-ed3a-6745-04e60ea72957 
```

To run queries against your data, you can use the time only UULIDs to generate your bounds:
```go
queryUULID := NewTimeOnlyUULID(time.Unix(1576481036, 999999999)) 
queryUULID.ULIDString() // 01DW6SF6P70000000000000000
queryUULID.UUIDString() // 016f0d97-9ac7-0000-0000-000000000000
```

These time-only zeroed IDs can be used in `<`, `<=`, `>`, `>=` and range queries against your primary key indexes, in the same way you might run queries against ISO8601 timestamps.    
