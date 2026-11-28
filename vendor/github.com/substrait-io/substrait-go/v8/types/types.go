// SPDX-License-Identifier: Apache-2.0

//lint:file-ignore SA1019 Using a deprecated function, variable, constant or field

package types

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	substraitgo "github.com/substrait-io/substrait-go/v8"
	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

type Version = proto.Version

type Nullability = proto.Type_Nullability

const (
	NullabilityUnspecified = proto.Type_NULLABILITY_UNSPECIFIED
	NullabilityNullable    = proto.Type_NULLABILITY_NULLABLE
	NullabilityRequired    = proto.Type_NULLABILITY_REQUIRED
)

type TypeName string

const (
	TypeNameI8               TypeName = "i8"
	TypeNameI16              TypeName = "i16"
	TypeNameI32              TypeName = "i32"
	TypeNameI64              TypeName = "i64"
	TypeNameFp32             TypeName = "fp32"
	TypeNameFp64             TypeName = "fp64"
	TypeNameString           TypeName = "string"
	TypeNameBinary           TypeName = "binary"
	TypeNameBoolean          TypeName = "boolean"
	TypeNameDate             TypeName = "date"
	TypeNameTime             TypeName = "time"
	TypeNameTimestamp        TypeName = "timestamp"
	TypeNameTimestampTz      TypeName = "timestamp_tz"
	TypeNameIntervalYear     TypeName = "interval_year"
	TypeNameIntervalDay      TypeName = "interval_day"
	TypeNameIntervalCompound TypeName = "interval_compound"
	TypeNameUUID             TypeName = "uuid"
	TypeNameUDT              TypeName = "u!"

	TypeNameFixedBinary          TypeName = "fixedbinary"
	TypeNameFixedChar            TypeName = "fixedchar"
	TypeNameVarChar              TypeName = "varchar"
	TypeNameDecimal              TypeName = "decimal"
	TypeNamePrecisionTime        TypeName = "precision_time"
	TypeNamePrecisionTimestamp   TypeName = "precision_timestamp"
	TypeNamePrecisionTimestampTz TypeName = "precision_timestamp_tz"
)

var simpleTypeNameMap = map[TypeName]Type{
	TypeNameI8:           &Int8Type{},
	TypeNameI16:          &Int16Type{},
	TypeNameI32:          &Int32Type{},
	TypeNameI64:          &Int64Type{},
	TypeNameFp32:         &Float32Type{},
	TypeNameFp64:         &Float64Type{},
	TypeNameString:       &StringType{},
	TypeNameBinary:       &BinaryType{},
	TypeNameBoolean:      &BooleanType{},
	TypeNameDate:         &DateType{},
	TypeNameTime:         &TimeType{},
	TypeNameTimestamp:    &TimestampType{},
	TypeNameTimestampTz:  &TimestampTzType{},
	TypeNameIntervalYear: &IntervalYearType{},
	TypeNameUUID:         &UUIDType{},
	TypeNameUDT:          &UserDefinedType{},
}

var fixedTypeNameMap = map[TypeName]FixedType{
	TypeNameFixedBinary: &FixedBinaryType{},
	TypeNameFixedChar:   &FixedCharType{},
	TypeNameVarChar:     &VarCharType{},
}

var shortTypeNames = map[TypeName]string{
	TypeNameString:           "str",
	TypeNameBinary:           "vbin",
	TypeNameBoolean:          "bool",
	TypeNameTimestamp:        "ts",
	TypeNameTimestampTz:      "tstz",
	TypeNameIntervalYear:     "iyear",
	TypeNameIntervalDay:      "iday",
	TypeNameIntervalCompound: "icompound",

	TypeNameFixedBinary: "fbin",
	TypeNameFixedChar:   "fchar",
	TypeNameVarChar:     "vchar",

	TypeNameDecimal:              "dec",
	TypeNamePrecisionTime:        "pt",
	TypeNamePrecisionTimestamp:   "pts",
	TypeNamePrecisionTimestampTz: "ptstz",
}

func GetShortTypeName(name TypeName) string {
	if n, ok := shortTypeNames[name]; ok {
		return n
	}
	return string(name)
}

func SimpleTypeNameToType(name TypeName) (Type, error) {
	if t, ok := simpleTypeNameMap[name]; ok {
		return t, nil
	}
	return nil, substraitgo.ErrNotFound
}

func FixedTypeNameToType(name TypeName) (FixedType, error) {
	if t, ok := fixedTypeNameMap[name]; ok {
		return t, nil
	}
	return nil, substraitgo.ErrInvalidType
}

func GetTypeNameToTypeMap() map[string]Type {
	typeMap := make(map[string]Type)
	for k, v := range simpleTypeNameMap {
		typeMap[string(k)] = v
	}
	for k, v := range fixedTypeNameMap {
		typeMap[string(k)] = v
	}
	typeMap[string(TypeNameDecimal)] = &DecimalType{}
	typeMap[string(TypeNameIntervalDay)] = &IntervalDayType{}
	typeMap[string(TypeNamePrecisionTime)] = &PrecisionTimeType{}
	typeMap[string(TypeNamePrecisionTimestamp)] = &PrecisionTimestampType{}
	typeMap[string(TypeNamePrecisionTimestampTz)] = &PrecisionTimestampTzType{}
	return typeMap
}

type AggregationPhase = proto.AggregationPhase

const (
	AggPhaseUnspecified                = proto.AggregationPhase_AGGREGATION_PHASE_UNSPECIFIED
	AggPhaseInitialToIntermediate      = proto.AggregationPhase_AGGREGATION_PHASE_INITIAL_TO_INTERMEDIATE
	AggPhaseIntermediateToIntermediate = proto.AggregationPhase_AGGREGATION_PHASE_INTERMEDIATE_TO_INTERMEDIATE
	AggPhaseInitialToResult            = proto.AggregationPhase_AGGREGATION_PHASE_INITIAL_TO_RESULT
	AggPhaseIntermediateToResult       = proto.AggregationPhase_AGGREGATION_PHASE_INTERMEDIATE_TO_RESULT
)

type AggregationInvocation = proto.AggregateFunction_AggregationInvocation

const (
	AggInvocationUnspecified = proto.AggregateFunction_AGGREGATION_INVOCATION_UNSPECIFIED
	AggInvocationAll         = proto.AggregateFunction_AGGREGATION_INVOCATION_ALL
	AggInvocationDistinct    = proto.AggregateFunction_AGGREGATION_INVOCATION_DISTINCT
)

type BoundsType = proto.Expression_WindowFunction_BoundsType

const (
	BoundsTypeUnspecified = proto.Expression_WindowFunction_BOUNDS_TYPE_UNSPECIFIED
	BoundsTypeRows        = proto.Expression_WindowFunction_BOUNDS_TYPE_ROWS
	BoundsTypeRange       = proto.Expression_WindowFunction_BOUNDS_TYPE_RANGE
)

type SortDirection proto.SortField_SortDirection

const (
	SortUnspecified    = SortDirection(proto.SortField_SORT_DIRECTION_UNSPECIFIED)
	SortAscNullsFirst  = SortDirection(proto.SortField_SORT_DIRECTION_ASC_NULLS_FIRST)
	SortAscNullsLast   = SortDirection(proto.SortField_SORT_DIRECTION_ASC_NULLS_LAST)
	SortDescNullsFirst = SortDirection(proto.SortField_SORT_DIRECTION_DESC_NULLS_FIRST)
	SortDescNullsLast  = SortDirection(proto.SortField_SORT_DIRECTION_DESC_NULLS_LAST)
	SortClustered      = SortDirection(proto.SortField_SORT_DIRECTION_CLUSTERED)
)

func (s SortDirection) String() string { return proto.SortField_SortDirection(s).String() }

func (SortDirection) isSortKind() {}

type FunctionRef uint32

func (f FunctionRef) String() string { return "comparison_func_ref: " + strconv.Itoa(int(f)) }

func (FunctionRef) isSortKind() {}

type CastFailBehavior = proto.Expression_Cast_FailureBehavior

const (
	BehaviorUnspecified    = proto.Expression_Cast_FAILURE_BEHAVIOR_UNSPECIFIED
	BehaviorReturnNil      = proto.Expression_Cast_FAILURE_BEHAVIOR_RETURN_NULL
	BehaviorThrowException = proto.Expression_Cast_FAILURE_BEHAVIOR_THROW_EXCEPTION
)

type (
	IntervalYearToMonth  = proto.Expression_Literal_IntervalYearToMonth
	IntervalDayToSecond  = proto.Expression_Literal_IntervalDayToSecond
	VarChar              = proto.Expression_Literal_VarChar
	Decimal              = proto.Expression_Literal_Decimal
	UserDefinedLiteral   = proto.Expression_Literal_UserDefined
	PrecisionTime        = proto.Expression_Literal_PrecisionTime
	PrecisionTimestamp   = proto.Expression_Literal_PrecisionTimestamp_
	PrecisionTimestampTz = proto.Expression_Literal_PrecisionTimestampTz
)

// TypeFromProto returns the appropriate Type object from a protobuf
// type message.
func TypeFromProto(t *proto.Type) Type {
	switch t := t.Kind.(type) {
	case *proto.Type_Bool:
		return &BooleanType{
			Nullability:      t.Bool.Nullability,
			TypeVariationRef: t.Bool.TypeVariationReference,
		}
	case *proto.Type_I8_:
		return &Int8Type{
			Nullability:      t.I8.Nullability,
			TypeVariationRef: t.I8.TypeVariationReference,
		}
	case *proto.Type_I16_:
		return &Int16Type{
			Nullability:      t.I16.Nullability,
			TypeVariationRef: t.I16.TypeVariationReference,
		}
	case *proto.Type_I32_:
		return &Int32Type{
			Nullability:      t.I32.Nullability,
			TypeVariationRef: t.I32.TypeVariationReference,
		}
	case *proto.Type_I64_:
		return &Int64Type{
			Nullability:      t.I64.Nullability,
			TypeVariationRef: t.I64.TypeVariationReference,
		}
	case *proto.Type_Fp32:
		return &Float32Type{
			Nullability:      t.Fp32.Nullability,
			TypeVariationRef: t.Fp32.TypeVariationReference,
		}
	case *proto.Type_Fp64:
		return &Float64Type{
			Nullability:      t.Fp64.Nullability,
			TypeVariationRef: t.Fp64.TypeVariationReference,
		}
	case *proto.Type_String_:
		return &StringType{
			Nullability:      t.String_.Nullability,
			TypeVariationRef: t.String_.TypeVariationReference,
		}
	case *proto.Type_Binary_:
		return &BinaryType{
			Nullability:      t.Binary.Nullability,
			TypeVariationRef: t.Binary.TypeVariationReference,
		}
	case *proto.Type_Timestamp_:
		return &TimestampType{
			Nullability:      t.Timestamp.Nullability,
			TypeVariationRef: t.Timestamp.TypeVariationReference,
		}
	case *proto.Type_Date_:
		return &DateType{
			Nullability:      t.Date.Nullability,
			TypeVariationRef: t.Date.TypeVariationReference,
		}
	case *proto.Type_Time_:
		return &TimeType{
			Nullability:      t.Time.Nullability,
			TypeVariationRef: t.Time.TypeVariationReference,
		}
	case *proto.Type_IntervalYear_:
		return &IntervalYearType{
			Nullability:      t.IntervalYear.Nullability,
			TypeVariationRef: t.IntervalYear.TypeVariationReference,
		}
	case *proto.Type_IntervalDay_:
		var precision = PrecisionMicroSeconds
		if t.IntervalDay.Precision != nil {
			var err error
			precision, err = ProtoToTimePrecision(*t.IntervalDay.Precision)
			if err != nil {
				panic(fmt.Sprintf("Invalid precision %v", err))
			}
		}
		return &IntervalDayType{
			Nullability:      t.IntervalDay.Nullability,
			TypeVariationRef: t.IntervalDay.TypeVariationReference,
			Precision:        precision,
		}
	case *proto.Type_IntervalCompound_:
		precision, err := ProtoToTimePrecision(t.IntervalCompound.Precision)
		if err != nil {
			panic(fmt.Sprintf("Invalid precision %v", err))
		}
		return &IntervalCompoundType{
			nullability:      t.IntervalCompound.Nullability,
			typeVariationRef: t.IntervalCompound.TypeVariationReference,
			precision:        precision,
		}
	case *proto.Type_TimestampTz:
		return &TimestampTzType{
			Nullability:      t.TimestampTz.Nullability,
			TypeVariationRef: t.TimestampTz.TypeVariationReference,
		}
	case *proto.Type_Uuid:
		return &UUIDType{
			Nullability:      t.Uuid.Nullability,
			TypeVariationRef: t.Uuid.TypeVariationReference,
		}
	case *proto.Type_FixedBinary_:
		return &FixedBinaryType{
			Nullability:      t.FixedBinary.Nullability,
			TypeVariationRef: t.FixedBinary.TypeVariationReference,
			Length:           t.FixedBinary.Length,
		}
	case *proto.Type_FixedChar_:
		return &FixedCharType{
			Nullability:      t.FixedChar.Nullability,
			TypeVariationRef: t.FixedChar.TypeVariationReference,
			Length:           t.FixedChar.Length,
		}
	case *proto.Type_Varchar:
		return &VarCharType{
			Nullability:      t.Varchar.Nullability,
			TypeVariationRef: t.Varchar.TypeVariationReference,
			Length:           t.Varchar.Length,
		}
	case *proto.Type_Decimal_:
		return &DecimalType{
			Nullability:      t.Decimal.Nullability,
			TypeVariationRef: t.Decimal.TypeVariationReference,
			Scale:            t.Decimal.Scale,
			Precision:        t.Decimal.Precision,
		}
	case *proto.Type_PrecisionTime_:
		precision, err := ProtoToTimePrecision(t.PrecisionTime.Precision)
		if err != nil {
			panic(fmt.Sprintf("Invalid precision %v", err))
		}
		return &PrecisionTimeType{
			Nullability:      t.PrecisionTime.Nullability,
			TypeVariationRef: t.PrecisionTime.TypeVariationReference,
			Precision:        precision,
		}
	case *proto.Type_PrecisionTimestamp_:
		precision, err := ProtoToTimePrecision(t.PrecisionTimestamp.Precision)
		if err != nil {
			panic(fmt.Sprintf("Invalid precision %v", err))
		}
		return &PrecisionTimestampType{
			Nullability:      t.PrecisionTimestamp.Nullability,
			TypeVariationRef: t.PrecisionTimestamp.TypeVariationReference,
			Precision:        precision,
		}
	case *proto.Type_PrecisionTimestampTz:
		precision, err := ProtoToTimePrecision(t.PrecisionTimestampTz.Precision)
		if err != nil {
			panic(fmt.Sprintf("Invalid precision %v", err))
		}
		return &PrecisionTimestampTzType{PrecisionTimestampType{
			Nullability:      t.PrecisionTimestampTz.Nullability,
			TypeVariationRef: t.PrecisionTimestampTz.TypeVariationReference,
			Precision:        precision,
		}}
	case *proto.Type_Struct_:
		fields := make([]Type, len(t.Struct.Types))
		for i, f := range t.Struct.Types {
			fields[i] = TypeFromProto(f)
		}
		return &StructType{
			Nullability:      t.Struct.Nullability,
			TypeVariationRef: t.Struct.TypeVariationReference,
			Types:            fields,
		}
	case *proto.Type_Func_:
		params := make([]Type, len(t.Func.ParameterTypes))
		for i, p := range t.Func.ParameterTypes {
			params[i] = TypeFromProto(p)
		}
		return &FuncType{
			Nullability:    t.Func.Nullability,
			ParameterTypes: params,
			ReturnType:     TypeFromProto(t.Func.ReturnType),
		}
	case *proto.Type_List_:
		return &ListType{
			Nullability:      t.List.Nullability,
			TypeVariationRef: t.List.TypeVariationReference,
			Type:             TypeFromProto(t.List.Type),
		}
	case *proto.Type_Map_:
		return &MapType{
			Nullability:      t.Map.Nullability,
			TypeVariationRef: t.Map.TypeVariationReference,
			Key:              TypeFromProto(t.Map.Key),
			Value:            TypeFromProto(t.Map.Value),
		}
	case *proto.Type_UserDefined_:
		params := make([]TypeParam, len(t.UserDefined.TypeParameters))
		for i, p := range t.UserDefined.TypeParameters {
			params[i] = TypeParamFromProto(p)
		}
		return &UserDefinedType{
			Nullability:      t.UserDefined.Nullability,
			TypeVariationRef: t.UserDefined.TypeVariationReference,
			TypeReference:    t.UserDefined.TypeReference,
			TypeParameters:   params,
		}
	}
	panic("unimplemented type from proto")
}

type (
	Date        int32
	FixedChar   string
	Time        int64
	Timestamp   int64
	TimestampTz int64
	FixedBinary []byte
	UUID        []byte
	Enum        string

	FunctionOption = proto.FunctionOption

	// FuncArg corresponds to the protobuf FunctionArgument. Anything
	// which could be a function argument should meet this interface.
	// This is either an Expression, a Type, or an Enum (string).
	// These are the actual arguments for a function present in a plan.
	FuncArg interface {
		fmt.Stringer
		ToProtoFuncArg() *proto.FunctionArgument
	}

	SortKind interface {
		isSortKind()
		fmt.Stringer
	}

	// Type corresponds to the proto.Type message and represents
	// a specific concrete type. These are types which can be present in plan (are serializable)
	Type interface {
		FuncArg
		isRootRef()
		fmt.Stringer
		ShortString() string
		GetType() Type
		GetNullability() Nullability
		GetTypeVariationReference() uint32
		Equals(Type) bool
		// WithNullability returns a copy of this type but with
		// the nullability set to the passed in value
		WithNullability(Nullability) Type
		// GetParameters returns all parameters of this type and will be used in function return type derivation
		GetParameters() []interface{}
	}

	TimeConverter interface {
		// ToTime converts the current value into a time.Time assuming microsecond precision.
		ToTime() time.Time
		// ToPrecisionTime converts the current value using the provided precision into a time.Time.
		ToPrecisionTime(precision TimePrecision) time.Time
	}

	TimePrinter interface {
		// ToTimeString returns a human consumable version of the current value.
		ToTimeString() string
	}

	IsoTimePrinter interface {
		// ToIsoTimeString returns a human consumable version of the current value in ISO8601 format.
		ToIsoTimeString() string
	}

	IsoValuePrinter interface {
		// IsoValueString API returns the value in ISO8601 format. This is used in normalizing the function testcases
		IsoValueString() string
	}

	// CompositeType this represents a concrete type having components
	CompositeType interface {
		Type
		// ParameterString this returns parameter string
		// for e.g. parameter decimal<P, S>, ParameterString returns "P,S"
		ParameterString() string
		// BaseString this returns long name for parameter string
		// for e.g. parameter decimal<P, S>, BaseString returns "decimal"
		BaseString() string
	}

	// FuncDefArgType represents a type used in a function argument
	// This is an unresolved type of an argument present in the function definition.
	// This type can't be present in plan (not serializable)
	FuncDefArgType interface {
		fmt.Stringer
		//SetNullability set nullability as given argument
		SetNullability(Nullability) FuncDefArgType
		// HasParameterizedParam returns true if the type has at least one parameterized parameters
		// if all parameters are concrete then it returns false
		HasParameterizedParam() bool
		// GetParameterizedParams returns all parameterized parameters
		// it doesn't return concrete parameters
		GetParameterizedParams() []interface{}

		// MatchWithNullability This API return true if Type argument
		// is compatible with this param otherwise it returns false.
		// This method expects that nullability of argument is same as this type.
		MatchWithNullability(ot Type) bool
		// MatchWithoutNullability This API return true if Type argument
		// is compatible with this param otherwise it returns false.
		// This method ignores nullability for matching.
		MatchWithoutNullability(ot Type) bool
		ShortString() string
		GetNullability() Nullability

		// ReturnType resolves this unresolved type to a concrete Type.
		// Called when this type is used as a function return type (or nested within one).
		//
		// funcParameters are the abstract parameter types from the function definition.
		// argumentTypes are the concrete types passed to the function.
		ReturnType(funcParameters []FuncDefArgType, argumentTypes []Type) (Type, error)

		// WithParameters returns a new instance of this type with the given parameters.
		// This is used in function return type derivation
		WithParameters([]interface{}) (Type, error)
	}

	FixedType interface {
		CompositeType
		WithLength(int32) FixedType
		GetLength() int32
	}

	timestampPrecisionType interface {
		CompositeType
		GetPrecision() TimePrecision
	}
)

var CommonEnumType = &EnumType{}

// EnumType represents an enumeration function parameter.
// It supports a fixed set of declared string values as constant arguments.
type EnumType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Name             string
	Options          []string
}

func (e *EnumType) ToProtoFuncArg() *proto.FunctionArgument {
	// FIXME no proto for enum yet
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: TypeToProto(e)},
	}
}

func (e *EnumType) isRootRef() {}

func (e *EnumType) GetType() Type {
	return e
}

func (e *EnumType) GetTypeVariationReference() uint32 {
	return e.TypeVariationRef
}

func (e *EnumType) Equals(t Type) bool {
	return e.MatchWithNullability(t)
}

func (e *EnumType) WithNullability(n Nullability) Type {
	out := *e
	out.Nullability = n
	return &out
}

func (e *EnumType) GetParameters() []interface{} {
	return []interface{}{}
}

func (e *EnumType) String() string {
	return e.Name
}

func (e *EnumType) SetNullability(n Nullability) FuncDefArgType {
	e.Nullability = n
	return e
}

func (e *EnumType) HasParameterizedParam() bool {
	return false
}

func (e *EnumType) GetParameterizedParams() []interface{} {
	return nil
}

func (e *EnumType) MatchWithNullability(ot Type) bool {
	if e.Nullability != ot.GetNullability() {
		return false
	}
	return e.MatchWithoutNullability(ot)
}

func (e *EnumType) MatchWithoutNullability(ot Type) bool {
	if ot == CommonEnumType {
		return true
	}
	if odt, ok := ot.(*EnumType); ok {
		if e.Name != odt.Name {
			return false
		}
		if len(e.Options) != len(odt.Options) {
			return false
		}
		for i, v := range e.Options {
			if v != odt.Options[i] {
				return false
			}
		}
		return true
	}
	return false
}

func (e *EnumType) ShortString() string {
	// Enum args use "req" as their compound function name key, matching the
	// substrait spec's EnumTypeString constant in extensions/simple_extension.go.
	return "req"
}

func (e *EnumType) GetNullability() Nullability {
	return e.Nullability
}

func (e *EnumType) ReturnType(funcParameters []FuncDefArgType, argumentTypes []Type) (Type, error) {
	return e, nil
}

func (e *EnumType) WithParameters(params []interface{}) (Type, error) {
	panic("EnumType.WithParameters not implemented")
}

// TypeToProto properly constructs the appropriate protobuf message
// for the given type.
func TypeToProto(t Type) *proto.Type {
	switch t := t.(type) {
	case *BooleanType:
		return &proto.Type{Kind: &proto.Type_Bool{
			Bool: &proto.Type_Boolean{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *Int8Type:
		return &proto.Type{Kind: &proto.Type_I8_{
			I8: &proto.Type_I8{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *Int16Type:
		return &proto.Type{Kind: &proto.Type_I16_{
			I16: &proto.Type_I16{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *Int32Type:
		return &proto.Type{Kind: &proto.Type_I32_{
			I32: &proto.Type_I32{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *Int64Type:
		return &proto.Type{Kind: &proto.Type_I64_{
			I64: &proto.Type_I64{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *Float32Type:
		return &proto.Type{Kind: &proto.Type_Fp32{
			Fp32: &proto.Type_FP32{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *Float64Type:
		return &proto.Type{Kind: &proto.Type_Fp64{
			Fp64: &proto.Type_FP64{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *StringType:
		return &proto.Type{Kind: &proto.Type_String_{
			String_: &proto.Type_String{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *BinaryType:
		return &proto.Type{Kind: &proto.Type_Binary_{
			Binary: &proto.Type_Binary{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *DateType:
		return &proto.Type{Kind: &proto.Type_Date_{
			Date: &proto.Type_Date{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *TimeType:
		return &proto.Type{Kind: &proto.Type_Time_{
			Time: &proto.Type_Time{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *TimestampTzType:
		return &proto.Type{Kind: &proto.Type_TimestampTz{
			TimestampTz: &proto.Type_TimestampTZ{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *TimestampType:
		return &proto.Type{Kind: &proto.Type_Timestamp_{
			Timestamp: &proto.Type_Timestamp{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *IntervalYearType:
		return &proto.Type{Kind: &proto.Type_IntervalYear_{
			IntervalYear: &proto.Type_IntervalYear{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *IntervalDayType:
		precision := t.Precision.ToProtoVal()
		return &proto.Type{Kind: &proto.Type_IntervalDay_{
			IntervalDay: &proto.Type_IntervalDay{
				Precision:              &precision,
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case IntervalCompoundType:
		precision := t.precision.ToProtoVal()
		return &proto.Type{Kind: &proto.Type_IntervalCompound_{
			IntervalCompound: &proto.Type_IntervalCompound{
				Precision:              precision,
				Nullability:            t.nullability,
				TypeVariationReference: t.typeVariationRef}}}
	case *UUIDType:
		return &proto.Type{Kind: &proto.Type_Uuid{
			Uuid: &proto.Type_UUID{
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *FixedCharType:
		return &proto.Type{Kind: &proto.Type_FixedChar_{
			FixedChar: &proto.Type_FixedChar{
				Length:                 t.Length,
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *VarCharType:
		return &proto.Type{Kind: &proto.Type_Varchar{
			Varchar: &proto.Type_VarChar{
				Length:                 t.Length,
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *FixedBinaryType:
		return &proto.Type{Kind: &proto.Type_FixedBinary_{
			FixedBinary: &proto.Type_FixedBinary{
				Length:                 t.Length,
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *DecimalType:
		return t.ToProto()
	case *PrecisionTimeType:
		return &proto.Type{Kind: &proto.Type_PrecisionTime_{
			PrecisionTime: &proto.Type_PrecisionTime{
				Precision:              int32(t.Precision),
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *PrecisionTimestampType:
		return &proto.Type{Kind: &proto.Type_PrecisionTimestamp_{
			PrecisionTimestamp: &proto.Type_PrecisionTimestamp{
				Precision:              int32(t.Precision),
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *PrecisionTimestampTzType:
		return &proto.Type{Kind: &proto.Type_PrecisionTimestampTz{
			PrecisionTimestampTz: &proto.Type_PrecisionTimestampTZ{
				Precision:              int32(t.Precision),
				Nullability:            t.Nullability,
				TypeVariationReference: t.TypeVariationRef}}}
	case *StructType:
		return t.ToProto()
	case *FuncType:
		return t.ToProto()
	case *ListType:
		return t.ToProto()
	case *MapType:
		return t.ToProto()
	case *UserDefinedType:
		return t.ToProto()
	}
	panic("unimplemented type")
}

type primitiveTypeIFace interface {
	bool | int8 | int16 | ~int32 | ~int64 |
		float32 | float64 | ~string |
		[]byte | IntervalYearToMonth | IntervalDayToSecond | UUID
}

var emptyFixedChar FixedChar

var typeNames = map[reflect.Type]string{
	reflect.PointerTo(reflect.TypeOf(true)):           "boolean",
	reflect.PointerTo(reflect.TypeOf(int8(0))):        "i8",
	reflect.PointerTo(reflect.TypeOf(int16(0))):       "i16",
	reflect.PointerTo(reflect.TypeOf(int32(0))):       "i32",
	reflect.PointerTo(reflect.TypeOf(int64(0))):       "i64",
	reflect.PointerTo(reflect.TypeOf(float32(0))):     "fp32",
	reflect.PointerTo(reflect.TypeOf(float64(0))):     "fp64",
	reflect.PointerTo(reflect.TypeOf([]byte{})):       "binary",
	reflect.PointerTo(reflect.TypeOf("")):             "string",
	reflect.PointerTo(reflect.TypeOf(Timestamp(0))):   "timestamp",
	reflect.PointerTo(reflect.TypeOf(Date(0))):        "date",
	reflect.PointerTo(reflect.TypeOf(Time(0))):        "time",
	reflect.PointerTo(reflect.TypeOf(TimestampTz(0))): "timestamp_tz",
	reflect.PointerTo(reflect.TypeOf(UUID{})):         "uuid",
	reflect.TypeOf(&IntervalYearToMonth{}):            "interval_year",
	reflect.TypeOf(&IntervalDayToSecond{}):            "interval_day",
	reflect.TypeOf(&FixedBinary{}):                    "fixedbinary",
	reflect.TypeOf(&emptyFixedChar):                   "fixedchar",
	reflect.TypeOf(&VarChar{}):                        "varchar",
	reflect.TypeOf(&PrecisionTimeType{}):              "precision_time",
	reflect.TypeOf(&PrecisionTimestampType{}):         "precision_timestamp",
	reflect.TypeOf(&PrecisionTimestampTzType{}):       "precision_timestamp_tz",
}

var shortNames = map[reflect.Type]string{
	reflect.PointerTo(reflect.TypeOf(true)):           "bool",
	reflect.PointerTo(reflect.TypeOf(int8(0))):        "i8",
	reflect.PointerTo(reflect.TypeOf(int16(0))):       "i16",
	reflect.PointerTo(reflect.TypeOf(int32(0))):       "i32",
	reflect.PointerTo(reflect.TypeOf(int64(0))):       "i64",
	reflect.PointerTo(reflect.TypeOf(float32(0))):     "fp32",
	reflect.PointerTo(reflect.TypeOf(float64(0))):     "fp64",
	reflect.PointerTo(reflect.TypeOf([]byte{})):       "vbin",
	reflect.PointerTo(reflect.TypeOf("")):             "str",
	reflect.PointerTo(reflect.TypeOf(Timestamp(0))):   "ts",
	reflect.PointerTo(reflect.TypeOf(Date(0))):        "date",
	reflect.PointerTo(reflect.TypeOf(Time(0))):        "time",
	reflect.PointerTo(reflect.TypeOf(TimestampTz(0))): "tstz",
	reflect.PointerTo(reflect.TypeOf(UUID{})):         "uuid",
	reflect.TypeOf(&IntervalYearToMonth{}):            "iyear",
	reflect.TypeOf(&IntervalDayToSecond{}):            "iday",
	reflect.TypeOf(&FixedBinary{}):                    "fbin",
	reflect.TypeOf(&emptyFixedChar):                   "fchar",
	reflect.TypeOf(&VarChar{}):                        "vchar",
}

func strNullable(t Type) string {
	return strFromNullability(t.GetNullability())
}

func strFromNullability(nullability Nullability) string {
	if nullability == NullabilityNullable {
		return "?"
	}
	return ""
}

// PrimitiveType is a generic implementation of simple primitive types
// which only need to track if they are nullable and if they are a type
// variation.
type PrimitiveType[T primitiveTypeIFace] struct {
	Nullability      Nullability
	TypeVariationRef uint32
}

func (*PrimitiveType[T]) isRootRef() {}
func (s *PrimitiveType[T]) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *PrimitiveType[T]) GetParameters() []interface{} {
	return []interface{}{}
}

func (s *PrimitiveType[T]) GetType() Type                     { return s }
func (s *PrimitiveType[T]) GetNullability() Nullability       { return s.Nullability }
func (s *PrimitiveType[T]) GetTypeVariationReference() uint32 { return s.TypeVariationRef }
func (s *PrimitiveType[T]) Equals(rhs Type) bool {
	if o, ok := rhs.(*PrimitiveType[T]); ok {
		return *o == *s
	}

	return false
}

func (s *PrimitiveType[T]) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: TypeToProto(s)},
	}
}

func (*PrimitiveType[T]) ShortString() string {
	var z *T
	if n, ok := shortNames[reflect.TypeOf(z)]; ok {
		return n
	}
	return reflect.TypeOf(z).Elem().Name()
}

func (s *PrimitiveType[T]) String() string {
	var z *T
	if n, ok := typeNames[reflect.TypeOf(z)]; ok {
		return n + strNullable(s)
	}
	return reflect.TypeOf(z).Elem().Name() + strNullable(s)
}

func (s *PrimitiveType[T]) HasParameterizedParam() bool {
	// primitive type doesn't have abstract parameters
	return false
}

func (s *PrimitiveType[T]) GetParameterizedParams() []interface{} {
	// primitive type doesn't have any abstract parameters
	return nil
}

func (s *PrimitiveType[T]) SetNullability(n Nullability) FuncDefArgType {
	s.Nullability = n
	return s
}

func (s *PrimitiveType[T]) MatchWithNullability(ot Type) bool {
	if s.Nullability != ot.GetNullability() {
		return false
	}
	return s.MatchWithoutNullability(ot)
}

func (s *PrimitiveType[T]) MatchWithoutNullability(ot Type) bool {
	if _, ok := ot.(*PrimitiveType[T]); ok {
		return true
	}
	return false
}

func (s *PrimitiveType[T]) ReturnType([]FuncDefArgType, []Type) (Type, error) {
	return s, nil
}

func (s *PrimitiveType[T]) WithParameters([]interface{}) (Type, error) {
	return s, nil
}

// create type aliases to the generic structs
//
// The parameterized types below use parameterizedTypeSingleIntegerParam to
// create types that accept a single integer parameter (e.g., precision)
// Other parameterized types (e.g. DecimalType) have their
// own dedicated type definitions (ParameterizedDecimalType) rather than using the generic parameterized type.
type (
	BooleanType                           = PrimitiveType[bool]
	Int8Type                              = PrimitiveType[int8]
	Int16Type                             = PrimitiveType[int16]
	Int32Type                             = PrimitiveType[int32]
	Int64Type                             = PrimitiveType[int64]
	Float32Type                           = PrimitiveType[float32]
	Float64Type                           = PrimitiveType[float64]
	StringType                            = PrimitiveType[string]
	BinaryType                            = PrimitiveType[[]byte]
	TimestampType                         = PrimitiveType[Timestamp]
	DateType                              = PrimitiveType[Date]
	TimeType                              = PrimitiveType[Time]
	TimestampTzType                       = PrimitiveType[TimestampTz]
	IntervalYearType                      = PrimitiveType[IntervalYearToMonth]
	UUIDType                              = PrimitiveType[UUID]
	FixedCharType                         = FixedLenType[FixedChar]
	VarCharType                           = FixedLenType[VarChar]
	FixedBinaryType                       = FixedLenType[FixedBinary]
	ParameterizedVarCharType              = parameterizedTypeSingleIntegerParam[*VarCharType]
	ParameterizedFixedCharType            = parameterizedTypeSingleIntegerParam[*FixedCharType]
	ParameterizedFixedBinaryType          = parameterizedTypeSingleIntegerParam[*FixedBinaryType]
	ParameterizedPrecisionTimeType        = parameterizedTypeSingleIntegerParam[*PrecisionTimeType]
	ParameterizedPrecisionTimestampType   = parameterizedTypeSingleIntegerParam[*PrecisionTimestampType]
	ParameterizedPrecisionTimestampTzType = parameterizedTypeSingleIntegerParam[*PrecisionTimestampTzType]
	ParameterizedIntervalDayType          = parameterizedTypeSingleIntegerParam[*IntervalDayType]
)

// FixedLenType is any of the types which also need to track their specific
// length as they have a fixed length.
type FixedLenType[T FixedChar | VarChar | FixedBinary] struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Length           int32
}

func (*FixedLenType[T]) isRootRef() {}
func (s *FixedLenType[T]) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *FixedLenType[T]) GetParameters() []interface{} {
	return []interface{}{int64(s.Length)}
}

func (s *FixedLenType[T]) GetType() Type                     { return s }
func (s *FixedLenType[T]) GetNullability() Nullability       { return s.Nullability }
func (s *FixedLenType[T]) GetTypeVariationReference() uint32 { return s.TypeVariationRef }
func (s *FixedLenType[T]) Equals(rhs Type) bool {
	if o, ok := rhs.(*FixedLenType[T]); ok {
		return *o == *s
	}

	return false
}

func (s *FixedLenType[T]) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: TypeToProto(s)},
	}
}

func (*FixedLenType[T]) ShortString() string {
	var z *T
	return shortNames[reflect.TypeOf(z)]
}

func (s *FixedLenType[T]) String() string {
	var z *T
	return fmt.Sprintf("%s%s<%d>",
		typeNames[reflect.TypeOf(z)], strNullable(s), s.Length)
}

func (s *FixedLenType[T]) ParameterString() string {
	return fmt.Sprintf("%d", s.Length)
}

func (s *FixedLenType[T]) BaseString() string {
	var z *T
	return typeNames[reflect.TypeOf(z)]
}

func (s *FixedLenType[T]) WithLength(length int32) FixedType {
	out := *s
	out.Length = length
	return &out
}

func (s *FixedLenType[T]) GetLength() int32 {
	return s.Length
}

func (s *FixedLenType[T]) GetReturnType(length int32, nullability Nullability) Type {
	out := *s
	out.Length = length
	out.Nullability = nullability
	return &out
}

// DecimalType is a decimal type with concrete precision and scale parameters, e.g. Decimal(10, 2).
type DecimalType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Scale, Precision int32
}

func (*DecimalType) isRootRef() {}
func (s *DecimalType) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *DecimalType) GetParameters() []interface{} {
	return []interface{}{int64(s.Precision), int64(s.Scale)}
}

func (s *DecimalType) GetType() Type                     { return s }
func (s *DecimalType) GetNullability() Nullability       { return s.Nullability }
func (s *DecimalType) GetTypeVariationReference() uint32 { return s.TypeVariationRef }
func (s *DecimalType) Equals(rhs Type) bool {
	if o, ok := rhs.(*DecimalType); ok {
		return *o == *s
	}

	return false
}

func (s *DecimalType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: s.ToProto()},
	}
}

func (s *DecimalType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_Decimal_{
		Decimal: &proto.Type_Decimal{
			Scale: s.Scale, Precision: s.Precision,
			Nullability:            s.Nullability,
			TypeVariationReference: s.TypeVariationRef}}}
}

func (*DecimalType) ShortString() string { return "dec" }
func (s *DecimalType) String() string {
	return fmt.Sprintf("decimal%s<%d,%d>", strNullable(s),
		s.Precision, s.Scale)
}

func (s *DecimalType) ParameterString() string {
	return fmt.Sprintf("%d,%d", s.Precision, s.Scale)
}

func (*DecimalType) BaseString() string {
	return "decimal"
}

type StructType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Types            []Type
}

func (*StructType) isRootRef() {}
func (s *StructType) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *StructType) GetParameters() []interface{} {
	params := make([]interface{}, len(s.Types))
	for i, p := range s.Types {
		params[i] = p
	}
	return params
}

func (s *StructType) GetType() Type                     { return s }
func (s *StructType) GetNullability() Nullability       { return s.Nullability }
func (s *StructType) GetTypeVariationReference() uint32 { return s.TypeVariationRef }

func (t *StructType) Equals(rhs Type) bool {
	if b, ok := rhs.(*StructType); ok {
		switch {
		case t.Nullability != b.Nullability:
			return false
		case t.TypeVariationRef != b.TypeVariationRef:
			return false
		case len(t.Types) != len(b.Types):
			return false
		}

		for i := range t.Types {
			if !t.Types[i].Equals(b.Types[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *StructType) ToProto() *proto.Type {
	children := make([]*proto.Type, len(t.Types))
	for i, c := range t.Types {
		children[i] = TypeToProto(c)
	}

	return &proto.Type{Kind: &proto.Type_Struct_{
		Struct: &proto.Type_Struct{Types: children,
			TypeVariationReference: t.TypeVariationRef,
			Nullability:            t.Nullability}}}
}

func (t *StructType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: t.ToProto()},
	}
}

func (*StructType) ShortString() string { return "struct" }

func (t *StructType) String() string {
	var b strings.Builder
	b.WriteString("struct")
	b.WriteString(strNullable(t))
	b.WriteByte('<')
	for i, f := range t.Types {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.String())
	}
	b.WriteByte('>')
	return b.String()
}

func (t *StructType) ParameterString() string {
	sb := strings.Builder{}
	for i, typ := range t.Types {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(typ.String())
	}
	return sb.String()
}

func (*StructType) BaseString() string {
	return "struct"
}

// DepthFirstNameCount returns the number of names required to name all
// fields in the struct in depth-first order. Each field consumes one name,
// and nested StructType fields recursively consume additional names,
// including structs nested inside list or map types.
func (t *StructType) DepthFirstNameCount() int {
	count := 0
	for _, typ := range t.Types {
		count++
		count += nestedNameCount(typ)
	}
	return count
}

// nestedNameCount returns the number of additional names consumed by
// nested types. It mirrors the depth-first traversal in NamedStruct.String().
func nestedNameCount(t Type) int {
	switch t := t.(type) {
	case *StructType:
		return t.DepthFirstNameCount()
	case *ListType:
		return nestedNameCount(t.Type)
	case *MapType:
		return nestedNameCount(t.Key) + nestedNameCount(t.Value)
	default:
		return 0
	}
}

// FuncType represents a function type for higher-order functions.
// It describes a function that takes parameters of specified types and
// returns a value of a specified type.
//
// Note: FuncType does not support type variations (always returns 0 for
// GetTypeVariationReference) because function types are abstract and have
// no physical representation.
type FuncType struct {
	Nullability    Nullability
	ParameterTypes []Type
	ReturnType     Type
}

func (*FuncType) isRootRef() {}

func (f *FuncType) WithNullability(n Nullability) Type {
	out := *f
	out.Nullability = n
	return &out
}

func (f *FuncType) GetParameters() []interface{} {
	// Return all type components: parameter types + return type
	// This is needed for type derivation with parameterized function types like func<T1, T2 -> T3>
	params := make([]interface{}, len(f.ParameterTypes)+1)
	for i, p := range f.ParameterTypes {
		params[i] = p
	}
	params[len(f.ParameterTypes)] = f.ReturnType
	return params
}

func (f *FuncType) GetType() Type                     { return f }
func (f *FuncType) GetNullability() Nullability       { return f.Nullability }
func (f *FuncType) GetTypeVariationReference() uint32 { return 0 } // FuncType doesn't support variations

func (f *FuncType) Equals(rhs Type) bool {
	if b, ok := rhs.(*FuncType); ok {
		switch {
		case f.Nullability != b.Nullability:
			return false
		case len(f.ParameterTypes) != len(b.ParameterTypes):
			return false
		case !f.ReturnType.Equals(b.ReturnType):
			return false
		}

		for i := range f.ParameterTypes {
			if !f.ParameterTypes[i].Equals(b.ParameterTypes[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (f *FuncType) ToProto() *proto.Type {
	params := make([]*proto.Type, len(f.ParameterTypes))
	for i, p := range f.ParameterTypes {
		params[i] = TypeToProto(p)
	}

	return &proto.Type{Kind: &proto.Type_Func_{
		Func: &proto.Type_Func{
			ParameterTypes: params,
			ReturnType:     TypeToProto(f.ReturnType),
			Nullability:    f.Nullability,
		}}}
}

func (f *FuncType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: f.ToProto()},
	}
}

func (*FuncType) ShortString() string { return "func" }

func (f *FuncType) String() string {
	var b strings.Builder
	b.WriteString("func")
	b.WriteString(strNullable(f))
	b.WriteByte('<')
	for i, p := range f.ParameterTypes {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.String())
	}
	b.WriteString(" -> ")
	b.WriteString(f.ReturnType.String())
	b.WriteByte('>')
	return b.String()
}

type ListType struct {
	Nullability      Nullability
	TypeVariationRef uint32

	Type Type
}

func (*ListType) isRootRef() {}
func (s *ListType) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *ListType) GetParameters() []interface{} {
	return []interface{}{s.Type}
}

func (s *ListType) GetType() Type                     { return s }
func (s *ListType) GetNullability() Nullability       { return s.Nullability }
func (s *ListType) GetTypeVariationReference() uint32 { return s.TypeVariationRef }

func (t *ListType) Equals(rhs Type) bool {
	if b, ok := rhs.(*ListType); ok {
		switch {
		case t.Nullability != b.Nullability:
			return false
		case t.TypeVariationRef != b.TypeVariationRef:
			return false
		}

		return t.Type.Equals(b.Type)
	}
	return false
}

func (t *ListType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_List_{
		List: &proto.Type_List{Nullability: t.Nullability,
			Type:                   TypeToProto(t.Type),
			TypeVariationReference: t.TypeVariationRef}}}
}

func (t *ListType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: t.ToProto()},
	}
}

func (*ListType) ShortString() string { return "list" }

func (t *ListType) String() string {
	return "list" + strNullable(t) + "<" + t.Type.String() + ">"
}

func (s *ListType) ParameterString() string {
	return s.Type.String()
}

func (*ListType) BaseString() string {
	return "list"
}

type MapType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Key, Value       Type
}

func (*MapType) isRootRef() {}
func (s *MapType) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *MapType) GetParameters() []interface{} {
	return []interface{}{s.Key, s.Value}
}

func (s *MapType) GetType() Type                     { return s }
func (s *MapType) GetNullability() Nullability       { return s.Nullability }
func (s *MapType) GetTypeVariationReference() uint32 { return s.TypeVariationRef }

func (t *MapType) Equals(rhs Type) bool {
	if b, ok := rhs.(*MapType); ok {
		switch {
		case t.Nullability != b.Nullability:
			return false
		case t.TypeVariationRef != b.TypeVariationRef:
			return false
		}

		return t.Key.Equals(b.Key) && t.Value.Equals(b.Value)
	}
	return false
}

func (t *MapType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_Map_{
		Map: &proto.Type_Map{Nullability: t.Nullability,
			TypeVariationReference: t.TypeVariationRef,
			Key:                    TypeToProto(t.Key),
			Value:                  TypeToProto(t.Value)}}}
}

func (t *MapType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: t.ToProto()},
	}
}

func (t *MapType) ShortString() string { return "map" }

func (t *MapType) String() string {
	return "map" + strNullable(t) + "<" + t.Key.String() + ", " + t.Value.String() + ">"
}

func (t *MapType) ParameterString() string {
	return fmt.Sprintf("%s, %s", t.Key.String(), t.Value.String())
}

func (*MapType) BaseString() string {
	return "map"
}

// TypeParam represents a type parameter for a user defined type
type TypeParam interface {
	ToProto() *proto.Type_Parameter
	Equals(TypeParam) bool
}

// rather than creating a new one of these for every call ToProto which
// will always be the same empty object we can just create this once
// and return the same one every time.
var nullTypeParam = &proto.Type_Parameter_Null{}

// NullParameter is an explicitly null/unspecified parameter, to select
// the default value (if any).
type NullParameter struct{}

func (NullParameter) Equals(p TypeParam) bool {
	_, ok := p.(NullParameter)
	return ok
}

func (NullParameter) ToProto() *proto.Type_Parameter {
	return &proto.Type_Parameter{Parameter: nullTypeParam}
}

// DataTypeParameter is like the i32 in LIST<i32>
type DataTypeParameter struct {
	Type
}

func (d *DataTypeParameter) Equals(p TypeParam) bool {
	if dt, ok := p.(*DataTypeParameter); ok {
		return d.Type.Equals(dt.Type)
	}
	return false
}

func (d *DataTypeParameter) ToProto() *proto.Type_Parameter {
	return &proto.Type_Parameter{Parameter: &proto.Type_Parameter_DataType{
		DataType: TypeToProto(d.Type)}}
}

// BooleanParameter is a type parameter like <true> for a type.
type BooleanParameter bool

func (b BooleanParameter) Equals(p TypeParam) bool {
	if rhs, ok := p.(BooleanParameter); ok {
		return b == rhs
	}
	return false
}

func (b BooleanParameter) ToProto() *proto.Type_Parameter {
	return &proto.Type_Parameter{Parameter: &proto.Type_Parameter_Boolean{
		Boolean: bool(b)}}
}

// IntegerParameter is the type parameter like 10 in VARCHAR<10>
type IntegerParameter int64

func (b IntegerParameter) Equals(p TypeParam) bool {
	if rhs, ok := p.(IntegerParameter); ok {
		return b == rhs
	}
	return false
}

func (p IntegerParameter) ToProto() *proto.Type_Parameter {
	return &proto.Type_Parameter{Parameter: &proto.Type_Parameter_Integer{
		Integer: int64(p)}}
}

// EnumParameter is a type parameter that is some enum value
type EnumParameter string

func (b EnumParameter) Equals(p TypeParam) bool {
	if rhs, ok := p.(EnumParameter); ok {
		return b == rhs
	}
	return false
}

func (p EnumParameter) ToProto() *proto.Type_Parameter {
	return &proto.Type_Parameter{Parameter: &proto.Type_Parameter_Enum{
		Enum: string(p)}}
}

// StringParameter is a type parameter which is a string value
type StringParameter string

func (p StringParameter) String() string {
	return string(p)
}

func (p StringParameter) Equals(o TypeParam) bool {
	if rhs, ok := o.(StringParameter); ok {
		return p == rhs
	}
	return false
}

func (p StringParameter) ToProto() *proto.Type_Parameter {
	return &proto.Type_Parameter{Parameter: &proto.Type_Parameter_String_{
		String_: string(p)}}
}

func (p StringParameter) Evaluate(symbolTable map[string]any) (any, error) {
	if v, ok := symbolTable[string(p)]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("symbol not found: stringParameter %s", p)
}

// TypeParamFromProto converts a protobuf Type_Parameter message to
// a TypeParam object for processing.
func TypeParamFromProto(p *proto.Type_Parameter) TypeParam {
	switch p := p.Parameter.(type) {
	case *proto.Type_Parameter_Null:
		return NullParameter{}
	case *proto.Type_Parameter_Boolean:
		return BooleanParameter(p.Boolean)
	case *proto.Type_Parameter_DataType:
		return &DataTypeParameter{TypeFromProto(p.DataType)}
	case *proto.Type_Parameter_Integer:
		return IntegerParameter(p.Integer)
	case *proto.Type_Parameter_Enum:
		return EnumParameter(p.Enum)
	case *proto.Type_Parameter_String_:
		return StringParameter(p.String_)
	}
	return nil
}

type UserDefinedType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	TypeReference    uint32
	TypeParameters   []TypeParam
}

func (*UserDefinedType) isRootRef() {}
func (s *UserDefinedType) WithNullability(n Nullability) Type {
	out := *s
	out.Nullability = n
	return &out
}

func (s *UserDefinedType) GetParameters() []interface{} {
	params := make([]interface{}, len(s.TypeParameters))
	for i, p := range s.TypeParameters {
		params[i] = p
	}
	return params
}

func (s *UserDefinedType) GetType() Type                     { return s }
func (s *UserDefinedType) GetNullability() Nullability       { return s.Nullability }
func (s *UserDefinedType) GetTypeVariationReference() uint32 { return s.TypeVariationRef }

func (t *UserDefinedType) Equals(rhs Type) bool {
	if other, ok := rhs.(*UserDefinedType); ok {
		switch {
		case t.Nullability != other.Nullability:
			return false
		case t.TypeVariationRef != other.TypeVariationRef:
			return false
		case t.TypeReference != other.TypeReference:
			return false
		case len(t.TypeParameters) != len(other.TypeParameters):
			return false
		}

		for i := range t.TypeParameters {
			if !t.TypeParameters[i].Equals(other.TypeParameters[i]) {
				return false
			}
		}
		return true
	}

	return false
}

func (t *UserDefinedType) ToProto() *proto.Type {
	params := make([]*proto.Type_Parameter, len(t.TypeParameters))
	for i, p := range t.TypeParameters {
		params[i] = p.ToProto()
	}

	return &proto.Type{Kind: &proto.Type_UserDefined_{
		UserDefined: &proto.Type_UserDefined{
			Nullability:            t.Nullability,
			TypeVariationReference: t.TypeVariationRef,
			TypeReference:          t.TypeReference,
			TypeParameters:         params,
		}}}
}

func (t *UserDefinedType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: t.ToProto()},
	}
}

// exists for meeting the interface, but the correct short name for
// a user defined type is "u!name" which requires looking up the
// type first via the type reference to find the name.
func (*UserDefinedType) ShortString() string { return "" }

func (t *UserDefinedType) String() string {
	return "user_defined_type"
}

func (e Enum) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Enum{Enum: string(e)},
	}
}

func (e Enum) String() string { return string(e) }

type NamedStruct struct {
	Names  []string
	Struct StructType
}

func NewNamedStructFromProto(n *proto.NamedStruct) NamedStruct {
	if n == nil {
		return NamedStruct{}
	}

	fields := make([]Type, len(n.Struct.Types))
	for i, f := range n.Struct.Types {
		fields[i] = TypeFromProto(f)
	}

	return NamedStruct{
		Names: n.Names,
		Struct: StructType{
			Nullability:      n.Struct.Nullability,
			TypeVariationRef: n.Struct.TypeVariationReference,
			Types:            fields,
		},
	}
}

func (n NamedStruct) ToProto() *proto.NamedStruct {
	return &proto.NamedStruct{
		Names:  n.Names,
		Struct: n.Struct.ToProto().GetStruct(),
	}
}

func (n NamedStruct) String() string {
	var b strings.Builder

	// names are in depth-first order
	nameIdx := 0

	var writeType func(t Type)

	writeType = func(t Type) {
		switch t := t.(type) {
		case *StructType:
			b.WriteString("struct<")
			for i, c := range t.Types {
				if i != 0 {
					b.WriteString(", ")
				}
				b.WriteString(n.Names[nameIdx])
				nameIdx++
				b.WriteString(": ")
				writeType(c)
			}
			b.WriteString(">")
			b.WriteString(strNullable(t))
		case *MapType:
			b.WriteString("map<")
			writeType(t.Key)
			b.WriteString(",")
			writeType(t.Value)
			b.WriteString(">")
			b.WriteString(strNullable(t))
		case *ListType:
			b.WriteString("list<")
			writeType(t.Type)
			b.WriteString(">")
			b.WriteString(strNullable(t))
		default:
			b.WriteString(t.String())
		}
	}

	b.WriteString("NSTRUCT<")
	for i, t := range n.Struct.Types {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(n.Names[nameIdx])
		b.WriteString(": ")
		nameIdx++

		writeType(t)
	}
	b.WriteString(">")

	return b.String()
}

// RecordType is the type of a record (or row) comprising a list of fields (or columns).
type RecordType struct {
	types []Type
}

func NewRecordTypeFromTypes(types []Type) *RecordType {
	return &RecordType{types: types}
}

func NewRecordTypeFromStruct(s StructType) *RecordType {
	return &RecordType{types: s.Types}
}

func (r RecordType) Equals(other *RecordType) bool {
	return r.AsStructType().Equals(other.AsStructType())
}

func (r RecordType) String() string {
	return r.AsStructType().String()
}

func (r RecordType) GetFieldRef(index int32) Type {
	return r.types[index]
}

func (r RecordType) FieldCount() int32 {
	return int32(len(r.types))
}

func (r RecordType) AsStructType() *StructType {
	return &StructType{Nullability: NullabilityRequired, Types: r.types}
}

func (r RecordType) Types() []Type {
	return r.types
}

func (r RecordType) Concat(other RecordType) RecordType {
	return RecordType{types: append(r.Types(), other.Types()...)}
}

func (d Date) ToTimeString() string {
	date := civil.Date{Year: 1970, Month: time.January, Day: 1}
	date = date.AddDays(int(d))
	return date.String()
}

func timeFromPrecisionUnits(units int64, precision TimePrecision) time.Time {
	var tm time.Time
	switch precision {
	case PrecisionSeconds:
		tm = time.Unix(units, 0)
	case PrecisionDeciSeconds:
		tm = time.Unix(units/10, units%10*100000000)
	case PrecisionCentiSeconds:
		tm = time.Unix(units/100, units%100*10000000)
	case PrecisionMilliSeconds:
		tm = time.UnixMilli(units)
	case PrecisionEMinus4Seconds:
		tm = time.Unix(units/10000, units%10000*100000)
	case PrecisionEMinus5Seconds:
		tm = time.Unix(units/100000, units%100000*10000)
	case PrecisionMicroSeconds:
		tm = time.UnixMicro(units)
	case PrecisionEMinus7Seconds:
		tm = time.Unix(units/10000000, units%10000000*100)
	case PrecisionEMinus8Seconds:
		tm = time.Unix(units/100000000, units%100000000*10)
	case PrecisionNanoSeconds:
		tm = time.Unix(units/1000000000, units%1000000000)
	default:
		panic("unsupported precision")
	}
	return tm
}

func (t Time) ToTimeString() string {
	tm := time.UnixMicro(int64(t))
	return tm.UTC().Format(time.TimeOnly)
}

func (t Time) ToIsoTimeString() string {
	tm := time.UnixMicro(int64(t))
	return tm.UTC().Format("15:04:05.000000")
}

func (t Time) ToPrecisionTime(precision TimePrecision) time.Time {
	return timeFromPrecisionUnits(int64(t), precision)
}

func (t Timestamp) ToTime() time.Time {
	return time.UnixMicro(int64(t))
}

func (t Timestamp) ToPrecisionTime(precision TimePrecision) time.Time {
	return timeFromPrecisionUnits(int64(t), precision)
}

func (t Timestamp) ToTimeString() string {
	tm := any(t).(TimeConverter).ToTime()
	return tm.UTC().Format("2006-01-02 15:04:05.999999999")
}

func (t Timestamp) ToIsoTimeString() string {
	tm := any(t).(TimeConverter).ToTime()
	return tm.UTC().Format("2006-01-02T15:04:05.999999999")
}

func (t TimestampTz) ToTime() time.Time {
	return time.UnixMicro(int64(t))
}

func (t TimestampTz) ToPrecisionTime(precision TimePrecision) time.Time {
	return timeFromPrecisionUnits(int64(t), precision)
}

func (t TimestampTz) ToTimeString() string {
	tm := any(t).(TimeConverter).ToTime()
	return tm.UTC().Format(time.RFC3339Nano)
}

func (t TimestampTz) ToIsoTimeString() string {
	tm := any(t).(TimeConverter).ToTime()
	return tm.UTC().Format("2006-01-02T15:04:05.999999999")
}

func GetTimeValueByPrecision(tm time.Time, precision TimePrecision) int64 {
	switch precision {
	case PrecisionSeconds:
		return tm.Unix()
	case PrecisionDeciSeconds:
		return tm.UnixMilli() / 100
	case PrecisionCentiSeconds:
		return tm.UnixMilli() / 10
	case PrecisionMilliSeconds:
		return tm.UnixMilli()
	case PrecisionEMinus4Seconds:
		return tm.UnixMicro() / 100
	case PrecisionEMinus5Seconds:
		return tm.UnixMicro() / 10
	case PrecisionMicroSeconds:
		return tm.UnixMicro()
	case PrecisionEMinus7Seconds:
		return tm.UnixNano() / 100
	case PrecisionEMinus8Seconds:
		return tm.UnixNano() / 10
	case PrecisionNanoSeconds:
		return tm.UnixNano()
	default:
		panic(fmt.Sprintf("unknown TimePrecision %v", precision))
	}
}
