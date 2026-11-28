package schema

import (
	"database/sql/driver"
	"encoding/json"
	"net"
	"net/netip"
	"reflect"
	"time"
)

var (
	bytesType          = reflect.TypeFor[[]byte]()
	timePtrType        = reflect.TypeFor[*time.Time]()
	timeType           = reflect.TypeFor[time.Time]()
	ipType             = reflect.TypeFor[net.IP]()
	ipNetType          = reflect.TypeFor[net.IPNet]()
	netipPrefixType    = reflect.TypeFor[netip.Prefix]()
	netipAddrType      = reflect.TypeFor[netip.Addr]()
	jsonRawMessageType = reflect.TypeFor[json.RawMessage]()

	driverValuerType  = reflect.TypeFor[driver.Valuer]()
	queryAppenderType = reflect.TypeFor[QueryAppender]()
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
)

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func fieldByIndex(v reflect.Value, index []int) (_ reflect.Value, ok bool) {
	if len(index) == 1 {
		return v.Field(index[0]), true
	}

	for i, idx := range index {
		if i > 0 {
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return v, false
				}
				v = v.Elem()
			}
		}
		v = v.Field(idx)
	}
	return v, true
}
