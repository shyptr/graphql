package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"reflect"
	"strconv"
	"time"
)

// SchemaBuilder.
//
// use to build go type into graphql type.
// include:
// 	struct->object/input
// 	enum
// 	interface
//	scalar(int,float,string...eg.)
// 	union struct
// 	defined directive
type SchemaBuilder struct {
	scalars    map[string]*ScalarBuilder
	enums      map[string]*EnumBuilder
	objects    map[string]*ObjectBuilder
	unions     map[string]*UnionBuilder
	interfaces map[string]*InterfaceBuilder
	directives map[string]*DirectiveBuilder
}

// NewSchema create a SchemaBuilder builder.
func NewSchema() *SchemaBuilder {
	schema := &SchemaBuilder{
		scalars: map[string]*ScalarBuilder{
			"Boolean":    Boolean,
			"Int":        Int,
			"Int8":       Int8,
			"Int16":      Int16,
			"Int32":      Int32,
			"Int64":      Int64,
			"Uint":       Uint,
			"Uint8":      Uint8,
			"Uint16":     Uint16,
			"Uint32":     Uint32,
			"Uint64":     Uint64,
			"Float":      Float,
			"Float64":    Float64,
			"String":     String,
			"ID":         ID,
			"Map":        MMap,
			"Time":       Time,
			"Bytes":      Bytes,
			"NullString": NullString,
			"NullTime":   NullTime,
			"NullBool":   NullBool,
			"NullFloat":  NullFloat,
			"NullInt64":  NullInt64,
			"NullInt32":  NullInt32,
			"Upload":     UploadScalar,
		},
		enums:      make(map[string]*EnumBuilder),
		objects:    make(map[string]*ObjectBuilder),
		unions:     make(map[string]*UnionBuilder),
		interfaces: make(map[string]*InterfaceBuilder),
		directives: map[string]*DirectiveBuilder{
			"include":    IncludeDirective,
			"skip":       SkipDirective,
			"deprecated": DeprecatedDirective,
		},
	}
	return schema
}

// Scalar is used to register custom scalars.
func (s *SchemaBuilder) Scalar(scalarType interface{}, opts ...Option) *ScalarBuilder {
	reflectType := reflect.TypeOf(scalarType)
	if reflectType.Kind() == reflect.Ptr {
		panic("scalarType must not be a ptr")
	}

	options := options{
		name: reflectType.Name(),
		serialize: func(value interface{}) (interface{}, error) {
			switch v := value.(type) {
			case string, float64, int64, bool, int, int8, int16, int32, uint, uint8, uint16, uint32, uint64, float32, time.Time:
				return v, nil
			case *string, *float64, *int64, *bool, *int, *int8, *int16, *int32, *uint, *uint8, *uint16, *uint32, *uint64, *float32, *time.Time:
				return v, nil
			case []byte:
				return string(v), nil
			case *[]byte:
				return string(*v), nil
			default:
				marshal, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				return string(marshal), nil
			}
		},
		parseValue: func(value interface{}) (interface{}, error) {
			if value == nil {
				return nil, nil
			}
			var x []byte
			switch v := value.(type) {
			case []byte:
				x = v
			case string:
				x = []byte(v)
			case float64:
				x = []byte(strconv.FormatFloat(v, 'g', -1, 64))
			case int64:
				x = []byte(strconv.FormatInt(v, 10))
			case bool:
				if v {
					x = []byte{'t', 'r', 'u', 'e'}
				} else {
					x = []byte{'f', 'a', 'l', 's', 'e'}
				}
			default:
				return nil, errors.New("unknown type")
			}
			r := reflect.New(reflectType).Interface()
			err := json.Unmarshal(x, r)
			return reflect.ValueOf(r).Elem().Interface(), err
		},
		parseLiteral: func(valueAST ast.Value) (interface{}, error) {
			r := reflect.New(reflectType).Interface()
			err := json.Unmarshal([]byte(valueAST.Raw), r)
			return reflect.ValueOf(r).Elem().Interface(), err
		},
	}
	for _, o := range opts {
		o(&options)
	}

	if _, ok := s.scalars[options.name]; ok {
		panic("duplicate scalar name")
	}

	scalar := &ScalarBuilder{
		Name:         options.name,
		Description:  options.description,
		Type:         reflectType,
		Serialize:    options.serialize,
		ParseValue:   options.parseValue,
		ParseLiteral: options.parseLiteral,
	}
	s.scalars[scalar.Name] = scalar
	return scalar
}

// Enum registers an enumType in the schema. The val should be any arbitrary value
// of the enumType to be used for reflection, and the enumMap should be
// the corresponding map of the enums.
func (s *SchemaBuilder) Enum(enumType interface{}, values interface{}, opts ...Option) {
	reflectType := reflect.TypeOf(enumType)
	enumValues := reflect.ValueOf(values)
	if enumValues.Kind() != reflect.Map {
		panic("enum values must be a map")
	}

	options := options{
		name: reflectType.Name(),
	}
	for _, o := range opts {
		o(&options)
	}

	if _, ok := s.enums[options.name]; ok {
		panic(fmt.Sprintf("duplicate enum %s", options.name))
	}

	rMap := make(map[interface{}]string)
	eMap := make(map[string]interface{})
	emIter := enumValues.MapRange()
	for emIter.Next() {
		val := emIter.Value()
		for val.Kind() == reflect.Interface {
			val = val.Elem()
		}
		valInterface := val.Interface()
		if val.Kind() != reflectType.Kind() {
			panic("enum types are not equal")
		}
		key := emIter.Key().String()
		eMap[key] = valInterface
		rMap[valInterface] = key
	}

	s.enums[options.name] = &EnumBuilder{
		Name:          options.name,
		Description:   options.description,
		Type:          reflectType,
		Values:        eMap,
		ReverseValues: rMap,
	}
}

// Object register a struct as a Graphql Object in SchemaBuilder.
func (s *SchemaBuilder) Object(objectType interface{}, opts ...Option) *ObjectBuilder {
	reflectType := reflect.TypeOf(objectType)
	if reflectType.Kind() != reflect.Struct {
		panic("objectType must be a struct")
	}

	options := options{
		name: reflectType.Name(),
	}
	for _, o := range opts {
		o(&options)
	}

	if object, ok := s.objects[options.name]; ok {
		if object.Type != reflectType {
			panic(fmt.Sprintf("re-registered object with different type, already registered type: %s.%s", object.Type.PkgPath(), object.Type.Name()))
		}
		return object
	}

	object := &ObjectBuilder{
		Name:        options.name,
		Description: options.description,
		Type:        reflectType,
		Fields:      make(map[string]*FieldBuilder),
		Interface:   make(map[string]reflect.Type),
	}

	for _, iface := range options.interfaces {
		if !reflectType.Implements(iface) {
			panic("object must implements the interface " + iface.Name())
		}
		object.Interface[iface.Name()] = iface
	}

	s.objects[object.Name] = object
	return object
}

// Union registers a map as a GraphQL Union in our Schema.
func (s *SchemaBuilder) Union(unionType interface{}, opts ...Option) {
	reflectType := reflect.TypeOf(unionType)
	if reflectType.Kind() != reflect.Struct {
		panic("union must be a struct")
	}

	options := options{
		name: reflectType.Name(),
	}
	for _, o := range opts {
		o(&options)
	}

	if _, ok := s.unions[options.name]; ok {
		panic("duplicate union " + options.name)
	}

	types := make([]reflect.Type, reflectType.NumField())
	for i := 0; i < reflectType.NumField(); i++ {
		f := reflectType.Field(i)
		if f.Type.Kind() != reflect.Ptr || f.Type.Elem().Kind() != reflect.Struct {
			panic("union's member must be a object struct ptr")
		}
		types[i] = f.Type
	}

	s.unions[options.name] = &UnionBuilder{
		Name:        options.name,
		Description: options.description,
		Type:        reflectType,
		Types:       types,
		ResolveType: options.resolveType,
	}
}

// Interface registers a Interface as a GraphQL Interface in our Schema.
func (s *SchemaBuilder) Interface(interfaceType interface{}, typeResolve ResolveTypeFn, opts ...Option) *InterfaceBuilder {
	if interfaceType == nil {
		panic("nil type passed to Interface")
	}

	reflectType := reflect.TypeOf(interfaceType)
	if reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	if reflectType.Kind() != reflect.Interface {
		panic("Interface must be a interface Operation in Golang")
	}

	options := options{
		name: reflectType.Name(),
	}
	for _, o := range opts {
		o(&options)
	}

	if _, ok := s.interfaces[options.name]; ok {
		panic("duplicate interface " + options.name)
	}

	s.interfaces[options.name] = &InterfaceBuilder{
		Name:          options.name,
		Description:   options.description,
		Type:          reflectType,
		ResolveType:   typeResolve,
		PossibleTypes: options.possibleTypes,
		Fields:        make(map[string]*FieldBuilder),
	}
	return s.interfaces[options.name]
}

// Directive defined directive for schema.
func (s *SchemaBuilder) Directive(name string, locations []DirectiveLocation, fn DirectiveFn, opts ...Option) {

	options := options{
		name: name,
	}
	for _, o := range opts {
		o(&options)
	}

	// Ensure directive is named
	if options.name == "" {
		panic("Directive must be named.")
	}
	// Ensure locations are provided for directive
	if len(locations) == 0 {
		panic("Must provide locations for directive.")
	}

	if fn == nil {
		panic("Must provide option func for directive")
	}

	s.directives[name] = &DirectiveBuilder{
		Name:        options.name,
		Description: options.description,
		Locations:   locations,
		Args:        options.input,
		DirectiveFn: fn,
	}
}

type Query struct{}

// Query returns an Object struct that we can use to register all the top level
// graphql Query functions we'd like to expose.
func (s *SchemaBuilder) Query() *ObjectBuilder {
	return s.Object(Query{})
}

type Mutation struct{}

// Mutation returns an Object struct that we can use to register all the top level
// graphql mutations functions we'd like to expose.
func (s *SchemaBuilder) Mutation() *ObjectBuilder {
	return s.Object(Mutation{})
}

type Subscription struct {
	Payload []byte
}

// Subscription returns an Object struct that we can use to register all the top level
// graphql subscription functions we'd like to expose.
func (s *SchemaBuilder) Subscription() *ObjectBuilder {
	return s.Object(Subscription{})
}

type schemaBuilder struct {
	types        map[reflect.Type]Type
	cacheTypes   map[reflect.Type]ResolveTypeFn
	objects      map[reflect.Type]*Object
	enums        map[reflect.Type]*Enum
	inputObjects map[reflect.Type]*InputObject
	interfaces   map[reflect.Type]*Interface
	scalars      map[reflect.Type]*Scalar
	unions       map[reflect.Type]*Union
}
