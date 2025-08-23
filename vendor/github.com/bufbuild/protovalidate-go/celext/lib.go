// Copyright 2023-2024 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package celext

import (
	"bytes"
	"math"
	"net"
	"net/mail"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/ext"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// DefaultEnv produces a cel.Env with the necessary cel.EnvOption and
// cel.ProgramOption values preconfigured for usage throughout the
// module. If useUTC is true, timestamp operations use the UTC timezone instead
// of the local timezone.
func DefaultEnv(useUTC bool) (*cel.Env, error) {
	return cel.NewEnv(
		// we bind in the global type registry optimistically to ensure expressions
		// operating against Any WKTs can resolve their underlying type if it's
		// known to the application. They will otherwise fail with a runtime error
		// if the type is unknown.
		cel.TypeDescs(protoregistry.GlobalFiles),
		cel.Lib(Lib{
			useUTC: useUTC,
		}),
	)
}

// RequiredCELEnvOptions returns the options required to have expressions which
// rely on the provided descriptor.
func RequiredCELEnvOptions(fieldDesc protoreflect.FieldDescriptor) []cel.EnvOption {
	if fieldDesc.IsMap() {
		return append(
			RequiredCELEnvOptions(fieldDesc.MapKey()),
			RequiredCELEnvOptions(fieldDesc.MapValue())...,
		)
	}
	if fieldDesc.Kind() == protoreflect.MessageKind ||
		fieldDesc.Kind() == protoreflect.GroupKind {
		return []cel.EnvOption{
			cel.Types(dynamicpb.NewMessage(fieldDesc.Message())),
		}
	}
	return nil
}

// Lib is the collection of functions and settings required by protovalidate
// beyond the standard definitions of the CEL Specification:
//
//	https://github.com/google/cel-spec/blob/master/doc/langdef.md#list-of-standard-definitions
//
// All implementations of protovalidate MUST implement these functions and
// should avoid exposing additional functions as they will not be portable.
type Lib struct {
	useUTC bool
}

//nolint:funlen
func (l Lib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.DefaultUTCTimeZone(l.useUTC),
		cel.CrossTypeNumericComparisons(true),
		cel.EagerlyValidateDeclarations(true),
		// TODO: reduce this to just the functionality we want to support
		ext.Strings(ext.StringsValidateFormatCalls(true)),
		cel.Variable("now", cel.TimestampType),
		cel.Function("unique",
			l.uniqueMemberOverload(cel.BoolType, l.uniqueScalar),
			l.uniqueMemberOverload(cel.IntType, l.uniqueScalar),
			l.uniqueMemberOverload(cel.UintType, l.uniqueScalar),
			l.uniqueMemberOverload(cel.DoubleType, l.uniqueScalar),
			l.uniqueMemberOverload(cel.StringType, l.uniqueScalar),
			l.uniqueMemberOverload(cel.BytesType, l.uniqueBytes),
		),
		cel.Function("isNan",
			cel.MemberOverload(
				"double_is_nan_bool",
				[]*cel.Type{cel.DoubleType},
				cel.BoolType,
				cel.UnaryBinding(func(value ref.Val) ref.Val {
					num, ok := value.Value().(float64)
					if !ok {
						return types.UnsupportedRefValConversionErr(value)
					}
					return types.Bool(math.IsNaN(num))
				}),
			),
		),
		cel.Function("isInf",
			cel.MemberOverload(
				"double_is_inf_bool",
				[]*cel.Type{cel.DoubleType},
				cel.BoolType,
				cel.UnaryBinding(func(value ref.Val) ref.Val {
					num, ok := value.Value().(float64)
					if !ok {
						return types.UnsupportedRefValConversionErr(value)
					}
					return types.Bool(math.IsInf(num, 0))
				}),
			),
			cel.MemberOverload(
				"double_int_is_inf_bool",
				[]*cel.Type{cel.DoubleType, cel.IntType},
				cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					num, ok := lhs.Value().(float64)
					if !ok {
						return types.UnsupportedRefValConversionErr(lhs)
					}
					sign, ok := rhs.Value().(int64)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(math.IsInf(num, int(sign)))
				}),
			),
		),
		cel.Function("isHostname",
			cel.MemberOverload(
				"string_is_hostname_bool",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					host, ok := args[0].Value().(string)
					if !ok {
						return types.Bool(false)
					}
					return types.Bool(l.validateHostname(host))
				}),
			),
		),
		cel.Function("isEmail",
			cel.MemberOverload(
				"string_is_email_bool",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					addr, ok := args[0].Value().(string)
					if !ok {
						return types.Bool(false)
					}
					return types.Bool(l.validateEmail(addr))
				}),
			),
		),
		cel.Function("isIp",
			cel.MemberOverload(
				"string_is_ip_bool",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					addr, ok := args[0].Value().(string)
					if !ok {
						return types.Bool(false)
					}
					return types.Bool(l.validateIP(addr, 0))
				}),
			),
			cel.MemberOverload(
				"string_int_is_ip_bool",
				[]*cel.Type{cel.StringType, cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					addr, aok := args[0].Value().(string)
					vers, vok := args[1].Value().(int64)
					if !aok || !vok {
						return types.Bool(false)
					}
					return types.Bool(l.validateIP(addr, vers))
				})),
		),
		cel.Function("isIpPrefix",
			cel.MemberOverload(
				"string_is_ip_prefix_bool",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					prefix, ok := args[0].Value().(string)
					if !ok {
						return types.Bool(false)
					}
					return types.Bool(l.validateIPPrefix(prefix, 0, false))
				})),
			cel.MemberOverload(
				"string_int_is_ip_prefix_bool",
				[]*cel.Type{cel.StringType, cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					prefix, pok := args[0].Value().(string)
					vers, vok := args[1].Value().(int64)
					if !pok || !vok {
						return types.Bool(false)
					}
					return types.Bool(l.validateIPPrefix(prefix, vers, false))
				})),
			cel.MemberOverload(
				"string_bool_is_ip_prefix_bool",
				[]*cel.Type{cel.StringType, cel.BoolType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					prefix, pok := args[0].Value().(string)
					strict, sok := args[1].Value().(bool)
					if !pok || !sok {
						return types.Bool(false)
					}
					return types.Bool(l.validateIPPrefix(prefix, 0, strict))
				})),
			cel.MemberOverload(
				"string_int_bool_is_ip_prefix_bool",
				[]*cel.Type{cel.StringType, cel.IntType, cel.BoolType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					prefix, pok := args[0].Value().(string)
					vers, vok := args[1].Value().(int64)
					strict, sok := args[2].Value().(bool)
					if !pok || !vok || !sok {
						return types.Bool(false)
					}
					return types.Bool(l.validateIPPrefix(prefix, vers, strict))
				})),
		),
		cel.Function("isUri",
			cel.MemberOverload(
				"string_is_uri_bool",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					s, ok := args[0].Value().(string)
					if !ok {
						return types.Bool(false)
					}
					uri, err := url.Parse(s)
					return types.Bool(err == nil && uri.IsAbs())
				}),
			),
		),
		cel.Function("isUriRef",
			cel.MemberOverload(
				"string_is_uri_ref_bool",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					s, ok := args[0].Value().(string)
					if !ok {
						return types.Bool(false)
					}
					_, err := url.Parse(s)
					return types.Bool(err == nil)
				}),
			),
		),
		cel.Function(overloads.Contains,
			cel.MemberOverload(
				overloads.ContainsString, []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					substr, ok := rhs.Value().(string)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(strings.Contains(lhs.Value().(string), substr))
				}),
			),
			cel.MemberOverload("contains_bytes", []*cel.Type{cel.BytesType, cel.BytesType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					substr, ok := rhs.Value().([]byte)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(bytes.Contains(lhs.Value().([]byte), substr))
				}),
			),
		),
		cel.Function(overloads.EndsWith,
			cel.MemberOverload(
				overloads.EndsWithString, []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					suffix, ok := rhs.Value().(string)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(strings.HasSuffix(lhs.Value().(string), suffix))
				}),
			),
			cel.MemberOverload("ends_with_bytes", []*cel.Type{cel.BytesType, cel.BytesType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					suffix, ok := rhs.Value().([]byte)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(bytes.HasSuffix(lhs.Value().([]byte), suffix))
				}),
			),
		),
		cel.Function(overloads.StartsWith,
			cel.MemberOverload(
				overloads.StartsWithString, []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					prefix, ok := rhs.Value().(string)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(strings.HasPrefix(lhs.Value().(string), prefix))
				}),
			),
			cel.MemberOverload("starts_with_bytes", []*cel.Type{cel.BytesType, cel.BytesType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					prefix, ok := rhs.Value().([]byte)
					if !ok {
						return types.UnsupportedRefValConversionErr(rhs)
					}
					return types.Bool(bytes.HasPrefix(lhs.Value().([]byte), prefix))
				}),
			),
		),
		cel.Function("isHostAndPort",
			cel.MemberOverload("string_bool_is_host_and_port_bool",
				[]*cel.Type{cel.StringType, cel.BoolType}, cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					val, vok := lhs.Value().(string)
					portReq, pok := rhs.Value().(bool)
					if !vok || !pok {
						return types.Bool(false)
					}
					return types.Bool(l.isHostAndPort(val, portReq))
				}),
			),
		),
	}
}

func (l Lib) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{
		cel.EvalOptions(
			cel.OptOptimize,
		),
	}
}

func (l Lib) uniqueMemberOverload(itemType *cel.Type, overload func(lister traits.Lister) ref.Val) cel.FunctionOpt {
	return cel.MemberOverload(
		itemType.String()+"_unique_bool",
		[]*cel.Type{cel.ListType(itemType)},
		cel.BoolType,
		cel.UnaryBinding(func(value ref.Val) ref.Val {
			list, ok := value.(traits.Lister)
			if !ok {
				return types.UnsupportedRefValConversionErr(value)
			}
			return overload(list)
		}),
	)
}

func (l Lib) uniqueScalar(list traits.Lister) ref.Val {
	size, ok := list.Size().Value().(int64)
	if !ok {
		return types.UnsupportedRefValConversionErr(list.Size().Value())
	}
	if size <= 1 {
		return types.Bool(true)
	}
	exist := make(map[ref.Val]struct{}, size)
	for i := int64(0); i < size; i++ {
		val := list.Get(types.Int(i))
		if _, ok := exist[val]; ok {
			return types.Bool(false)
		}
		exist[val] = struct{}{}
	}
	return types.Bool(true)
}

// uniqueBytes is an overload implementation of the unique function that
// compares bytes type CEL values. This function is used instead of uniqueScalar
// as the bytes ([]uint8) type is not hashable in Go; we cheat this by converting
// the value to a string.
func (l Lib) uniqueBytes(list traits.Lister) ref.Val {
	size, ok := list.Size().Value().(int64)
	if !ok {
		return types.UnsupportedRefValConversionErr(list.Size().Value())
	}
	if size <= 1 {
		return types.Bool(true)
	}
	exist := make(map[any]struct{}, size)
	for i := int64(0); i < size; i++ {
		val := list.Get(types.Int(i)).Value()
		if b, ok := val.([]uint8); ok {
			val = string(b)
		}
		if _, ok := exist[val]; ok {
			return types.Bool(false)
		}
		exist[val] = struct{}{}
	}
	return types.Bool(true)
}

func (l Lib) validateEmail(addr string) bool {
	a, err := mail.ParseAddress(addr)
	if err != nil || strings.ContainsRune(addr, '<') || a.Address != addr {
		return false
	}

	addr = a.Address
	if len(addr) > 254 {
		return false
	}

	parts := strings.SplitN(addr, "@", 2)
	return len(parts[0]) <= 64 && l.validateHostname(parts[1])
}

func (l Lib) validateHostname(host string) bool {
	if len(host) > 253 {
		return false
	}

	s := strings.ToLower(strings.TrimSuffix(host, "."))
	allDigits := false
	// split hostname on '.' and validate each part
	for _, part := range strings.Split(s, ".") {
		allDigits = true
		// if part is empty, longer than 63 chars, or starts/ends with '-', it is invalid
		if l := len(part); l == 0 || l > 63 || part[0] == '-' || part[l-1] == '-' {
			return false
		}
		// for each character in part
		for _, ch := range part {
			// if the character is not a-z, 0-9, or '-', it is invalid
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return false
			}
			allDigits = allDigits && ch >= '0' && ch <= '9'
		}
	}

	// the last part cannot be all numbers
	return !allDigits
}

func (l Lib) validateIP(addr string, ver int64) bool {
	address := net.ParseIP(addr)
	if address == nil {
		return false
	}
	switch ver {
	case 0:
		return true
	case 4:
		return address.To4() != nil
	case 6:
		return address.To4() == nil
	default:
		return false
	}
}

func (l Lib) validateIPPrefix(p string, ver int64, strict bool) bool {
	prefix, err := netip.ParsePrefix(p)
	if err != nil {
		return false
	}
	if strict && (prefix.Addr() != prefix.Masked().Addr()) {
		return false
	}
	switch ver {
	case 0:
		return true
	case 4:
		return prefix.Addr().Is4()
	case 6:
		return prefix.Addr().Is6()
	default:
		return false
	}
}

func (l Lib) isHostAndPort(val string, portRequired bool) bool {
	if len(val) == 0 {
		return false
	}

	splitIdx := strings.LastIndexByte(val, ':')
	if val[0] == '[' { // ipv6
		end := strings.IndexByte(val, ']')
		switch end + 1 {
		case len(val): // no port
			return !portRequired && l.validateIP(val[1:end], 6)
		case splitIdx: // port
			return l.validateIP(val[1:end], 6) &&
				l.validatePort(val[splitIdx+1:])
		default: // malformed
			return false
		}
	}

	if splitIdx < 0 {
		return !portRequired &&
			(l.validateHostname(val) ||
				l.validateIP(val, 4))
	}

	host, port := val[:splitIdx], val[splitIdx+1:]
	return (l.validateHostname(host) ||
		l.validateIP(host, 4)) &&
		l.validatePort(port)
}

func (l Lib) validatePort(val string) bool {
	n, err := strconv.ParseUint(val, 10, 32)
	return err == nil && n <= 65535
}
