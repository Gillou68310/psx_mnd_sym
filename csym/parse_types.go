package csym

import (
	"fmt"
	"log"
	"strings"

	sym "github.com/mefistotelis/psx_mnd_sym"
	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

// ParseTypes parses the SYM types into the equivalent C types.
func (p *Parser) ParseTypes(syms []*sym.Symbol) {
	p.initTaggedTypes(syms)
	if p.opts.Verbose {
		fmt.Printf("Parsing %d symbol tags for types...\n", len(syms))
	}
	// Parse symbols.
	for i := 0; i < len(syms); i++ {
		s := syms[i]
		switch body := s.Body.(type) {
		case *sym.Def:
			switch body.Class {
			case sym.ClassSTRTAG:
				n := p.parseStructTag(body, syms[i+1:])
				i += n
			case sym.ClassUNTAG:
				n := p.parseUnionTag(body, syms[i+1:])
				i += n
			case sym.ClassENTAG:
				n := p.parseEnumTag(body, syms[i+1:])
				i += n
			case sym.ClassTPDEF:
				// TODO: Replace with parseDef?
				p.parseTypedef(body.Type, nil, "", body.Name)
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassTPDEF:
				// TODO: Replace with parseDef?
				p.parseTypedef(body.Type, body.Dims, body.Tag, body.Name)
			}
			// We are not using 'default:', here nor in body.Class switches; that is because
			// such verification is made when parsing declarations (`parse_decls.go`)
		}
	}
	if p.opts.Verbose {
		fmt.Printf("Created %d structs, %d enums, %d unions, %d types.\n",
			len(p.Structs), len(p.Enums), len(p.Unions), len(p.Types))
	}
}

// initTaggedTypes adds scaffolding types for structs, unions and enums.
func (p *Parser) initTaggedTypes(syms []*sym.Symbol) {
	if p.opts.Verbose {
		fmt.Printf("Initializing tagged types...\n")
	}
	// Bool used for NULL type.
	boolDef := &c.VarDecl{
		Class: c.Typedef,
		Var: c.Var{
			Type: c.Int,
			Name: "bool",
		},
	}
	p.Types["bool"] = boolDef
	// Add scaffolding types for structs, unions and enums, so they may be
	// referrenced before defined.
	p.emptyStruct("__vtbl_ptr_type", 0)
	for _, s := range syms {
		switch body := s.Body.(type) {
		case *sym.Def:
			tag := validName(body.Name)
			switch body.Class {
			case sym.ClassSTRTAG:
				p.emptyStruct(tag, body.Size)
			case sym.ClassUNTAG:
				p.emptyUnion(tag, body.Size)
			case sym.ClassENTAG:
				p.emptyEnum(tag)
			}
		}
	}
}

// parseStructTag parses a struct tag sequence of symbols.
func (p *Parser) parseStructTag(body *sym.Def, syms []*sym.Symbol) (n int) {
	if base := body.Type.Base(); base != sym.BaseStruct {
		panic(fmt.Errorf("support for base type %q not yet implemented", base))
	}
	tag := validName(body.Name)
	t := findEmptyStruct(p, tag, body.Size)
	for n = 0; n < len(syms); n++ {
		s := syms[n]
		switch body := s.Body.(type) {
		case *sym.Def:
			switch body.Class {
			case sym.ClassMOS:
				field := c.Field{
					Offset: s.Hdr.Value,
					Size:   body.Size,
					Var: c.Var{
						Type: p.parseType(body.Type, nil, ""),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
			case sym.ClassFIELD:
				// TODO: Figure out what FIELD represents. Use method for now.
				method := c.Field{
					Offset: s.Hdr.Value,
					Size:   body.Size,
					Var: c.Var{
						Type: p.parseType(body.Type, nil, ""),
						Name: validName(body.Name),
					},
				}
				t.Methods = append(t.Methods, method)
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassMOS:
				field := c.Field{
					Offset: s.Hdr.Value,
					Size:   body.Size,
					Var: c.Var{
						Type: p.parseType(body.Type, body.Dims, body.Tag),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
			case sym.ClassEOS:
				return n + 1
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		}
	}
	panic("unreachable")
}

// parseUnionTag parses a union tag sequence of symbols.
func (p *Parser) parseUnionTag(body *sym.Def, syms []*sym.Symbol) (n int) {
	if base := body.Type.Base(); base != sym.BaseUnion {
		panic(fmt.Errorf("support for base type %q not yet implemented", base))
	}
	tag := validName(body.Name)
	t := findEmptyUnion(p, tag, body.Size)
	for n = 0; n < len(syms); n++ {
		s := syms[n]
		switch body := s.Body.(type) {
		case *sym.Def:
			switch body.Class {
			case sym.ClassMOU:
				field := c.Field{
					Offset: s.Hdr.Value,
					Size:   body.Size,
					Var: c.Var{
						Type: p.parseType(body.Type, nil, ""),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassMOU:
				field := c.Field{
					Offset: s.Hdr.Value,
					Size:   body.Size,
					Var: c.Var{
						Type: p.parseType(body.Type, body.Dims, body.Tag),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
			case sym.ClassEOS:
				return n + 1
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		}
	}
	panic("unreachable")
}

// parseEnumTag parses an enum tag sequence of symbols.
func (p *Parser) parseEnumTag(body *sym.Def, syms []*sym.Symbol) (n int) {
	if base := body.Type.Base(); base != sym.BaseEnum {
		panic(fmt.Errorf("support for base type %q not yet implemented", base))
	}
	tag := validName(body.Name)
	t := findEmptyEnum(p, tag)
	for n = 0; n < len(syms); n++ {
		s := syms[n]
		switch body := s.Body.(type) {
		case *sym.Def:
			switch body.Class {
			case sym.ClassMOE:
				name := validName(body.Name)
				name = uniqueEnum(name, p.enumMembers)
				member := &c.EnumMember{
					Value: s.Hdr.Value,
					Name:  name,
				}
				t.Members = append(t.Members, member)
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassEOS:
				return n + 1
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		}
	}
	panic("unreachable")
}

// parseTypedef parses a typedef symbol.
func (p *Parser) parseTypedef(t sym.Type, dims []uint32, tag, name string) {
	name = validName(name)
	def := &c.VarDecl{
		Class: c.Typedef,
		Var: c.Var{
			Type: p.parseType(t, dims, tag),
			Name: name,
		},
	}
	p.Typedefs = append(p.Typedefs, def)
	p.Types[name] = def
}

// ### [ Helper functions ] ####################################################

// Duplicate enum member format string.
const duplicateEnumFormat = "%s_DUPLICATE_%d"

// uniqueEnum returns a unique enum member based on the given tag and set of
// present enum members.
func uniqueEnum(name string, members map[string]bool) string {
	newName := name
	for i := 0; members[newName]; i++ {
		newName = fmt.Sprintf(duplicateEnumFormat, name, i)
	}
	members[newName] = true
	return newName
}

// SliceIndex returns index within slece for which the func returns true
func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

// AddStruct adds the type instance to lists within parser
func (p *Parser) AddStruct(t *c.StructType) *c.StructType {
	p.Structs = append(p.Structs, t)
	p.StructTags[t.Tag] = append(p.StructTags[t.Tag], t)
	return t
}

func replaceStructsInSlice(structs []*c.StructType, typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(structs); i++ {
		t1, ok := typeRemap[structs[i]]
		if ok {
			if t1 == nil {
				structs[i] = nil
			} else {
				structs[i] = t1.(*c.StructType)
			}
		}
	}
}

// ReplaceStructs replaces types with another in lists within parser
// It does not replace use of the struct in other types
func (p *Parser) ReplaceStructs(typeRemap map[c.Type]c.Type) {
	for _, structs := range p.StructTags {
		replaceStructsInSlice(structs, typeRemap)
	}
	replaceStructsInSlice(p.Structs, typeRemap)
}

func rmNilStructsFromSlice(structs []*c.StructType) []*c.StructType {
	for i := 0; i < len(structs); {
		if structs[i] != nil {
			i++
			continue
		}
		if i < len(structs)-1 {
			copy(structs[i:], structs[i+1:])
		}
		structs[len(structs)-1] = nil
		structs = structs[:len(structs)-1]
	}
	return structs
}

func (p *Parser) RmNilStructs() {
	for tag, structs := range p.StructTags {
		p.StructTags[tag] = rmNilStructsFromSlice(structs)
	}
	p.Structs = rmNilStructsFromSlice(p.Structs)
}

func (p *Parser) emptyStruct(tag string, size uint32) *c.StructType {
	t := &c.StructType{
		Tag:     tag,
		Size:    size,
		Emitted: false,
	}
	return p.AddStruct(t)
}

// findStruct returns the struct with the given tag and size.
func (p *Parser) findStruct(tag string, size uint32, matchSize bool) *c.StructType {
	var t *c.StructType = nil
	nameExists := false
	structs, ok := p.StructTags[tag]
	if ok {
		nameExists = len(structs) > 0
		for i := 0; i < len(structs); i++ {
			tt := structs[i]
			if matchSize && tt.Size != size {
				continue
			}
			t = tt
		}
	}
	if t == nil {
		t = p.emptyStruct(tag, size)
		if nameExists {
			t.Tag = UniqueStructTag(p.StructTags, t)
		}
		log.Printf("unable to locate struct %q, created empty", tag)
	}
	return t
}

// findEmptyStruct returns the struct with the given tag and size.
// It selects the struct which has no fields defined yet, and
// asserts that the type exists.
func findEmptyStruct(p *Parser, tag string, size uint32) *c.StructType {
	var t *c.StructType = nil
	structs, ok := p.StructTags[tag]
	if ok {
		for i := 0; i < len(structs); i++ {
			tt := structs[i]
			if tt.Size != size {
				continue
			}
			if len(tt.Fields) != 0 {
				continue
			}
			t = tt
		}
	}
	if t == nil {
		panic(fmt.Errorf("unable to locate struct %q size %d", tag, size))
	}
	return t
}

// AddUnion adds the type instance to lists within parser
func (p *Parser) AddUnion(t *c.UnionType) *c.UnionType {
	p.Unions = append(p.Unions, t)
	p.UnionTags[t.Tag] = append(p.UnionTags[t.Tag], t)
	return t
}

func replaceUnionsInSlice(unions []*c.UnionType, typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(unions); i++ {
		t1, ok := typeRemap[unions[i]]
		if ok {
			if t1 == nil {
				unions[i] = nil
			} else {
				unions[i] = t1.(*c.UnionType)
			}
		}
	}
}

// ReplaceUnions replaces types with another in lists within parser
// It does not replace use of the union in other types
func (p *Parser) ReplaceUnions(typeRemap map[c.Type]c.Type) {
	for _, unions := range p.UnionTags {
		replaceUnionsInSlice(unions, typeRemap)
	}
	replaceUnionsInSlice(p.Unions, typeRemap)
}

func rmNilUnionsFromSlice(unions []*c.UnionType) []*c.UnionType {
	for i := 0; i < len(unions); {
		if unions[i] != nil {
			i++
			continue
		}
		if i < len(unions)-1 {
			copy(unions[i:], unions[i+1:])
		}
		unions[len(unions)-1] = nil
		unions = unions[:len(unions)-1]
	}
	return unions
}

func (p *Parser) RmNilUnions() {
	for tag, unions := range p.UnionTags {
		p.UnionTags[tag] = rmNilUnionsFromSlice(unions)
	}
	p.Unions = rmNilUnionsFromSlice(p.Unions)
}

func (p *Parser) emptyUnion(tag string, size uint32) *c.UnionType {
	t := &c.UnionType{
		Tag:  tag,
		Size: size,
	}
	return p.AddUnion(t)
}

// findUnion returns the union with the given tag and size.
func (p *Parser) findUnion(tag string, size uint32, matchSize bool) *c.UnionType {
	var t *c.UnionType = nil
	nameExists := false
	unions, ok := p.UnionTags[tag]
	if ok {
		nameExists = len(unions) > 0
		for i := 0; i < len(unions); i++ {
			tt := unions[i]
			if matchSize && tt.Size != size {
				continue
			}
			t = tt
		}
	}
	if t == nil {
		t = p.emptyUnion(tag, size)
		if nameExists {
			t.Tag = UniqueUnionTag(p.UnionTags, t)
		}
		log.Printf("unable to locate union %q, created empty", tag)
	}
	return t
}

// findEmptyUnion returns the union with the given tag and size.
// It selects the union which has no fields defined yet, and
// asserts that the type exists.
func findEmptyUnion(p *Parser, tag string, size uint32) *c.UnionType {
	var t *c.UnionType = nil
	unions, ok := p.UnionTags[tag]
	if ok {
		for i := 0; i < len(unions); i++ {
			tt := unions[i]
			if tt.Size != size {
				continue
			}
			if len(tt.Fields) != 0 {
				continue
			}
			t = tt
		}
	}
	if t == nil {
		panic(fmt.Errorf("unable to locate union %q size %d", tag, size))
	}
	return t
}

// AddEnum adds the type instance to lists within parser
func (p *Parser) AddEnum(t *c.EnumType) *c.EnumType {
	p.Enums = append(p.Enums, t)
	p.EnumTags[t.Tag] = append(p.EnumTags[t.Tag], t)
	return t
}

func replaceEnumsInSlice(enums []*c.EnumType, typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(enums); i++ {
		t1, ok := typeRemap[enums[i]]
		if ok {
			if t1 == nil {
				enums[i] = nil
			} else {
				enums[i] = t1.(*c.EnumType)
			}
		}
	}
}

// ReplaceEnums replaces types with another in lists within parser
// It does not replace use of the enum in other types
func (p *Parser) ReplaceEnums(typeRemap map[c.Type]c.Type) {
	for _, enums := range p.EnumTags {
		replaceEnumsInSlice(enums, typeRemap)
	}
	replaceEnumsInSlice(p.Enums, typeRemap)
}

func rmNilEnumsFromSlice(enums []*c.EnumType) []*c.EnumType {
	for i := 0; i < len(enums); {
		if enums[i] != nil {
			i++
			continue
		}
		if i < len(enums)-1 {
			copy(enums[i:], enums[i+1:])
		}
		enums[len(enums)-1] = nil
		enums = enums[:len(enums)-1]
	}
	return enums
}

func (p *Parser) RmNilEnums() {
	for tag, enums := range p.EnumTags {
		p.EnumTags[tag] = rmNilEnumsFromSlice(enums)
	}
	p.Enums = rmNilEnumsFromSlice(p.Enums)
}

func (p *Parser) emptyEnum(tag string) *c.EnumType {
	t := &c.EnumType{
		Tag: tag,
	}
	return p.AddEnum(t)
}

// findEnum returns the enumeration with the given tag.
func (p *Parser) findEnum(tag string) *c.EnumType {
	var t *c.EnumType = nil
	nameExists := false
	enums, ok := p.EnumTags[tag]
	if ok {
		nameExists = len(enums) > 0
		for i := 0; i < len(enums); i++ {
			tt := enums[i]
			t = tt
		}
	}
	if t == nil {
		t = p.emptyEnum(tag)
		if nameExists {
			t.Tag = UniqueEnumTag(p.EnumTags, t)
		}
		log.Printf("unable to locate enum %q, created empty", tag)
	}
	return t
}

// findEmptyEnum returns the enumeration with the given tag.
// It selects the enum which has no members defined yet, and
// asserts that the type exists.
func findEmptyEnum(p *Parser, tag string) *c.EnumType {
	var t *c.EnumType = nil
	enums, ok := p.EnumTags[tag]
	if ok {
		for i := 0; i < len(enums); i++ {
			tt := enums[i]
			if len(tt.Members) != 0 {
				continue
			}
			t = tt
		}
	}
	if t == nil {
		panic(fmt.Errorf("unable to locate enum %q", tag))
	}
	return t
}

// parseType parses the SYM type into the equivalent C type.
func (p *Parser) parseType(t sym.Type, dims []uint32, tag string) c.Type {
	u := p.parseBase(t.Base(), tag)
	return parseMods(u, t.Mods(), dims)
}

// parseBase parses the SYM base type into the equivalent C type.
func (p *Parser) parseBase(base sym.Base, tag string) c.Type {
	tag = validName(tag)
	switch base {
	case sym.BaseNull:
		return p.Types["bool"]
	case sym.BaseVoid:
		return c.Void
	case sym.BaseChar:
		return c.Char
	case sym.BaseShort:
		return c.Short
	case sym.BaseInt:
		return c.Int
	case sym.BaseLong:
		return c.Long
	case sym.BaseStruct:
		return p.findStruct(tag, 0, false)
	case sym.BaseUnion:
		return p.findUnion(tag, 0, false)
	case sym.BaseEnum:
		return p.findEnum(tag)
	//case sym.BaseMOE:
	case sym.BaseUChar:
		return c.UChar
	case sym.BaseUShort:
		return c.UShort
	case sym.BaseUInt:
		return c.UInt
	case sym.BaseULong:
		return c.ULong
	case sym.BaseFloat:
		return c.Float
	default:
		panic(fmt.Errorf("base type %q not yet supported", base))
	}
}

// parseMods parses the SYM type modifiers into the equivalent C type modifiers.
func parseMods(t c.Type, mods []sym.Mod, dims []uint32) c.Type {
	j := 0
	for i := len(mods) - 1; i >= 0; i-- {
		mod := mods[i]
		switch mod {
		case sym.ModPointer:
			t = &c.PointerType{Elem: t}
		case sym.ModFunction:
			t = &c.FuncType{
				RetType: t,
			}
		case sym.ModArray:
			t = &c.ArrayType{
				Elem: t,
				Len:  int(dims[j]),
			}
			j++
		}
	}
	return t
}

// validName returns a valid C identifier based on the given name.
func validName(name string) string {
	f := func(r rune) rune {
		switch {
		case 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9':
			return r
		default:
			return '_'
		}
	}
	return strings.Map(f, name)
}
