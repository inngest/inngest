package libduckdb

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func TestDecodeStringTInlineAndPointer(t *testing.T) {
	var empty cStringT
	if got := string(decodeStringT(&empty)); got != "" {
		t.Fatalf("empty string = %q, want empty", got)
	}

	inline := cStringT{length: 12}
	copy(inline.data[:], "hello world!")
	if got := string(decodeStringT(&inline)); got != "hello world!" {
		t.Fatalf("inline string = %q, want %q", got, "hello world!")
	}

	backing := []byte("this is a much longer duckdb varchar payload")
	ptrStr := cStringT{length: uint32(len(backing))}
	copy(ptrStr.data[0:4], backing[:4])
	*(*unsafe.Pointer)(unsafe.Pointer(&ptrStr.data[4])) = unsafe.Pointer(&backing[0])

	got := string(decodeStringT(&ptrStr))
	if got != string(backing) {
		t.Fatalf("pointer string = %q, want %q", got, string(backing))
	}
}

func TestDecodeUUID(t *testing.T) {
	hi := uint64(0x0123456789abcdef)
	lo := uint64(0xfedcba9876543210)
	h := cHugeInt{
		lower: lo,
		upper: int64(hi ^ (uint64(1) << 63)),
	}
	got := decodeUUID(h)
	want := "01234567-89ab-cdef-fedc-ba9876543210"
	if got != want {
		t.Fatalf("decodeUUID = %q, want %q", got, want)
	}
}

func TestDecodeValidityBitmask(t *testing.T) {
	if !isValid(nil, 42) {
		t.Fatal("nil validity mask must treat all rows as valid")
	}
	words := []uint64{
		0b101,
		uint64(1) << 3,
	}
	if !isValid(&words[0], 0) {
		t.Fatal("row 0 should be valid")
	}
	if isValid(&words[0], 1) {
		t.Fatal("row 1 should be NULL")
	}
	if !isValid(&words[0], 2) {
		t.Fatal("row 2 should be valid")
	}
	if isValid(&words[0], 64) {
		t.Fatal("row 64 should be NULL")
	}
	if !isValid(&words[0], 67) {
		t.Fatal("row 67 should be valid")
	}
}

func TestDecodeTimestampAndDate(t *testing.T) {
	wantTS := time.Date(2026, 11, 23, 17, 30, 45, 123456000, time.UTC)
	if got := decodeTimestampMicros(wantTS.UnixMicro()); !got.Equal(wantTS) {
		t.Fatalf("decodeTimestampMicros = %v, want %v", got, wantTS)
	}
	wantDate := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	days := int32(wantDate.Unix() / 86400)
	if got := decodeDateDays(days); !got.Equal(wantDate) {
		t.Fatalf("decodeDateDays = %v, want %v", got, wantDate)
	}
}

func findRealLib(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("DUCKDB_LIB_PATH"); p != "" {
		return p
	}
	const fallback = "/tmp/duckdb-purego-test/libduckdb.dylib"
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	t.Skip("real libduckdb not available; set DUCKDB_LIB_PATH to run integration tests")
	return ""
}

func TestPuregoAllCoreTypes(t *testing.T) {
	libPath := findRealLib(t)
	lib, err := Load(libPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", libPath, err)
	}
	defer lib.Close()

	if v := lib.Version(); v == "" {
		t.Fatal("expected non-empty DuckDB library version")
	} else {
		t.Logf("loaded libduckdb version: %s", v)
	}

	db, err := lib.Open(":memory:", map[string]string{"threads": "2"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	conn, err := db.Connect()
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	sql := `SELECT
		NULL::INTEGER AS c_null,
		true AS c_bool,
		CAST(-42 AS TINYINT) AS c_i8,
		CAST(-1234 AS SMALLINT) AS c_i16,
		CAST(-123456 AS INTEGER) AS c_i32,
		CAST(-9007199254740993 AS BIGINT) AS c_i64,
		CAST(200 AS UTINYINT) AS c_u8,
		CAST(60000 AS USMALLINT) AS c_u16,
		CAST(3000000000 AS UINTEGER) AS c_u32,
		CAST(18000000000000000000 AS UBIGINT) AS c_u64,
		CAST(3.25 AS FLOAT) AS c_f32,
		CAST(3.141592653589793 AS DOUBLE) AS c_f64,
		'short' AS c_vshort,
		'this is a long varchar string exceeding twelve bytes inline limit' AS c_vlong,
		UUID '01234567-89ab-cdef-fedc-ba9876543210' AS c_uuid,
		TIMESTAMP '2026-11-23 17:30:45.123456' AS c_ts,
		TIMESTAMPTZ '2026-11-23 17:30:45.123456+00' AS c_tstz,
		DATE '2026-11-23' AS c_date,
		'\xDE\xAD\xBE\xEF'::BLOB AS c_blob`

	res, err := conn.Query(sql)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer res.Close()

	chunk, err := res.NextChunk()
	if err != nil {
		t.Fatalf("NextChunk: %v", err)
	}
	if len(chunk) != 1 {
		t.Fatalf("expected 1 row, got %d", len(chunk))
	}
	row := chunk[0]

	if row[0] != nil {
		t.Errorf("c_null = %#v, want nil", row[0])
	}
	if row[1] != true {
		t.Errorf("c_bool = %#v, want true", row[1])
	}
	if row[2] != int8(-42) {
		t.Errorf("c_i8 = %#v, want int8(-42)", row[2])
	}
	if row[3] != int16(-1234) {
		t.Errorf("c_i16 = %#v, want int16(-1234)", row[3])
	}
	if row[4] != int32(-123456) {
		t.Errorf("c_i32 = %#v, want int32(-123456)", row[4])
	}
	if row[5] != int64(-9007199254740993) {
		t.Errorf("c_i64 = %#v, want int64(-9007199254740993)", row[5])
	}
	if row[6] != uint8(200) {
		t.Errorf("c_u8 = %#v, want uint8(200)", row[6])
	}
	if row[7] != uint16(60000) {
		t.Errorf("c_u16 = %#v, want uint16(60000)", row[7])
	}
	if row[8] != uint32(3000000000) {
		t.Errorf("c_u32 = %#v, want uint32(3000000000)", row[8])
	}
	if row[9] != uint64(18000000000000000000) {
		t.Errorf("c_u64 = %#v, want uint64(18000000000000000000)", row[9])
	}
	if row[10] != float32(3.25) {
		t.Errorf("c_f32 = %#v, want float32(3.25)", row[10])
	}
	if row[11] != float64(3.141592653589793) {
		t.Errorf("c_f64 = %#v, want float64(3.141592653589793)", row[11])
	}
	if row[12] != "short" {
		t.Errorf("c_vshort = %#v, want %q", row[12], "short")
	}
	wantLong := "this is a long varchar string exceeding twelve bytes inline limit"
	if row[13] != wantLong {
		t.Errorf("c_vlong = %#v, want %q", row[13], wantLong)
	}
	if row[14] != "01234567-89ab-cdef-fedc-ba9876543210" {
		t.Errorf("c_uuid = %#v, want canonical UUID", row[14])
	}
	wantTS := time.Date(2026, 11, 23, 17, 30, 45, 123456000, time.UTC)
	if ts, ok := row[15].(time.Time); !ok || !ts.Equal(wantTS) {
		t.Errorf("c_ts = %#v, want %v", row[15], wantTS)
	}
	if tstz, ok := row[16].(time.Time); !ok || !tstz.Equal(wantTS) {
		t.Errorf("c_tstz = %#v, want %v", row[16], wantTS)
	}
	wantDate := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	if d, ok := row[17].(time.Time); !ok || !d.Equal(wantDate) {
		t.Errorf("c_date = %#v, want %v", row[17], wantDate)
	}
	wantBlob := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if b, ok := row[18].([]byte); !ok || !bytes.Equal(b, wantBlob) {
		t.Errorf("c_blob = %#v, want %x", row[18], wantBlob)
	}

	eof, err := res.NextChunk()
	if err != nil || eof != nil {
		t.Fatalf("expected EOF (nil, nil), got (%v, %v)", eof, err)
	}
}

func TestPuregoMultiChunkAndConcurrentConns(t *testing.T) {
	libPath := findRealLib(t)
	lib, err := Load(libPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer lib.Close()

	db, err := lib.Open(":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	const n = 4
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := db.Connect()
			if err != nil {
				errs <- err
				return
			}
			defer conn.Close()

			res, err := conn.Query("SELECT range AS i, CASE WHEN range % 3 = 0 THEN NULL ELSE 'row_' || range::VARCHAR END AS label FROM range(5000)")
			if err != nil {
				errs <- err
				return
			}
			defer res.Close()

			total := 0
			chunks := 0
			for {
				rows, err := res.NextChunk()
				if err != nil {
					errs <- err
					return
				}
				if rows == nil {
					break
				}
				chunks++
				total += len(rows)
			}
			if total != 5000 || chunks < 2 {
				errs <- fmt.Errorf("worker %d: got %d rows in %d chunks, want 5000 rows in >=2 chunks", id, total, chunks)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestPuregoInterrupt(t *testing.T) {
	libPath := findRealLib(t)
	lib, err := Load(libPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer lib.Close()

	db, err := lib.Open(":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	conn, err := db.Connect()
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		_, err := conn.Query("SELECT count(*) FROM range(100000000) a, range(1000) b")
		done <- err
	}()

	time.Sleep(100 * time.Millisecond)
	conn.Interrupt()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected interrupted query to return an error")
		}
		t.Logf("interrupted query returned expected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for duckdb_interrupt to abort query")
	}
}

func TestPuregoComplexTypesAndStreamTableFunction(t *testing.T) {
	libPath := findRealLib(t)
	lib, err := Load(libPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer lib.Close()

	db, err := lib.Open(":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	conn, err := db.Connect()
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	// Complex types: LIST, STRUCT, MAP, ARRAY, ENUM, UNION, VARIANT, INTERVAL, BIT, JSON, DECIMAL.
	if err := conn.Exec("CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy')"); err != nil {
		t.Fatalf("CREATE TYPE: %v", err)
	}
	res, err := conn.Query(`SELECT
		[1, 2, 3] AS c_list,
		{'a': 1, 'b': 'hi'} AS c_struct,
		MAP {'a': 1, 'b': 2} AS c_map,
		[1, 2, 3]::INTEGER[3] AS c_arr,
		'happy'::mood AS c_enum,
		union_value(a := 1) AS c_union,
		'{"x": 1, "y": [2, 3]}'::JSON::VARIANT AS c_var,
		INTERVAL '1 year 2 months 3 days 04:05:06.789' AS c_iv,
		BIT '101010' AS c_bit,
		'{"k":"v"}'::JSON AS c_json,
		123.45::DECIMAL(10,2) AS c_dec`)
	if err != nil {
		t.Fatalf("Query complex: %v", err)
	}
	cols := res.Columns()
	wantTypes := []string{
		"INTEGER[]", "STRUCT(a INTEGER, b VARCHAR)", "MAP(VARCHAR, INTEGER)",
		"INTEGER[3]", "ENUM('sad', 'ok', 'happy')", "UNION(a INTEGER)",
		"VARIANT", "INTERVAL", "BIT", "JSON", "DECIMAL(10,2)",
	}
	for i, wt := range wantTypes {
		if cols[i].TypeName != wt {
			t.Errorf("col %d TypeName = %q, want %q", i, cols[i].TypeName, wt)
		}
	}
	rows, err := res.NextChunk()
	res.Close()
	if err != nil || len(rows) != 1 {
		t.Fatalf("NextChunk complex: rows=%d err=%v", len(rows), err)
	}
	if rows[0][4] != "happy" {
		t.Errorf("enum = %#v", rows[0][4])
	}
	if rows[0][7] != "1 year 2 months 3 days 04:05:06.789" {
		t.Errorf("interval = %#v", rows[0][7])
	}
	if rows[0][8] != "101010" {
		t.Errorf("bit = %#v", rows[0][8])
	}

	// Stream table function test.
	if err := conn.RegisterStreamTableFunction(); err != nil {
		t.Fatalf("RegisterStreamTableFunction: %v", err)
	}
	st := RegisterStream("test-stream-1")
	defer UnregisterStream("test-stream-1")
	go func() {
		if err := st.Push(StreamChunk{
			RowCount: 2,
			Columns: []StreamColumn{
				{Kind: StreamColumnVarchar, Bytes: []byte("ab"), Ends: []uint32{1, 2}},
				{Kind: StreamColumnBigint, Int64s: []int64{10, 20}},
			},
		}); err != nil {
			t.Errorf("Push: %v", err)
		}
		st.Close()
	}()

	sres, err := conn.Query("SELECT c0, c1 FROM scan_data_from_quack_client('test-stream-1', NULL::STRUCT(c0 VARCHAR, c1 BIGINT), ordered := true)")
	if err != nil {
		t.Fatalf("Query stream: %v", err)
	}
	srows, err := sres.NextChunk()
	sres.Close()
	if err != nil {
		t.Fatalf("NextChunk stream: %v", err)
	}
	if len(srows) != 2 || srows[0][0] != "a" || srows[0][1] != int64(10) || srows[1][0] != "b" || srows[1][1] != int64(20) {
		t.Fatalf("unexpected stream rows: %#v", srows)
	}
}
