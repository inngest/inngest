<!--
 - Licensed to the Apache Software Foundation (ASF) under one or more
 - contributor license agreements.  See the NOTICE file distributed with
 - this work for additional information regarding copyright ownership.
 - The ASF licenses this file to You under the Apache License, Version 2.0
 - (the "License"); you may not use this file except in compliance with
 - the License.  You may obtain a copy of the License at
 -
 -   http://www.apache.org/licenses/LICENSE-2.0
 -
 - Unless required by applicable law or agreed to in writing, software
 - distributed under the License is distributed on an "AS IS" BASIS,
 - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 - See the License for the specific language governing permissions and
 - limitations under the License.
 -->

# Catalog Implementations

## Integration Testing

The Catalog implementations can be manually tested using the CLI implemented
in the `cmd/iceberg` folder.

### REST Catalog

To test the REST catalog implementation, we have a docker configuration
for RustFS and the Apache Iceberg REST fixture.

You can spin up the local catalog from the repository root by running
`make integration-setup`. You can then follow the steps of the Iceberg
[Quickstart](https://iceberg.apache.org/spark-quickstart/#creating-a-table) tutorial, which we've summarized below.

#### Setup your Iceberg catalog

First launch a pyspark console by running:

```bash
docker exec -it spark-iceberg pyspark
```

Once in the pyspark shell, we create a simple table with a namespace of 
"demo.nyc" called "taxis":

```python
from pyspark.sql.types import DoubleType, FloatType, LongType, StructType,StructField, StringType
schema = StructType([
  StructField("vendor_id", LongType(), True),
  StructField("trip_id", LongType(), True),
  StructField("trip_distance", FloatType(), True),
  StructField("fare_amount", DoubleType(), True),
  StructField("store_and_fwd_flag", StringType(), True)
])

df = spark.createDataFrame([], schema)
df.writeTo("demo.nyc.taxis").create()
```

Finally, we write another data-frame to the table to add new files:

```python
schema = spark.table("demo.nyc.taxis").schema
data = [
    (1, 1000371, 1.8, 15.32, "N"),
    (2, 1000372, 2.5, 22.15, "N"),
    (2, 1000373, 0.9, 9.01, "N"),
    (1, 1000374, 8.4, 42.13, "Y")
  ]
df = spark.createDataFrame(data, schema)
df.writeTo("demo.nyc.taxis").append()
```

#### Testing with the CLI

Now that we have a table in the catalog which is running. You can use the 
CLI which is implemented in the `cmd/iceberg` folder. You will need to set
the following environment variables (which can also be found in the 
docker-compose.yml):

```
AWS_S3_ENDPOINT=http://localhost:9000
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=admin
AWS_SECRET_ACCESS_KEY=password
```

With those environment variables set you can now run the CLI:

```bash
$ go run ./cmd/iceberg list --catalog rest --uri http://localhost:8181
┌──────┐
| IDs  |
| ---- |
| demo |
└──────┘
```

You can retrieve the schema of the table:

```bash
$ go run ./cmd/iceberg schema --catalog rest --uri http://localhost:8181 demo.nyc.taxis
Current Schema, id=0
├──1: vendor_id: optional long
├──2: trip_id: optional long
├──3: trip_distance: optional float
├──4: fare_amount: optional double
└──5: store_and_fwd_flag: optional string
```

You can get the file list:

```bash
$ go run ./cmd/iceberg files --catalog rest --uri http://localhost:8181 demo.nyc.taxis
Snapshots: rest.demo.nyc.taxis
└─┬Snapshot 7004656639550124164, schema 0: s3://warehouse/demo/nyc/taxis/metadata/snap-7004656639550124164-1-0d533cd4-f0c1-45a6-a691-f2be3abe5491.avro
  └─┬Manifest: s3://warehouse/demo/nyc/taxis/metadata/0d533cd4-f0c1-45a6-a691-f2be3abe5491-m0.avro
    ├──Datafile: s3://warehouse/demo/nyc/taxis/data/00004-24-244255d4-8bf6-41bd-8885-bf7d2136fddf-00001.parquet
    ├──Datafile: s3://warehouse/demo/nyc/taxis/data/00009-29-244255d4-8bf6-41bd-8885-bf7d2136fddf-00001.parquet
    ├──Datafile: s3://warehouse/demo/nyc/taxis/data/00014-34-244255d4-8bf6-41bd-8885-bf7d2136fddf-00001.parquet
    └──Datafile: s3://warehouse/demo/nyc/taxis/data/00019-39-244255d4-8bf6-41bd-8885-bf7d2136fddf-00001.parquet
```

and so on, for the various options available in the CLI.
