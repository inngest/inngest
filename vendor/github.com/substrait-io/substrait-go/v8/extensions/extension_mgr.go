// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"sync"

	"github.com/creasty/defaults"
	"github.com/goccy/go-yaml"
	"github.com/substrait-io/substrait"
	substraitgo "github.com/substrait-io/substrait-go/v8"
	"github.com/substrait-io/substrait-protobuf/go/substraitpb/extensions"
)

type AdvancedExtension = extensions.AdvancedExtension

const SubstraitDefaultURIPrefix = "https://github.com/substrait-io/substrait/blob/main/extensions/"
const SubstraitDefaultURNPrefix = "extension:io.substrait:"

var (
	getDefaultCollectionOnce = sync.OnceValues(loadDefaultCollection)
	unsupportedExtensions    = map[string]bool{
		"unknown.yaml": true,
	}
)

var urnPattern = regexp.MustCompile(`^extension:[^:]+:[^:]+$`)

// validateURN validates that a URN follows the format "extension:<owner>:<id>"
func validateUrn(urn string) bool {
	return urnPattern.MatchString(urn)
}

// GetDefaultCollectionWithNoError returns a Collection that is loaded with the default Substrait extension definitions.
// This version is provided for the ease of use of legacy code. Please use GetDefaultCollection instead.
func GetDefaultCollectionWithNoError() *Collection {
	c, err := GetDefaultCollection()
	if err != nil {
		panic(err)
	}
	return c
}

// GetDefaultCollection returns a Collection that is loaded with the default Substrait extension definitions.
func GetDefaultCollection() (*Collection, error) {
	return getDefaultCollectionOnce()
}

func loadDefaultCollection() (*Collection, error) {
	substraitFS := substrait.GetSubstraitExtensionsFS()
	entries, err := substraitFS.ReadDir("extensions")
	if err != nil {
		return nil, err
	}

	var defaultCollection Collection
	for _, ent := range entries {
		err2 := loadExtensionFile(&defaultCollection, substraitFS, ent)
		if err2 != nil {
			return nil, err2
		}
	}
	return &defaultCollection, nil
}

func loadExtensionFile(collection *Collection, substraitFS embed.FS, ent fs.DirEntry) error {
	f, err := substraitFS.Open(path.Join("extensions/", ent.Name()))
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	fileStat, err := f.Stat()
	if err != nil {
		return err
	}
	fileName := path.Base(fileStat.Name())
	if _, ok := unsupportedExtensions[fileName]; !ok {
		err = collection.Load(SubstraitDefaultURIPrefix+ent.Name(), f)
		if err != nil {
			return err
		}
	}
	return nil
}

// This is just an implementation detail during the uri -> urn migration
// This entire struct will be dropped once the migration is complete.
type uriUrnBiMap struct {
	uriToUrn map[string]string
	urnToUri map[string]string
}

func newuriUrnBiMap() *uriUrnBiMap {
	return &uriUrnBiMap{
		uriToUrn: make(map[string]string),
		urnToUri: make(map[string]string),
	}
}

// Here we are assuming that we are never overwriting.
// This is okay because we actually check for the existence right before
// calling this function
func (bm *uriUrnBiMap) add(uri, urn string) {
	bm.uriToUrn[uri] = urn
	bm.urnToUri[urn] = uri
}

func (bm *uriUrnBiMap) getUri(urn string) (string, bool) {
	uri, found := bm.urnToUri[urn]
	return uri, found
}

func (bm *uriUrnBiMap) getUrn(uri string) (string, bool) {
	urn, found := bm.uriToUrn[uri]
	return urn, found
}

// ID is the unique identifier for a substrait object
type ID struct {
	URN string
	// Name of the object. For functions, a simple name may be used for lookups,
	// but as a unique identifier the compound name will be used
	Name string
}

type Collection struct {
	urnSet      map[string]struct{}
	urnUriBiMap *uriUrnBiMap

	simpleNameMap map[ID]string

	scalarMap        map[ID]*ScalarFunctionVariant
	aggregateMap     map[ID]*AggregateFunctionVariant
	windowMap        map[ID]*WindowFunctionVariant
	typeMap          map[ID]Type
	typeVariationMap map[ID]TypeVariation
	fileMetadataMap  map[string]map[string]any // keyed by URN
}

func (c *Collection) GetType(id ID) (t Type, ok bool) {
	t, ok = c.typeMap[id]
	return
}

// GetFileMetadata returns the top-level metadata from the extension file
// identified by its URN. Returns nil if no metadata was provided or the URN is not loaded.
func (c *Collection) GetFileMetadata(urn string) map[string]any {
	return c.fileMetadataMap[urn]
}

func (c *Collection) GetTypeVariation(id ID) (tv TypeVariation, ok bool) {
	tv, ok = c.typeVariationMap[id]
	return
}

var void = struct{}{}

type variants interface {
	*ScalarFunctionVariant | *AggregateFunctionVariant | *WindowFunctionVariant
	Name() string
	CompoundName() string
}

func checkMaps[T variants](id ID, m map[ID]T, simpleNames map[ID]string) (T, bool) {
	if sv, ok := m[id]; ok {
		return sv, true
	}

	if compound, ok := simpleNames[id]; ok {
		id.Name = compound
		if sv, ok := m[id]; ok {
			return sv, true
		}
	}

	return nil, false
}

type extFn[T variants] interface {
	GetVariants(urn string) []T
}

func addToMaps[T variants](id ID, fn extFn[T], m map[ID]T, simpleMap map[string]string) {
	variants := fn.GetVariants(id.URN)
	for _, v := range variants {
		id.Name = v.CompoundName()
		m[id] = v
	}

	if len(variants) == 1 {
		v := variants[0]
		if _, ok := simpleMap[v.Name()]; ok {
			delete(simpleMap, v.Name())
		} else {
			simpleMap[v.Name()] = v.CompoundName()
		}
	}
}

func (c *Collection) GetScalarFunc(id ID) (*ScalarFunctionVariant, bool) {
	return checkMaps(id, c.scalarMap, c.simpleNameMap)
}

func (c *Collection) GetAggregateFunc(id ID) (*AggregateFunctionVariant, bool) {
	return checkMaps(id, c.aggregateMap, c.simpleNameMap)
}

func (c *Collection) GetWindowFunc(id ID) (*WindowFunctionVariant, bool) {
	return checkMaps(id, c.windowMap, c.simpleNameMap)
}

func (c *Collection) init() {
	if c.urnSet == nil {
		c.urnSet = make(map[string]struct{})
		c.urnUriBiMap = newuriUrnBiMap()
		c.simpleNameMap = make(map[ID]string)
		c.scalarMap = make(map[ID]*ScalarFunctionVariant)
		c.aggregateMap = make(map[ID]*AggregateFunctionVariant)
		c.windowMap = make(map[ID]*WindowFunctionVariant)
		c.typeMap = make(map[ID]Type)
		c.typeVariationMap = make(map[ID]TypeVariation)
		c.fileMetadataMap = make(map[string]map[string]any)
	}
}

func (c *Collection) Load(uri string, r io.Reader) error {
	c.init()

	if c.URILoaded(uri) {
		return fmt.Errorf("%w: uri '%s' already loaded",
			substraitgo.ErrKeyExists, uri)
	}

	var file SimpleExtensionFile
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&file); err != nil {
		return err
	}

	urn := file.Urn
	if urn == "" {
		return fmt.Errorf("%w: missing urn", substraitgo.ErrInvalidSimpleExtention)
	}
	if !validateUrn(urn) {
		return fmt.Errorf("%w: invalid urn, expected format is \"extension:<owner>:<id>\", got: %s", substraitgo.ErrInvalidSimpleExtention, urn)
	}
	if c.URNLoaded(urn) {
		return fmt.Errorf("%w:  urn %s already loaded", substraitgo.ErrKeyExists, urn)
	}

	c.urnSet[urn] = void
	c.urnUriBiMap.add(uri, urn)

	if file.Metadata != nil {
		c.fileMetadataMap[urn] = file.Metadata
	}

	id := ID{URN: urn}
	for _, t := range file.Types {
		id.Name = t.Name
		c.typeMap[id] = t
	}

	for _, t := range file.TypeVariations {
		id.Name = t.Name
		c.typeVariationMap[id] = t
	}

	// if there is only one implementation for a given function
	// it should be findable by its simple name in addition to the
	// compound name.
	simpleNames := make(map[string]string)

	for _, f := range file.ScalarFunctions {
		if err := defaults.Set(&f); err != nil {
			return fmt.Errorf("failure setting defaults for scalar functions: %w", err)
		}
		addToMaps[*ScalarFunctionVariant](id, &f, c.scalarMap, simpleNames)
	}

	for _, f := range file.AggregateFunctions {
		if err := defaults.Set(&f); err != nil {
			return fmt.Errorf("failure setting defaults for aggregate functions: %w", err)
		}
		addToMaps[*AggregateFunctionVariant](id, &f, c.aggregateMap, simpleNames)
	}

	for _, f := range file.WindowFunctions {
		if err := defaults.Set(&f); err != nil {
			return fmt.Errorf("failure setting defaults for window functions: %w", err)
		}
		addToMaps[*WindowFunctionVariant](id, &f, c.windowMap, simpleNames)
	}

	// Aggregate functions can be used as Window Functions
	for _, f := range file.AggregateFunctions {
		// Convert each aggregate implementation to a window implementation
		windowImpls := make([]WindowFunctionImpl, len(f.Impls))
		for i, aggImpl := range f.Impls {
			windowImpls[i] = WindowFunctionImpl{
				AggregateFunctionImpl: aggImpl,
				WindowType:            StreamingWindow, // Set window type to STREAMING
			}
		}

		wf := WindowFunction{
			Name:        f.Name,
			Description: f.Description,
			Impls:       windowImpls,
			Metadata:    f.Metadata,
		}
		if err := defaults.Set(&wf); err != nil {
			return fmt.Errorf("failure setting defaults for window functions: %w", err)
		}
		addToMaps[*WindowFunctionVariant](id, &wf, c.windowMap, simpleNames)
	}

	// add simple name aliases
	for k, v := range simpleNames {
		id.Name = k
		c.simpleNameMap[id] = v
	}

	return nil
}

func (c *Collection) URNLoaded(urn string) bool {
	_, ok := c.urnSet[urn]
	return ok
}

func (c *Collection) URILoaded(uri string) bool {
	_, found := c.urnUriBiMap.getUrn(uri)
	return found
}

func (c *Collection) GetAllScalarFunctions() []*ScalarFunctionVariant {
	return getValues(c.scalarMap)
}

func (c *Collection) GetAllAggregateFunctions() []*AggregateFunctionVariant {
	return getValues(c.aggregateMap)
}

func (c *Collection) GetAllWindowFunctions() []*WindowFunctionVariant {
	return getValues(c.windowMap)
}

func getValues[M ~map[K]V, K comparable, V any](m M) []V {
	result := make([]V, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

type Set interface {
	DecodeTypeVariation(anchor uint32) (ID, bool)
	DecodeFunc(anchor uint32) (ID, bool)
	DecodeType(anchor uint32) (ID, bool)
	LookupTypeVariation(anchor uint32, c *Collection) (TypeVariation, bool)
	LookupType(anchor uint32, c *Collection) (Type, bool)
	LookupScalarFunction(anchor uint32, c *Collection) (*ScalarFunctionVariant, bool)
	LookupAggregateFunction(anchor uint32, c *Collection) (*AggregateFunctionVariant, bool)
	LookupWindowFunction(anchor uint32, c *Collection) (*WindowFunctionVariant, bool)

	FindURN(urn string) (uint32, bool)
	GetTypeAnchor(id ID) uint32
	GetFuncAnchor(id ID) uint32
	GetTypeVariationAnchor(id ID) uint32

	ToProto(c *Collection) ([]*extensions.SimpleExtensionURN, []*extensions.SimpleExtensionDeclaration)
}

func NewSet() Set {
	return &set{
		urns:             make(map[uint32]string),
		funcMap:          make(map[uint32]ID),
		funcs:            make(map[ID]uint32),
		types:            make(map[ID]uint32),
		typesMap:         make(map[uint32]ID),
		typeVariationMap: make(map[uint32]ID),
		typeVariations:   make(map[ID]uint32),
	}
}

type set struct {
	urns map[uint32]string

	typesMap map[uint32]ID
	types    map[ID]uint32

	typeVariationMap map[uint32]ID
	typeVariations   map[ID]uint32

	funcMap map[uint32]ID
	funcs   map[ID]uint32
}

func (e *set) ToProto(c *Collection) ([]*extensions.SimpleExtensionURN, []*extensions.SimpleExtensionDeclaration) {
	urnBackRef := make(map[string]uint32)

	urns := make([]*extensions.SimpleExtensionURN, 0, len(e.urns))
	for anchor, urn := range e.urns {
		urnBackRef[urn] = anchor
		urns = append(urns, &extensions.SimpleExtensionURN{
			ExtensionUrnAnchor: anchor,
			Urn:                urn,
		})
	}

	// Sort URN extensions by the anchor for consistent output
	sort.Slice(urns, func(i, j int) bool { return urns[i].ExtensionUrnAnchor < urns[j].ExtensionUrnAnchor })

	decls := make([]*extensions.SimpleExtensionDeclaration, 0, len(e.types)+len(e.typeVariations)+len(e.funcs))
	for id, anchor := range e.types {
		decls = append(decls, &extensions.SimpleExtensionDeclaration{
			MappingType: &extensions.SimpleExtensionDeclaration_ExtensionType_{
				ExtensionType: &extensions.SimpleExtensionDeclaration_ExtensionType{
					ExtensionUrnReference: urnBackRef[id.URN],
					TypeAnchor:            anchor,
					Name:                  id.Name,
				},
			},
		})
	}

	sort.Slice(decls, func(i, j int) bool {
		return decls[i].GetExtensionType().TypeAnchor < decls[j].GetExtensionType().TypeAnchor
	})
	typesCount := len(decls)

	for id, anchor := range e.typeVariations {
		decls = append(decls, &extensions.SimpleExtensionDeclaration{
			MappingType: &extensions.SimpleExtensionDeclaration_ExtensionTypeVariation_{
				ExtensionTypeVariation: &extensions.SimpleExtensionDeclaration_ExtensionTypeVariation{
					ExtensionUrnReference: urnBackRef[id.URN],
					TypeVariationAnchor:   anchor,
					Name:                  id.Name,
				},
			},
		})
	}

	typeDecls := decls[typesCount:]
	sort.Slice(typeDecls, func(i, j int) bool {
		return decls[i].GetExtensionTypeVariation().TypeVariationAnchor < decls[j].GetExtensionTypeVariation().TypeVariationAnchor
	})

	typeVarCount := len(decls)
	for id, anchor := range e.funcs {
		decls = append(decls, &extensions.SimpleExtensionDeclaration{
			MappingType: &extensions.SimpleExtensionDeclaration_ExtensionFunction_{
				ExtensionFunction: &extensions.SimpleExtensionDeclaration_ExtensionFunction{
					ExtensionUrnReference: urnBackRef[id.URN],
					FunctionAnchor:        anchor,
					Name:                  id.Name,
				},
			},
		})
	}

	typeVarDecls := decls[typeVarCount:]
	sort.Slice(typeVarDecls, func(i, j int) bool {
		return decls[i].GetExtensionFunction().GetFunctionAnchor() < decls[j].GetExtensionFunction().GetFunctionAnchor()
	})

	return urns, decls
}

func (e *set) LookupWindowFunction(anchor uint32, c *Collection) (sv *WindowFunctionVariant, ok bool) {
	id, ok := e.funcMap[anchor]
	if !ok {
		return nil, false
	}

	return c.GetWindowFunc(id)
}

func (e *set) LookupAggregateFunction(anchor uint32, c *Collection) (sv *AggregateFunctionVariant, ok bool) {
	id, ok := e.funcMap[anchor]
	if !ok {
		return nil, false
	}

	return c.GetAggregateFunc(id)
}

func (e *set) LookupScalarFunction(anchor uint32, c *Collection) (sv *ScalarFunctionVariant, ok bool) {
	id, ok := e.funcMap[anchor]
	if !ok {
		return nil, false
	}

	return c.GetScalarFunc(id)
}

func (e *set) LookupType(anchor uint32, c *Collection) (tv Type, ok bool) {
	id, ok := e.typesMap[anchor]
	if !ok {
		return tv, false
	}

	tv, ok = c.typeMap[id]
	return
}

func (e *set) LookupTypeVariation(anchor uint32, c *Collection) (tv TypeVariation, ok bool) {
	id, ok := e.typeVariationMap[anchor]
	if !ok {
		return tv, false
	}

	tv, ok = c.typeVariationMap[id]
	return
}

func (e *set) DecodeTypeVariation(anchor uint32) (id ID, ok bool) {
	id, ok = e.typeVariationMap[anchor]
	return
}

func (e *set) DecodeFunc(anchor uint32) (id ID, ok bool) {
	id, ok = e.funcMap[anchor]
	return
}

func (e *set) DecodeType(anchor uint32) (id ID, ok bool) {
	id, ok = e.typesMap[anchor]
	return
}

func (e *set) GetTypeAnchor(id ID) uint32 {
	a, ok := e.types[id]
	if !ok {
		_, err := e.addOrGetURN(id.URN)
		if err != nil {
			panic(err)
		}
		a = uint32(len(e.types)) + 1
		e.encodeType(a, id)
	}
	return a
}

func (e *set) GetFuncAnchor(id ID) uint32 {
	a, ok := e.funcs[id]
	if !ok {
		_, err := e.addOrGetURN(id.URN)
		if err != nil {
			panic(err)
		}
		a = uint32(len(e.funcs)) + 1
		e.encodeFunc(a, id)
	}
	return a
}

func (e *set) GetTypeVariationAnchor(id ID) uint32 {
	a, ok := e.typeVariations[id]
	if !ok {
		_, err := e.addOrGetURN(id.URN)
		if err != nil {
			panic(err)
		}
		// add 1 to the length to avoid an anchor of 0
		// so that it's easier to tell when there is no
		// type variation.
		a = uint32(len(e.typeVariations)) + 1
		e.encodeTypeVariation(a, id)
	}
	return a
}

func (e *set) encodeType(anchor uint32, id ID) {
	e.typesMap[anchor] = id
	e.types[id] = anchor
}

func (e *set) encodeTypeVariation(anchor uint32, id ID) {
	e.typeVariationMap[anchor] = id
	e.typeVariations[id] = anchor
}

func (e *set) encodeFunc(anchor uint32, id ID) {
	e.funcMap[anchor] = id
	e.funcs[id] = anchor
}

func (e *set) FindURN(urn string) (uint32, bool) {
	for k, v := range e.urns {
		if v == urn {
			return k, true
		}
	}
	return 0, false
}

func (e *set) addOrGetURN(urn string) (uint32, error) {
	for k, v := range e.urns {
		if v == urn {
			return k, nil
		}
	}
	sz := uint32(len(e.urns)) + 1
	if _, ok := e.urns[sz]; ok {
		return 0, fmt.Errorf("%w: URN anchor %d already exists", substraitgo.ErrKeyExists, sz)
	}

	e.urns[sz] = urn
	return sz, nil
}

type TopLevel interface {
	GetExtensionUrns() []*extensions.SimpleExtensionURN
	GetExtensions() []*extensions.SimpleExtensionDeclaration
}

func GetExtensionSet(plan TopLevel, c *Collection) (Set, error) {
	urns := make(map[uint32]string)
	for _, urn := range plan.GetExtensionUrns() {
		urns[urn.ExtensionUrnAnchor] = urn.Urn
	}

	ret := &set{
		urns:             urns,
		funcMap:          make(map[uint32]ID),
		funcs:            make(map[ID]uint32),
		typesMap:         make(map[uint32]ID),
		types:            make(map[ID]uint32),
		typeVariationMap: make(map[uint32]ID),
		typeVariations:   make(map[ID]uint32),
	}

	resolveRefToURN := func(urnRef uint32) (string, error) {
		urn, urnOk := urns[urnRef]
		if !urnOk {
			return "", fmt.Errorf("unable to resolve extension reference: URN reference %d could not be resolved", urnRef)
		}
		// Validate that the URN exists in the Collection
		if _, found := c.urnUriBiMap.getUri(urn); !found {
			return "", fmt.Errorf("%w: URN '%s' not found in extension collection", substraitgo.ErrNotFound, urn)
		}
		return urn, nil
	}

	for _, ext := range plan.GetExtensions() {
		switch e := ext.MappingType.(type) {
		case *extensions.SimpleExtensionDeclaration_ExtensionTypeVariation_:
			etv := e.ExtensionTypeVariation
			urn, err := resolveRefToURN(etv.ExtensionUrnReference)
			if err != nil {
				return nil, err
			}
			ret.encodeTypeVariation(etv.TypeVariationAnchor, ID{
				URN:  urn,
				Name: etv.Name,
			})
		case *extensions.SimpleExtensionDeclaration_ExtensionType_:
			et := e.ExtensionType
			urn, err := resolveRefToURN(et.ExtensionUrnReference)
			if err != nil {
				return nil, err
			}
			ret.encodeType(et.TypeAnchor, ID{
				URN:  urn,
				Name: et.Name,
			})
		case *extensions.SimpleExtensionDeclaration_ExtensionFunction_:
			ef := e.ExtensionFunction
			urn, err := resolveRefToURN(ef.ExtensionUrnReference)
			if err != nil {
				return nil, err
			}
			ret.encodeFunc(ef.FunctionAnchor, ID{
				URN:  urn,
				Name: ef.Name,
			})
		}
	}

	return ret, nil
}
