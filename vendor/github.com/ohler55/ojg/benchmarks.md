# OjG Benchmarks

Benchmarks were run from the ojg/cmd/benchmark directory with the command:

```
go run *.go
```

```

Parse string/[]byte
       json.Unmarshal           55916 ns/op    17776 B/op    334 allocs/op
         oj.Parse               39570 ns/op    18488 B/op    429 allocs/op
   oj-reuse.Parse               17881 ns/op     5691 B/op    364 allocs/op
        gen.Parse               28670 ns/op    18488 B/op    429 allocs/op
  gen-reuse.Parse               19619 ns/op     5691 B/op    364 allocs/op
        sen.Parse               30486 ns/op    18488 B/op    431 allocs/op
  sen-reuse.Parse               20018 ns/op     5708 B/op    366 allocs/op

   oj-reuse.Parse        █████████████████████▉ 3.13
  gen-reuse.Parse        ███████████████████▉ 2.85
  sen-reuse.Parse        ███████████████████▌ 2.79
        gen.Parse        █████████████▋ 1.95
        sen.Parse        ████████████▊ 1.83
         oj.Parse        █████████▉ 1.41
       json.Unmarshal    ▓▓▓▓▓▓▓ 1.00

Unmarshal []byte to type
       json.Unmarshal           44513 ns/op     5944 B/op    122 allocs/op
         oj.Unmarshal           41010 ns/op     9705 B/op    457 allocs/op
        sen.Unmarshal           41763 ns/op     9690 B/op    457 allocs/op

         oj.Unmarshal    ███████▌ 1.09
        sen.Unmarshal    ███████▍ 1.07
       json.Unmarshal    ▓▓▓▓▓▓▓ 1.00

Tokenize
       json.Decode              77026 ns/op    22600 B/op   1175 allocs/op
         oj.Tokenize             7883 ns/op     1976 B/op    156 allocs/op
        sen.Tokenize             8347 ns/op     1976 B/op    158 allocs/op

         oj.Tokenize     ████████████████████████████████████████████████████████████████████▍ 9.77
        sen.Tokenize     ████████████████████████████████████████████████████████████████▌ 9.23
       json.Decode       ▓▓▓▓▓▓▓ 1.00

Parse io.Reader
       json.Decode              63029 ns/op    32449 B/op    344 allocs/op
         oj.ParseReader         34289 ns/op    22583 B/op    430 allocs/op
   oj-reuse.ParseReader         25094 ns/op     9788 B/op    365 allocs/op
        gen.ParseReder          43859 ns/op    22585 B/op    430 allocs/op
  gen-reuse.ParseReder          23066 ns/op     9788 B/op    365 allocs/op
        sen.ParseReader         36991 ns/op    22585 B/op    432 allocs/op
  sen-reuse.ParseReader         23363 ns/op     9788 B/op    367 allocs/op
         oj.TokenizeLoad        13610 ns/op     6072 B/op    157 allocs/op
        sen.TokenizeLoad        12485 ns/op     6072 B/op    159 allocs/op

        sen.TokenizeLoad ███████████████████████████████████▎ 5.05
         oj.TokenizeLoad ████████████████████████████████▍ 4.63
  gen-reuse.ParseReder   ███████████████████▏ 2.73
  sen-reuse.ParseReader  ██████████████████▉ 2.70
   oj-reuse.ParseReader  █████████████████▌ 2.51
         oj.ParseReader  ████████████▊ 1.84
        sen.ParseReader  ███████████▉ 1.70
        gen.ParseReder   ██████████  1.44
       json.Decode       ▓▓▓▓▓▓▓ 1.00

Parse chan interface{}
       json.Parse-chan          47625 ns/op    17790 B/op    335 allocs/op
         oj.Parse               34403 ns/op    18489 B/op    429 allocs/op
        gen.Parse               32320 ns/op    18487 B/op    429 allocs/op
        sen.Parse               35632 ns/op    18472 B/op    431 allocs/op

        gen.Parse        ██████████▎ 1.47
         oj.Parse        █████████▋ 1.38
        sen.Parse        █████████▎ 1.34
       json.Parse-chan   ▓▓▓▓▓▓▓ 1.00

Validate string/[]byte
       json.Valid               12056 ns/op        0 B/op      0 allocs/op
         oj.Valdate              3801 ns/op        0 B/op      0 allocs/op

         oj.Valdate      ██████████████████████▏ 3.17
       json.Valid        ▓▓▓▓▓▓▓ 1.00

Validate io.Reader
       json.Decode              72646 ns/op    32449 B/op    344 allocs/op
         oj.Valdate              7029 ns/op     4096 B/op      1 allocs/op

         oj.Valdate      ████████████████████████████████████████████████████████████████████████▎ 10.34
       json.Decode       ▓▓▓▓▓▓▓ 1.00

to JSON
       json.Marshal             48864 ns/op    17559 B/op    345 allocs/op
         oj.JSON                 6667 ns/op        0 B/op      0 allocs/op
        sen.SEN                  8167 ns/op        0 B/op      0 allocs/op

         oj.JSON         ███████████████████████████████████████████████████▎ 7.33
        sen.SEN          █████████████████████████████████████████▉ 5.98
       json.Marshal      ▓▓▓▓▓▓▓ 1.00

to JSON with indentation
       json.Marshal             78762 ns/op    26978 B/op    352 allocs/op
         oj.JSON                 7662 ns/op        0 B/op      0 allocs/op
        sen.Bytes                9053 ns/op        0 B/op      0 allocs/op
     pretty.JSON                62868 ns/op    36112 B/op    445 allocs/op
     pretty.SEN                 55533 ns/op    31160 B/op    396 allocs/op

         oj.JSON         ███████████████████████████████████████████████████████████████████████▉ 10.28
        sen.Bytes        ████████████████████████████████████████████████████████████▉ 8.70
     pretty.SEN          █████████▉ 1.42
     pretty.JSON         ████████▊ 1.25
       json.Marshal      ▓▓▓▓▓▓▓ 1.00

to JSON with indentation and sorted keys
         oj.JSON                13883 ns/op     2216 B/op     62 allocs/op
        sen.Bytes               15564 ns/op     2216 B/op     62 allocs/op
     pretty.JSON                85521 ns/op    36112 B/op    445 allocs/op
     pretty.SEN                 64236 ns/op    31160 B/op    396 allocs/op

         oj.JSON         ▓▓▓▓▓▓▓ 1.00
        sen.Bytes        ██████▏ 0.89
     pretty.SEN          █▌ 0.22
     pretty.JSON         █▏ 0.16

Write indented JSON
       json.Encode              86428 ns/op    28039 B/op    353 allocs/op
         oj.Write                7523 ns/op        0 B/op      0 allocs/op
        sen.Write                8950 ns/op        0 B/op      0 allocs/op
     pretty.WriteJSON           43611 ns/op    22544 B/op    441 allocs/op
     pretty.WriteSEN            47348 ns/op    19896 B/op    392 allocs/op

         oj.Write        ████████████████████████████████████████████████████████████████████████████████▍ 11.49
        sen.Write        ███████████████████████████████████████████████████████████████████▌ 9.66
     pretty.WriteJSON    █████████████▊ 1.98
     pretty.WriteSEN     ████████████▊ 1.83
       json.Encode       ▓▓▓▓▓▓▓ 1.00

Marshal Struct
       json.Marshal             11960 ns/op     3457 B/op      1 allocs/op
         oj.Marshal              8310 ns/op     1712 B/op     44 allocs/op

         oj.Marshal      ██████████  1.44
       json.Marshal      ▓▓▓▓▓▓▓ 1.00

Convert or Alter
        alt.Generify             3275 ns/op     1664 B/op     25 allocs/op
        alt.Alter                1695 ns/op      912 B/op     17 allocs/op

        alt.Alter        █████████████▌ 1.93
        alt.Generify     ▓▓▓▓▓▓▓ 1.00

JSONPath Get $..a[2].c
         jp.Get                239469 ns/op    19288 B/op   2227 allocs/op

         jp.Get          ▓▓▓▓▓▓▓ 1.00

JSONPath First  $..a[2].c
         jp.First               22625 ns/op     2880 B/op    233 allocs/op

         jp.First        ▓▓▓▓▓▓▓ 1.00

 Higher values (longer bars) are better in all cases. The bar graph compares the
 parsing performance. The lighter colored bar is the reference, usually the go
 json package.

 The Benchmarks reflect a use case where JSON is either provided as a string or
 read from a file (io.Reader) then parsed into simple go types of nil, bool, int64
 float64, string, []interface{}, or map[string]interface{}. When supported, an
 io.Writer benchmark is also included along with some miscellaneous operations.

Tests run on:
 OS:              Ubuntu 20.04.2 LTS
 Processor:       Intel(R) Core(TM) i7-8700 CPU
 Cores:           12
 Processor Speed: 3.20GHz
```
