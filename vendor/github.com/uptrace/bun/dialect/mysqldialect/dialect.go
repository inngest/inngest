package mysqldialect

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/mod/semver"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

const datetimeType = "DATETIME"

func init() {
	if Version() != bun.Version() {
		panic(fmt.Errorf("mysqldialect and Bun must have the same version: v%s != v%s",
			Version(), bun.Version()))
	}
}

type Dialect struct {
	schema.BaseDialect

	tables   *schema.Tables
	features feature.Feature
	loc      *time.Location
}

func New(opts ...DialectOption) *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.AutoIncrement |
		feature.DefaultPlaceholder |
		feature.UpdateMultiTable |
		feature.ValuesRow |
		feature.TableTruncate |
		feature.TableNotExists |
		feature.InsertIgnore |
		feature.InsertOnDuplicateKey |
		feature.SelectExists |
		feature.CompositeIn |
		feature.FKDefaultOnAction |
		feature.UpdateOrderLimit |
		feature.DeleteOrderLimit

	for _, opt := range opts {
		opt(d)
	}

	return d
}

type DialectOption func(d *Dialect)

func WithTimeLocation(loc string) DialectOption {
	return func(d *Dialect) {
		location, err := time.LoadLocation(loc)
		if err != nil {
			panic(fmt.Errorf("mysqldialect can't load provided location %s: %s", loc, err))
		}
		d.loc = location
	}
}

func WithoutFeature(other feature.Feature) DialectOption {
	return func(d *Dialect) {
		d.features = d.features.Remove(other)
	}
}

func (d *Dialect) Init(db *sql.DB) {
	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
		log.Printf("can't discover MySQL version: %s", err)
		return
	}

	if strings.Contains(version, "MariaDB") {
		version = semver.MajorMinor("v" + cleanupVersion(version))
		d.features |= feature.CreateIndexIfNotExists
		if semver.Compare(version, "v10.0.5") >= 0 {
			d.features |= feature.DeleteReturning
		}
		if semver.Compare(version, "v10.5.0") >= 0 {
			d.features |= feature.InsertReturning
		}
		return
	}

	version = "v" + cleanupVersion(version)
	if semver.Compare(version, "v8.0") >= 0 {
		d.features |= feature.CTE | feature.WithValues
	}
	if semver.Compare(version, "v8.0.16") >= 0 {
		d.features |= feature.DeleteTableAlias
	}
}

func cleanupVersion(s string) string {
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return s[:i]
	}
	return s
}

func (d *Dialect) Name() dialect.Name {
	return dialect.MySQL
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) OnTable(table *schema.Table) {
	for _, field := range table.FieldMap {
		field.DiscoveredSQLType = sqlType(field)
	}
}

func (d *Dialect) IdentQuote() byte {
	return '`'
}

func (d *Dialect) AppendTime(b []byte, tm time.Time) []byte {
	b = append(b, '\'')
	if d.loc != nil {
		b = tm.In(d.loc).AppendFormat(b, "2006-01-02 15:04:05.999999")
	} else {
		b = tm.AppendFormat(b, "2006-01-02 15:04:05.999999")
	}
	b = append(b, '\'')
	return b
}

func (*Dialect) AppendString(b []byte, s string) []byte {
	b = append(b, '\'')
loop:
	for _, r := range s {
		switch r {
		case '\000':
			continue loop
		case '\'':
			b = append(b, "''"...)
			continue loop
		case '\\':
			b = append(b, '\\', '\\')
			continue loop
		}

		if r < utf8.RuneSelf {
			b = append(b, byte(r))
			continue
		}

		l := len(b)
		if cap(b)-l < utf8.UTFMax {
			b = append(b, make([]byte, utf8.UTFMax)...)
		}
		n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
		b = b[:l+n]
	}
	b = append(b, '\'')
	return b
}

func (*Dialect) AppendBytes(b []byte, bs []byte) []byte {
	if bs == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, `X'`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

	b = append(b, '\'')

	return b
}

func (*Dialect) AppendJSON(b, jsonb []byte) []byte {
	b = append(b, '\'')

	for _, c := range jsonb {
		switch c {
		case '\'':
			b = append(b, "''"...)
		case '\\':
			b = append(b, `\\`...)
		default:
			b = append(b, c)
		}
	}

	b = append(b, '\'')

	return b
}

func (d *Dialect) DefaultVarcharLen() int {
	return 255
}

func (d *Dialect) AppendSequence(b []byte, _ *schema.Table, _ *schema.Field) []byte {
	return append(b, " AUTO_INCREMENT"...)
}

func (d *Dialect) DefaultSchema() string {
	return "mydb"
}

func sqlType(field *schema.Field) string {
	if field.DiscoveredSQLType == sqltype.Timestamp {
		return datetimeType
	}
	return field.DiscoveredSQLType
}
