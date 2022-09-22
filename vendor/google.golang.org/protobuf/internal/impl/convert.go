// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// unwrapper unwraps the value to the underlying value.
// This is implemented by List and Map.
type unwrapper interface {
	protoUnwrap() interface{}
}

// A Converter coverts to/from Go reflect.Value types and protobuf protoreflect.Value types.
type Converter interface {
	// PBValueOf converts a reflect.Value to a protoreflect.Value.
	PBValueOf(reflect.Value) protoreflect.Value

	// GoValueOf converts a protoreflect.Value to a reflect.Value.
	GoValueOf(protoreflect.Value) reflect.Value

	// IsValidPB returns whether a protoreflect.Value is compatible with this type.
	IsValidPB(protoreflect.Value) bool

	// IsValidGo returns whether a reflect.Value is compatible with this type.
	IsValidGo(reflect.Value) bool

	// New returns a new field value.
	// For scalars, it returns the default value of the field.
	// For composite types, it returns a new mutable value.
	New() protoreflect.Value

	// Zero returns a new field value.
	// For scalars, it returns the default value of the field.
	// For composite types, it returns an immutable, empty value.
	Zero() protoreflect.Value
}

// NewConverter matches a Go type with a protobuf field and returns a Converter
// that converts between the two. Enums must be a named int32 kind that
// implements protoreflect.Enum, and messages must be pointer to a named
// struct type that implements protoreflect.ProtoMessage.
//
// This matcher deliberately supports a wider range of Go types than what
// protoc-gen-go historically generated to be able to automatically wrap some
// v1 messages generated by other forks of protoc-gen-go.
func NewConverter(t reflect.Type, fd protoreflect.FieldDescriptor) Converter {
	switch {
	case fd.IsList():
		return newListConverter(t, fd)
	case fd.IsMap():
		return newMapConverter(t, fd)
	default:
		return newSingularConverter(t, fd)
	}
	panic(fmt.Sprintf("invalid Go type %v for field %v", t, fd.FullName()))
}

var (
	boolType    = reflect.TypeOf(bool(false))
	int32Type   = reflect.TypeOf(int32(0))
	int64Type   = reflect.TypeOf(int64(0))
	uint32Type  = reflect.TypeOf(uint32(0))
	uint64Type  = reflect.TypeOf(uint64(0))
	float32Type = reflect.TypeOf(float32(0))
	float64Type = reflect.TypeOf(float64(0))
	stringType  = reflect.TypeOf(string(""))
	bytesType   = reflect.TypeOf([]byte(nil))
	byteType    = reflect.TypeOf(byte(0))
)

var (
	boolZero    = protoreflect.ValueOfBool(false)
	int32Zero   = protoreflect.ValueOfInt32(0)
	int64Zero   = protoreflect.ValueOfInt64(0)
	uint32Zero  = protoreflect.ValueOfUint32(0)
	uint64Zero  = protoreflect.ValueOfUint64(0)
	float32Zero = protoreflect.ValueOfFloat32(0)
	float64Zero = protoreflect.ValueOfFloat64(0)
	stringZero  = protoreflect.ValueOfString("")
	bytesZero   = protoreflect.ValueOfBytes(nil)
)

func newSingularConverter(t reflect.Type, fd protoreflect.FieldDescriptor) Converter {
	defVal := func(fd protoreflect.FieldDescriptor, zero protoreflect.Value) protoreflect.Value {
		if fd.Cardinality() == protoreflect.Repeated {
			// Default isn't defined for repeated fields.
			return zero
		}
		return fd.Default()
	}
	switch fd.Kind() {
	case protoreflect.BoolKind:
		if t.Kind() == reflect.Bool {
			return &boolConverter{t, defVal(fd, boolZero)}
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if t.Kind() == reflect.Int32 {
			return &int32Converter{t, defVal(fd, int32Zero)}
		}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if t.Kind() == reflect.Int64 {
			return &int64Converter{t, defVal(fd, int64Zero)}
		}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if t.Kind() == reflect.Uint32 {
			return &uint32Converter{t, defVal(fd, uint32Zero)}
		}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if t.Kind() == reflect.Uint64 {
			return &uint64Converter{t, defVal(fd, uint64Zero)}
		}
	case protoreflect.FloatKind:
		if t.Kind() == reflect.Float32 {
			return &float32Converter{t, defVal(fd, float32Zero)}
		}
	case protoreflect.DoubleKind:
		if t.Kind() == reflect.Float64 {
			return &float64Converter{t, defVal(fd, float64Zero)}
		}
	case protoreflect.StringKind:
		if t.Kind() == reflect.String || (t.Kind() == reflect.Slice && t.Elem() == byteType) {
			return &stringConverter{t, defVal(fd, stringZero)}
		}
	case protoreflect.BytesKind:
		if t.Kind() == reflect.String || (t.Kind() == reflect.Slice && t.Elem() == byteType) {
			return &bytesConverter{t, defVal(fd, bytesZero)}
		}
	case protoreflect.EnumKind:
		// Handle enums, which must be a named int32 type.
		if t.Kind() == reflect.Int32 {
			return newEnumConverter(t, fd)
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return newMessageConverter(t)
	}
	panic(fmt.Sprintf("invalid Go type %v for field %v", t, fd.FullName()))
}

type boolConverter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *boolConverter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfBool(v.Bool())
}
func (c *boolConverter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(v.Bool()).Convert(c.goType)
}
func (c *boolConverter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(bool)
	return ok
}
func (c *boolConverter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *boolConverter) New() protoreflect.Value  { return c.def }
func (c *boolConverter) Zero() protoreflect.Value { return c.def }

type int32Converter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *int32Converter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfInt32(int32(v.Int()))
}
func (c *int32Converter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(int32(v.Int())).Convert(c.goType)
}
func (c *int32Converter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(int32)
	return ok
}
func (c *int32Converter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *int32Converter) New() protoreflect.Value  { return c.def }
func (c *int32Converter) Zero() protoreflect.Value { return c.def }

type int64Converter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *int64Converter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfInt64(int64(v.Int()))
}
func (c *int64Converter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(int64(v.Int())).Convert(c.goType)
}
func (c *int64Converter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(int64)
	return ok
}
func (c *int64Converter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *int64Converter) New() protoreflect.Value  { return c.def }
func (c *int64Converter) Zero() protoreflect.Value { return c.def }

type uint32Converter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *uint32Converter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfUint32(uint32(v.Uint()))
}
func (c *uint32Converter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(uint32(v.Uint())).Convert(c.goType)
}
func (c *uint32Converter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(uint32)
	return ok
}
func (c *uint32Converter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *uint32Converter) New() protoreflect.Value  { return c.def }
func (c *uint32Converter) Zero() protoreflect.Value { return c.def }

type uint64Converter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *uint64Converter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfUint64(uint64(v.Uint()))
}
func (c *uint64Converter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(uint64(v.Uint())).Convert(c.goType)
}
func (c *uint64Converter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(uint64)
	return ok
}
func (c *uint64Converter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *uint64Converter) New() protoreflect.Value  { return c.def }
func (c *uint64Converter) Zero() protoreflect.Value { return c.def }

type float32Converter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *float32Converter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfFloat32(float32(v.Float()))
}
func (c *float32Converter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(float32(v.Float())).Convert(c.goType)
}
func (c *float32Converter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(float32)
	return ok
}
func (c *float32Converter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *float32Converter) New() protoreflect.Value  { return c.def }
func (c *float32Converter) Zero() protoreflect.Value { return c.def }

type float64Converter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *float64Converter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfFloat64(float64(v.Float()))
}
func (c *float64Converter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(float64(v.Float())).Convert(c.goType)
}
func (c *float64Converter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(float64)
	return ok
}
func (c *float64Converter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *float64Converter) New() protoreflect.Value  { return c.def }
func (c *float64Converter) Zero() protoreflect.Value { return c.def }

type stringConverter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *stringConverter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfString(v.Convert(stringType).String())
}
func (c *stringConverter) GoValueOf(v protoreflect.Value) reflect.Value {
	// pref.Value.String never panics, so we go through an interface
	// conversion here to check the type.
	s := v.Interface().(string)
	if c.goType.Kind() == reflect.Slice && s == "" {
		return reflect.Zero(c.goType) // ensure empty string is []byte(nil)
	}
	return reflect.ValueOf(s).Convert(c.goType)
}
func (c *stringConverter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(string)
	return ok
}
func (c *stringConverter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *stringConverter) New() protoreflect.Value  { return c.def }
func (c *stringConverter) Zero() protoreflect.Value { return c.def }

type bytesConverter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func (c *bytesConverter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	if c.goType.Kind() == reflect.String && v.Len() == 0 {
		return protoreflect.ValueOfBytes(nil) // ensure empty string is []byte(nil)
	}
	return protoreflect.ValueOfBytes(v.Convert(bytesType).Bytes())
}
func (c *bytesConverter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(v.Bytes()).Convert(c.goType)
}
func (c *bytesConverter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().([]byte)
	return ok
}
func (c *bytesConverter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}
func (c *bytesConverter) New() protoreflect.Value  { return c.def }
func (c *bytesConverter) Zero() protoreflect.Value { return c.def }

type enumConverter struct {
	goType reflect.Type
	def    protoreflect.Value
}

func newEnumConverter(goType reflect.Type, fd protoreflect.FieldDescriptor) Converter {
	var def protoreflect.Value
	if fd.Cardinality() == protoreflect.Repeated {
		def = protoreflect.ValueOfEnum(fd.Enum().Values().Get(0).Number())
	} else {
		def = fd.Default()
	}
	return &enumConverter{goType, def}
}

func (c *enumConverter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	return protoreflect.ValueOfEnum(protoreflect.EnumNumber(v.Int()))
}

func (c *enumConverter) GoValueOf(v protoreflect.Value) reflect.Value {
	return reflect.ValueOf(v.Enum()).Convert(c.goType)
}

func (c *enumConverter) IsValidPB(v protoreflect.Value) bool {
	_, ok := v.Interface().(protoreflect.EnumNumber)
	return ok
}

func (c *enumConverter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}

func (c *enumConverter) New() protoreflect.Value {
	return c.def
}

func (c *enumConverter) Zero() protoreflect.Value {
	return c.def
}

type messageConverter struct {
	goType reflect.Type
}

func newMessageConverter(goType reflect.Type) Converter {
	return &messageConverter{goType}
}

func (c *messageConverter) PBValueOf(v reflect.Value) protoreflect.Value {
	if v.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), c.goType))
	}
	if c.isNonPointer() {
		if v.CanAddr() {
			v = v.Addr() // T => *T
		} else {
			v = reflect.Zero(reflect.PtrTo(v.Type()))
		}
	}
	if m, ok := v.Interface().(protoreflect.ProtoMessage); ok {
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	}
	return protoreflect.ValueOfMessage(legacyWrapMessage(v))
}

func (c *messageConverter) GoValueOf(v protoreflect.Value) reflect.Value {
	m := v.Message()
	var rv reflect.Value
	if u, ok := m.(unwrapper); ok {
		rv = reflect.ValueOf(u.protoUnwrap())
	} else {
		rv = reflect.ValueOf(m.Interface())
	}
	if c.isNonPointer() {
		if rv.Type() != reflect.PtrTo(c.goType) {
			panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), reflect.PtrTo(c.goType)))
		}
		if !rv.IsNil() {
			rv = rv.Elem() // *T => T
		} else {
			rv = reflect.Zero(rv.Type().Elem())
		}
	}
	if rv.Type() != c.goType {
		panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), c.goType))
	}
	return rv
}

func (c *messageConverter) IsValidPB(v protoreflect.Value) bool {
	m := v.Message()
	var rv reflect.Value
	if u, ok := m.(unwrapper); ok {
		rv = reflect.ValueOf(u.protoUnwrap())
	} else {
		rv = reflect.ValueOf(m.Interface())
	}
	if c.isNonPointer() {
		return rv.Type() == reflect.PtrTo(c.goType)
	}
	return rv.Type() == c.goType
}

func (c *messageConverter) IsValidGo(v reflect.Value) bool {
	return v.IsValid() && v.Type() == c.goType
}

func (c *messageConverter) New() protoreflect.Value {
	if c.isNonPointer() {
		return c.PBValueOf(reflect.New(c.goType).Elem())
	}
	return c.PBValueOf(reflect.New(c.goType.Elem()))
}

func (c *messageConverter) Zero() protoreflect.Value {
	return c.PBValueOf(reflect.Zero(c.goType))
}

// isNonPointer reports whether the type is a non-pointer type.
// This never occurs for generated message types.
func (c *messageConverter) isNonPointer() bool {
	return c.goType.Kind() != reflect.Ptr
}