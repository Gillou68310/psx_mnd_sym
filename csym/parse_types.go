package csym

import (
	"fmt"
	"log"
	"strings"

	sym "github.com/mefistotelis/psx_mnd_sym"
	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

// ParseTypes parses the SYM types/decls into the equivalent C types.
func (p *Parser) ParseTypesDecls(syms []*sym.Symbol) {
	p.initTaggedTypes()
	if p.opts.Verbose {
		fmt.Printf("Parsing %d symbol tags for types/decls...\n", len(syms))
	}
	// Parse symbols.
	for i := 0; i < len(syms); i++ {
		s := syms[i]
		switch body := s.Body.(type) {
		case *sym.Name1:
			p.parseSymbol(s.Hdr.Value, body.Name)
		case *sym.Name2:
			p.parseSymbol(s.Hdr.Value, body.Name)
		case *sym.SetSLD2:
			n := p.parseLineNumbers(s.Hdr.Value, body, syms[i+1:])
			i += n
		case *sym.EndSLD:
			// While rarely, a group of SLD entries might end without even starting.
			// So while most SLD entry types are handled in `parseLineNumbers()`,
			// this one should be allowed on this level. Nothing to do if it is found.
		case *sym.FuncStart:
			n := p.parseFunc(s.Hdr.Value, body, syms[i+1:])
			i += n
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
				p.parseTypedef(body.Type, nil, "", body.Name, body.Size)
			case sym.ClassEXT, sym.ClassSTAT:
				t := p.parseType(body.Type, nil, "", body.Size)
				p.parseGlobalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
			case sym.ClassMOS, sym.ClassMOU, sym.ClassMOE, sym.ClassFIELD:
				// nothing to do.
				panic("")
			case sym.Class103:
				if p.opts.Verbose {
					fmt.Printf("%s\n", body.Name)
				}
			default:
				panic(fmt.Sprintf("support for symbol class %q not yet implemented", body.Class))
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassTPDEF:
				// TODO: Replace with parseDef?
				p.parseTypedef(body.Type, body.Dims, body.Tag, body.Name, body.Size)
			case sym.ClassEXT, sym.ClassSTAT:
				t := p.parseType(body.Type, body.Dims, body.Tag, body.Size)
				switch t.(type) {
				case *c.PointerType:
					body.Size = 4
				}
				p.parseGlobalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
			case sym.ClassMOS, sym.ClassMOU, sym.ClassMOE, sym.ClassFIELD, sym.ClassEOS:
				panic("")
				// nothing to do.
			default:
				panic(fmt.Sprintf("support for symbol class %q not yet implemented", body.Class))
			}
			// We are not using 'default:', here nor in body.Class switches; that is because
			// such verification is made when parsing declarations (`parse_decls.go`)
		case *sym.Overlay:
			p.parseOverlay(s.Hdr.Value, body)
		case *sym.SetOverlay:
			overlay, ok := p.overlayIDs[s.Hdr.Value]
			if !ok {
				panic(fmt.Errorf("unable to locate overlay with ID %x", s.Hdr.Value))
			}
			p.CurOverlay = overlay
			p.initTaggedTypes()
		}
	}
	if p.opts.Verbose {
		fmt.Printf("Created %d functions, %d global variables in main segment\n", len(p.Overlay.Funcs), len(p.Overlay.Vars))
		fmt.Printf("Created %d structs, %d enums, %d unions, %d types in main segment.\n", len(p.Overlay.Structs), len(p.Overlay.Enums), len(p.Overlay.Unions), len(p.Overlay.Types))
		for _, overlay := range p.Overlays {
			fmt.Printf("Created %d functions, %d global variables in overlay %d\n", len(overlay.Funcs), len(overlay.Vars), overlay.ID)
			fmt.Printf("Created %d structs, %d enums, %d unions, %d types in overlay %d.\n", len(overlay.Structs), len(overlay.Enums), len(overlay.Unions), len(overlay.Types), overlay.ID)
		}
	}
}

// initTaggedTypes adds scaffolding types for structs, unions and enums.
func (p *Parser) initTaggedTypes() {
	if len(p.CurOverlay.Types["bool"]) > 0 {
		return
	}

	// Bool used for NULL type.
	boolDef := &c.VarDecl{
		Class: c.Typedef,
		Var: c.Var{
			Type: c.Int,
			Name: "bool",
		},
	}
	p.CurOverlay.Types["bool"] = append(p.CurOverlay.Types["bool"], boolDef)
}

func validateSize(t c.Type, size uint32, elmt uint32) {
	switch tt := t.(type) {
	case *c.StructType:
		if size != tt.Size*elmt {
			panic("")
		}
	case *c.UnionType:
		if size != tt.Size*elmt {
			panic("")
		}
	case *c.ArrayType:
		validateSize(tt.Elem, size, elmt*uint32(tt.Len))
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
						Type: p.parseType(body.Type, nil, "", body.Size),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
				validateSize(field.Type, field.Size, 1)
			case sym.ClassFIELD:
				field := c.Field{
					Offset:   s.Hdr.Value,
					Size:     body.Size,
					Bitfield: true,
					Var: c.Var{
						Type: p.parseType(body.Type, nil, "", body.Size),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
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
						Type: p.parseType(body.Type, body.Dims, body.Tag, body.Size),
						Name: validName(body.Name),
					},
				}
				switch field.Type.(type) {
				case *c.PointerType:
					field.Size = 4
				}
				t.Fields = append(t.Fields, field)
				validateSize(field.Type, field.Size, 1)
			case sym.ClassEOS:
				return n + 1
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		default:
			panic("")
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
						Type: p.parseType(body.Type, nil, "", body.Size),
						Name: validName(body.Name),
					},
				}
				t.Fields = append(t.Fields, field)
				validateSize(field.Type, field.Size, 1)
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
						Type: p.parseType(body.Type, body.Dims, body.Tag, body.Size),
						Name: validName(body.Name),
					},
				}
				switch field.Type.(type) {
				case *c.PointerType:
					field.Size = 4
				}
				t.Fields = append(t.Fields, field)
				validateSize(field.Type, field.Size, 1)
			case sym.ClassEOS:
				return n + 1
			default:
				panic(fmt.Errorf("support for class %q not yet implemented", body.Class))
			}
		default:
			panic("")
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
		default:
			panic("")
		}
	}
	panic("unreachable")
}

// parseTypedef parses a typedef symbol.
func (p *Parser) parseTypedef(t sym.Type, dims []uint32, tag, name string, size uint32) {
	name = validName(name)
	def := &c.VarDecl{
		Class:   c.Typedef,
		Emitted: false,
		Size:    size,
		Var: c.Var{
			Type: p.parseType(t, dims, tag, size),
			Name: name,
		},
	}

	p.CurOverlay.Typedefs = append(p.CurOverlay.Typedefs, def)
	p.CurOverlay.Types[name] = append(p.CurOverlay.Types[name], def)

	// Map typedefs to struct/union
	ntag := validName(tag)
	switch t.Base() {
	case sym.BaseStruct:
		ttt := p.findStruct(ntag, size, dims, t.Mods())
		for _, d := range ttt.Typedef {
			if d.String() == name {
				if !compareTypedef(d, def) {
					panic("")
				}
				return
			}
		}
		ttt.Typedef = append(ttt.Typedef, def)
	case sym.BaseUnion:
		ttt := p.findUnion(ntag, size, dims, t.Mods())
		for _, d := range ttt.Typedef {
			if d.String() == name {
				if !compareTypedef(d, def) {
					panic("")
				}
				return
			}
		}
		ttt.Typedef = append(ttt.Typedef, def)
	case sym.BaseEnum:
		ttt := p.findEnum(ntag)
		for _, d := range ttt.Typedef {
			if d.String() == name {
				if !compareTypedef(d, def) {
					panic("")
				}
				return
			}
		}
		ttt.Typedef = append(ttt.Typedef, def)
	}
}

// ### [ Helper functions ] ####################################################

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
	p.CurOverlay.Structs = append(p.CurOverlay.Structs, t)
	p.CurOverlay.StructTags[t.Tag] = append(p.CurOverlay.StructTags[t.Tag], t)
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
	for _, structs := range p.CurOverlay.StructTags {
		replaceStructsInSlice(structs, typeRemap)
	}
	replaceStructsInSlice(p.CurOverlay.Structs, typeRemap)
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
	for tag, structs := range p.CurOverlay.StructTags {
		p.CurOverlay.StructTags[tag] = rmNilStructsFromSlice(structs)
	}
	p.CurOverlay.Structs = rmNilStructsFromSlice(p.CurOverlay.Structs)
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
func (p *Parser) findStruct(tag string, size uint32, dims []uint32, mods []sym.Mod) *c.StructType {
	var t *c.StructType = nil
	nameExists := false
	structs, ok := p.CurOverlay.StructTags[tag]

	if size == 0 && p.opts.Verbose {
		fmt.Printf("Unknown struct size %s\n", tag)
	}

	if len(mods) > 1 && mods[0] == sym.ModArray && mods[len(mods)-1] == sym.ModPointer { // struct pointer array
		size = 0
	} else if size > 0 && len(dims) > 0 { // struct array
		div := 1
		for i := 0; i < len(dims); i++ {
			div *= int(dims[i])
		}
		if (size % dims[0]) != 0 {
			panic(fmt.Errorf("incorrect size"))
		}
		size /= uint32(div)
	}

	if ok {
		if size == 0 && len(structs) > 0 {
			t = structs[len(structs)-1]
		}
		nameExists = len(structs) > 0
		for i := len(structs) - 1; i >= 0; i-- {
			tt := structs[i]
			if tt.Size == size {
				t = tt
				break
			}
		}
	}
	if t == nil {
		t = p.emptyStruct(tag, size)
		if nameExists {
			panic("")
		}
		log.Printf("unable to locate struct %q size %d, created empty", tag, size)
	}
	return t
}

// findEmptyStruct returns the struct with the given tag and size.
// It selects the struct which has no fields defined yet, and
// asserts that the type exists.
func findEmptyStruct(p *Parser, tag string, size uint32) *c.StructType {
	var t *c.StructType = nil
	structs, ok := p.CurOverlay.StructTags[tag]
	if ok {
		for i := len(structs) - 1; i >= 0; i-- {
			tt := structs[i]
			// referenced before defined structs have size == 0?
			if tt.Size == 0 {
				tt.Size = size
			}
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
		t = p.emptyStruct(tag, size)
	}
	return t
}

// AddUnion adds the type instance to lists within parser
func (p *Parser) AddUnion(t *c.UnionType) *c.UnionType {
	p.CurOverlay.Unions = append(p.CurOverlay.Unions, t)
	p.CurOverlay.UnionTags[t.Tag] = append(p.CurOverlay.UnionTags[t.Tag], t)
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
	for _, unions := range p.CurOverlay.UnionTags {
		replaceUnionsInSlice(unions, typeRemap)
	}
	replaceUnionsInSlice(p.CurOverlay.Unions, typeRemap)
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
	for tag, unions := range p.CurOverlay.UnionTags {
		p.CurOverlay.UnionTags[tag] = rmNilUnionsFromSlice(unions)
	}
	p.CurOverlay.Unions = rmNilUnionsFromSlice(p.CurOverlay.Unions)
}

func (p *Parser) emptyUnion(tag string, size uint32) *c.UnionType {
	t := &c.UnionType{
		Tag:     tag,
		Size:    size,
		Emitted: false,
	}
	return p.AddUnion(t)
}

// findUnion returns the union with the given tag and size.
func (p *Parser) findUnion(tag string, size uint32, dims []uint32, mods []sym.Mod) *c.UnionType {
	var t *c.UnionType = nil
	nameExists := false
	unions, ok := p.CurOverlay.UnionTags[tag]

	if size == 0 && p.opts.Verbose {
		fmt.Printf("Unknown union size %s\n", tag)
	}

	if len(dims) > 0 || len(mods) > 0 {
		panic(fmt.Errorf("unimplemented")) // TODO
	}
	if size == 0 {
		panic(fmt.Errorf("invalid union %q size %d", tag, size))
	}

	if ok {
		nameExists = len(unions) > 0
		for i := len(unions) - 1; i >= 0; i-- {
			tt := unions[i]
			if tt.Size == size {
				t = tt
				break
			}
		}
	}
	if t == nil {
		t = p.emptyUnion(tag, size)
		if nameExists {
			panic("")
		}
		log.Printf("unable to locate union %q size %d, created empty", tag, size)
	}
	return t
}

// findEmptyUnion returns the union with the given tag and size.
// It selects the union which has no fields defined yet, and
// asserts that the type exists.
func findEmptyUnion(p *Parser, tag string, size uint32) *c.UnionType {
	var t *c.UnionType = nil
	unions, ok := p.CurOverlay.UnionTags[tag]
	if ok {
		for i := len(unions) - 1; i >= 0; i-- {
			tt := unions[i]
			// referenced before defined unions have size == 0?
			if tt.Size == 0 {
				tt.Size = size
			}
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
		t = p.emptyUnion(tag, size)
	}
	return t
}

// AddEnum adds the type instance to lists within parser
func (p *Parser) AddEnum(t *c.EnumType) *c.EnumType {
	p.CurOverlay.Enums = append(p.CurOverlay.Enums, t)
	p.CurOverlay.EnumTags[t.Tag] = append(p.CurOverlay.EnumTags[t.Tag], t)
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
	for _, enums := range p.CurOverlay.EnumTags {
		replaceEnumsInSlice(enums, typeRemap)
	}
	replaceEnumsInSlice(p.CurOverlay.Enums, typeRemap)
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
	for tag, enums := range p.CurOverlay.EnumTags {
		p.CurOverlay.EnumTags[tag] = rmNilEnumsFromSlice(enums)
	}
	p.CurOverlay.Enums = rmNilEnumsFromSlice(p.CurOverlay.Enums)
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
	enums, ok := p.CurOverlay.EnumTags[tag]
	if ok {
		nameExists = len(enums) > 0
		for i := len(enums) - 1; i >= 0; i-- {
			tt := enums[i]
			t = tt
		}
	}
	if t == nil {
		t = p.emptyEnum(tag)
		if nameExists {
			panic("")
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
	enums, ok := p.CurOverlay.EnumTags[tag]
	if ok {
		for i := len(enums) - 1; i >= 0; i-- {
			tt := enums[i]
			if len(tt.Members) != 0 {
				continue
			}
			t = tt
		}
	}
	if t == nil {
		t = p.emptyEnum(tag)
	}
	return t
}

// parseType parses the SYM type into the equivalent C type.
func (p *Parser) parseType(t sym.Type, dims []uint32, tag string, size uint32) c.Type {
	u := p.parseBase(t, tag, size, dims)
	return parseMods(u, t.Mods(), dims)
}

// parseBase parses the SYM base type into the equivalent C type.
func (p *Parser) parseBase(t sym.Type, tag string, size uint32, dims []uint32) c.Type {
	tag = validName(tag)
	switch t.Base() {
	case sym.BaseNull:
		return p.CurOverlay.Types["bool"][0]
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
		return p.findStruct(tag, size, dims, t.Mods())
	case sym.BaseUnion:
		return p.findUnion(tag, size, dims, t.Mods())
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
		panic(fmt.Errorf("base type %q not yet supported", t.Base()))
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

func (p *Parser) ReplaceTypedefs(typeRemap map[c.Type]c.Type) {
	for _, typedef := range p.CurOverlay.Types {
		replaceTypedefsInSlice(typedef, typeRemap)
	}
	replaceTypedefsInSlice(p.CurOverlay.Typedefs, typeRemap)
}

func replaceTypedefsInSlice(typedef []*c.VarDecl, typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(typedef); i++ {
		t1, ok := typeRemap[typedef[i]]
		if ok {
			if t1 == nil {
				typedef[i] = nil
			} else {
				typedef[i] = t1.(*c.VarDecl)
			}
		}
	}
}

func rmNilTypedefsFromSlice(typedefs []*c.VarDecl) []*c.VarDecl {
	for i := 0; i < len(typedefs); {
		if typedefs[i] != nil {
			i++
			continue
		}
		if i < len(typedefs)-1 {
			copy(typedefs[i:], typedefs[i+1:])
		}
		typedefs[len(typedefs)-1] = nil
		typedefs = typedefs[:len(typedefs)-1]
	}
	return typedefs
}

func (p *Parser) RmNilTypedefs() {
	for tag, typedefs := range p.CurOverlay.Types {
		p.CurOverlay.Types[tag] = rmNilTypedefsFromSlice(typedefs)
	}
	p.CurOverlay.Typedefs = rmNilTypedefsFromSlice(p.CurOverlay.Typedefs)
}
