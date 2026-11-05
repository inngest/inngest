package schema

import (
	"database/sql/driver"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/extra/bunjson"
	"github.com/uptrace/bun/internal"
	"github.com/vmihailenco/msgpack/v5"
)

type (
	AppenderFunc   func(gen QueryGen, b []byte, v reflect.Value) []byte
	CustomAppender func(typ reflect.Type) AppenderFunc
)

var appenders = []AppenderFunc{
	reflect.Bool:          AppendBoolValue,
	reflect.Int:           AppendIntValue,
	reflect.Int8:          AppendIntValue,
	reflect.Int16:         AppendIntValue,
	reflect.Int32:         AppendIntValue,
	reflect.Int64:         AppendIntValue,
	reflect.Uint:          AppendUintValue,
	reflect.Uint8:         AppendUintValue,
	reflect.Uint16:        AppendUintValue,
	reflect.Uint32:        appendUint32Value,
	reflect.Uint64:        appendUint64Value,
	reflect.Uintptr:       nil,
	reflect.Float32:       AppendFloat32Value,
	reflect.Float64:       AppendFloat64Value,
	reflect.Complex64:     nil,
	reflect.Complex128:    nil,
	reflect.Array:         AppendJSONValue,
	reflect.Chan:          nil,
	reflect.Func:          nil,
	reflect.Interface:     nil,
	reflect.Map:           AppendJSONValue,
	reflect.Ptr:           nil,
	reflect.Slice:         AppendJSONValue,
	reflect.String:        AppendStringValue,
	reflect.Struct:        AppendJSONValue,
	reflect.UnsafePointer: nil,
}

var appenderCache = xsync.NewMapOf[reflect.Type, AppenderFunc]()

func FieldAppender(dialect Dialect, field *Field) AppenderFunc {
	if field.Tag.HasOption("msgpack") {
		return appendMsgpack
	}

	fieldType := field.StructField.Type

	switch strings.ToUpper(field.UserSQLType) {
	case sqltype.JSON, sqltype.JSONB:
		if fieldType.Implements(driverValuerType) {
			return appendDriverValue
		}

		if fieldType.Kind() != reflect.Ptr {
			if reflect.PointerTo(fieldType).Implements(driverValuerType) {
				return addrAppender(appendDriverValue)
			}
		}

		return AppendJSONValue
	}

	return Appender(dialect, fieldType)
}

func Appender(dialect Dialect, typ reflect.Type) AppenderFunc {
	if v, ok := appenderCache.Load(typ); ok {
		return v
	}

	fn := appender(dialect, typ)

	if v, ok := appenderCache.LoadOrStore(typ, fn); ok {
		return v
	}
	return fn
}

func appender(dialect Dialect, typ reflect.Type) AppenderFunc {
	switch typ {
	case bytesType:
		return appendBytesValue
	case timeType:
		return appendTimeValue
	case timePtrType:
		return PtrAppender(appendTimeValue)
	case ipNetType:
		return appendIPNetValue
	case ipType, netipPrefixType, netipAddrType:
		return appendStringer
	case jsonRawMessageType:
		return appendJSONRawMessageValue
	}

	kind := typ.Kind()

	if typ.Implements(queryAppenderType) {
		if kind == reflect.Ptr {
			return nilAwareAppender(appendQueryAppenderValue)
		}
		return appendQueryAppenderValue
	}
	if typ.Implements(driverValuerType) {
		if kind == reflect.Ptr {
			return nilAwareAppender(appendDriverValue)
		}
		return appendDriverValue
	}

	if kind != reflect.Ptr {
		ptr := reflect.PointerTo(typ)
		if ptr.Implements(queryAppenderType) {
			return addrAppender(appendQueryAppenderValue)
		}
		if ptr.Implements(driverValuerType) {
			return addrAppender(appendDriverValue)
		}
	}

	switch kind {
	case reflect.Interface:
		return ifaceAppenderFunc
	case reflect.Ptr:
		if typ.Implements(jsonMarshalerType) {
			return nilAwareAppender(AppendJSONValue)
		}
		if fn := Appender(dialect, typ.Elem()); fn != nil {
			return PtrAppender(fn)
		}
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return appendBytesValue
		}
	case reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return appendArrayBytesValue
		}
	}

	return appenders[typ.Kind()]
}

func ifaceAppenderFunc(gen QueryGen, b []byte, v reflect.Value) []byte {
	if v.IsNil() {
		return dialect.AppendNull(b)
	}
	elem := v.Elem()
	appender := Appender(gen.Dialect(), elem.Type())
	return appender(gen, b, elem)
}

func nilAwareAppender(fn AppenderFunc) AppenderFunc {
	return func(gen QueryGen, b []byte, v reflect.Value) []byte {
		if v.IsNil() {
			return dialect.AppendNull(b)
		}
		return fn(gen, b, v)
	}
}

func PtrAppender(fn AppenderFunc) AppenderFunc {
	return func(gen QueryGen, b []byte, v reflect.Value) []byte {
		if v.IsNil() {
			return dialect.AppendNull(b)
		}
		return fn(gen, b, v.Elem())
	}
}

func AppendBoolValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	return gen.Dialect().AppendBool(b, v.Bool())
}

func AppendIntValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	return strconv.AppendInt(b, v.Int(), 10)
}

func AppendUintValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	return strconv.AppendUint(b, v.Uint(), 10)
}

func appendUint32Value(gen QueryGen, b []byte, v reflect.Value) []byte {
	return gen.Dialect().AppendUint32(b, uint32(v.Uint()))
}

func appendUint64Value(gen QueryGen, b []byte, v reflect.Value) []byte {
	return gen.Dialect().AppendUint64(b, v.Uint())
}

func AppendFloat32Value(gen QueryGen, b []byte, v reflect.Value) []byte {
	return dialect.AppendFloat32(b, float32(v.Float()))
}

func AppendFloat64Value(gen QueryGen, b []byte, v reflect.Value) []byte {
	return dialect.AppendFloat64(b, float64(v.Float()))
}

func appendBytesValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	return gen.Dialect().AppendBytes(b, v.Bytes())
}

func appendArrayBytesValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	if v.CanAddr() {
		return gen.Dialect().AppendBytes(b, v.Slice(0, v.Len()).Bytes())
	}

	tmp := make([]byte, v.Len())
	reflect.Copy(reflect.ValueOf(tmp), v)
	b = gen.Dialect().AppendBytes(b, tmp)
	return b
}

func AppendStringValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	return gen.Dialect().AppendString(b, v.String())
}

func AppendJSONValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	bb, err := bunjson.Marshal(v.Interface())
	if err != nil {
		return dialect.AppendError(b, err)
	}

	if len(bb) > 0 && bb[len(bb)-1] == '\n' {
		bb = bb[:len(bb)-1]
	}

	return gen.Dialect().AppendJSON(b, bb)
}

func appendTimeValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	tm := v.Interface().(time.Time)
	return gen.Dialect().AppendTime(b, tm)
}

func appendIPNetValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	ipnet := v.Interface().(net.IPNet)
	return gen.Dialect().AppendString(b, ipnet.String())
}

func appendStringer(gen QueryGen, b []byte, v reflect.Value) []byte {
	return gen.Dialect().AppendString(b, v.Interface().(fmt.Stringer).String())
}

func appendJSONRawMessageValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	bytes := v.Bytes()
	if bytes == nil {
		return dialect.AppendNull(b)
	}
	return gen.Dialect().AppendString(b, internal.String(bytes))
}

func appendQueryAppenderValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	return AppendQueryAppender(gen, b, v.Interface().(QueryAppender))
}

func appendDriverValue(gen QueryGen, b []byte, v reflect.Value) []byte {
	value, err := v.Interface().(driver.Valuer).Value()
	if err != nil {
		return dialect.AppendError(b, err)
	}
	if _, ok := value.(driver.Valuer); ok {
		return dialect.AppendError(b, fmt.Errorf("driver.Valuer returns unsupported type %T", value))
	}
	return gen.Append(b, value)
}

func addrAppender(fn AppenderFunc) AppenderFunc {
	return func(gen QueryGen, b []byte, v reflect.Value) []byte {
		if !v.CanAddr() {
			err := fmt.Errorf("bun: Append(nonaddressable %T)", v.Interface())
			return dialect.AppendError(b, err)
		}
		return fn(gen, b, v.Addr())
	}
}

func appendMsgpack(gen QueryGen, b []byte, v reflect.Value) []byte {
	hexEnc := internal.NewHexEncoder(b)

	enc := msgpack.GetEncoder()
	defer msgpack.PutEncoder(enc)

	enc.Reset(hexEnc)
	if err := enc.EncodeValue(v); err != nil {
		return dialect.AppendError(b, err)
	}

	if err := hexEnc.Close(); err != nil {
		return dialect.AppendError(b, err)
	}

	return hexEnc.Bytes()
}

func AppendQueryAppender(gen QueryGen, b []byte, app QueryAppender) []byte {
	bb, err := app.AppendQuery(gen, b)
	if err != nil {
		return dialect.AppendError(b, err)
	}
	return bb
}
