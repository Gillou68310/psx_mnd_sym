package csym

import (
	"fmt"
	"reflect"

	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

// RemoveDuplicateTypes goes through parsed types and marks exact duplicates.
func (p *Parser) RemoveDuplicateTypes() {
	if p.opts.Verbose {
		fmt.Printf("Remove duplicate types...\n")
	}
	p.removeStructsDuplicates()
	p.removeUnionsDuplicates()
	p.removeStructsDuplicates()
}

// removeStructsDuplicates goes through parsed symbols and marks exact duplicates.
func (p *Parser) removeStructsDuplicates() {
	// Create a type replacing map
	typeRemap := make(map[c.Type]c.Type)
	for _, structs := range p.StructTags {
		for i := 0; i < len(structs); i++ {
			t1 := structs[i]
			if _, ok := typeRemap[t1]; ok {
				continue
			}
			for k := i + 1; k < len(structs); k++ {
				t2 := structs[k]
				if _, ok := typeRemap[t2]; ok {
					continue
				}
				if !reflect.DeepEqual(t2, t1) {
					continue
				}
				typeRemap[t2] = t1
			}
		}
	}
	// Replace the pointers in uses of types within other types and declarations
	p.ReplaceUsedTypes(typeRemap)
	// Replace the pointers on main lists with nil, then remove nil items
	for t2, _ := range typeRemap {
		typeRemap[t2] = nil
	}
	p.ReplaceStructs(typeRemap)
	p.RmNilStructs()
	if p.opts.Verbose {
		fmt.Printf("Removed structs: %d\n", len(typeRemap))
	}
}

// removeUnionsDuplicates goes through parsed symbols and marks exact duplicates.
func (p *Parser) removeUnionsDuplicates() {
	// Create a type replacing map
	typeRemap := make(map[c.Type]c.Type)
	for _, unions := range p.UnionTags {
		for i := 0; i < len(unions); i++ {
			t1 := unions[i]
			if _, ok := typeRemap[t1]; ok {
				continue
			}
			for k := i + 1; k < len(unions); k++ {
				t2 := unions[k]
				if _, ok := typeRemap[t2]; ok {
					continue
				}
				if !reflect.DeepEqual(t2, t1) {
					continue
				}
				typeRemap[t2] = t1
			}
		}
	}
	// Replace the pointers in uses of types within other types and declarations
	p.ReplaceUsedTypes(typeRemap)
	// Replace the pointers on main lists with nil, then remove nil items
	for t2, _ := range typeRemap {
		typeRemap[t2] = nil
	}
	p.ReplaceUnions(typeRemap)
	p.RmNilUnions()
	if p.opts.Verbose {
		fmt.Printf("Removed unions: %d\n", len(typeRemap))
	}
}

// removeEnumsDuplicates goes through parsed symbols and marks exact duplicates.
func (p *Parser) removeEnumsDuplicates() {
	// Create a type replacing map
	typeRemap := make(map[c.Type]c.Type)
	for _, enums := range p.EnumTags {
		for i := 0; i < len(enums); i++ {
			t1 := enums[i]
			if _, ok := typeRemap[t1]; ok {
				continue
			}
			for k := i + 1; k < len(enums); k++ {
				t2 := enums[k]
				if _, ok := typeRemap[t2]; ok {
					continue
				}
				if !reflect.DeepEqual(t2, t1) {
					continue
				}
				typeRemap[t2] = t1
			}
		}
	}
	// Replace the pointers in uses of types within other types and declarations
	p.ReplaceUsedTypes(typeRemap)
	// Replace the pointers on main lists with nil, then remove nil items
	for t2, _ := range typeRemap {
		typeRemap[t2] = nil
	}
	p.ReplaceEnums(typeRemap)
	p.RmNilEnums()
	if p.opts.Verbose {
		fmt.Printf("Removed enums: %d\n", len(typeRemap))
	}
}

// replaceUsedSubtypesInType remaps sub-types within the Type interface.
func replaceUsedSubtypesInType(t c.Type, typeRemap map[c.Type]c.Type) {
	switch tp := t.(type) {
	case *c.PointerType:
		t1, ok := typeRemap[tp.Elem]
		if ok {
			tp.Elem = t1
		}
		replaceUsedSubtypesInType(tp.Elem, typeRemap)
	case *c.ArrayType:
		t1, ok := typeRemap[tp.Elem]
		if ok {
			tp.Elem = t1
		}
		replaceUsedSubtypesInType(tp.Elem, typeRemap)
	case *c.FuncType:
		t1, ok := typeRemap[tp.RetType]
		if ok {
			tp.RetType = t1
		}
		replaceUsedSubtypesInType(tp.RetType, typeRemap)
		for i := 0; i < len(tp.Params); i++ {
			replaceUsedTypesInVar(&tp.Params[i].Var, typeRemap)
		}
	case *c.UnionType:
		for i := 0; i < len(tp.Fields); i++ {
			replaceUsedTypesInVar(&tp.Fields[i].Var, typeRemap)
		}
	}
}

func replaceUsedTypesInVar(v *c.Var, typeRemap map[c.Type]c.Type) {
	t1, ok := typeRemap[v.Type]
	if ok {
		v.Type = t1
	}
	replaceUsedSubtypesInType(v.Type, typeRemap)
}

func (p *Parser) replaceUsedTypesInStructs(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.Structs); i++ {
		t := p.Structs[i]
		for k := 0; k < len(t.Fields); k++ {
			replaceUsedTypesInVar(&t.Fields[k].Var, typeRemap)
		}
		for k := 0; k < len(t.Methods); k++ {
			replaceUsedTypesInVar(&t.Methods[k].Var, typeRemap)
		}
	}
}

func (p *Parser) replaceUsedTypesInUnions(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.Unions); i++ {
		t := p.Unions[i]
		for k := 0; k < len(t.Fields); k++ {
			replaceUsedTypesInVar(&t.Fields[k].Var, typeRemap)
		}
	}
}
func (p *Parser) replaceUsedTypesInTypedefs(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.Typedefs); i++ {
		t := p.Typedefs[i]
		// Do not replace the typedef itself, only uses of types within
		replaceUsedSubtypesInType(t, typeRemap)
	}
}

func (p *Parser) replaceUsedVarTypesInInOverlay(overlay *Overlay, typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(overlay.Vars); i++ {
		t := overlay.Vars[i]
		replaceUsedTypesInVar(&t.Var, typeRemap)
	}
}

func (p *Parser) replaceUsedFuncTypesInInOverlay(overlay *Overlay, typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(overlay.Funcs); i++ {
		t := overlay.Funcs[i]
		replaceUsedTypesInVar(&t.Var, typeRemap)
		for k := 0; k < len(t.Blocks); k++ {
			b := t.Blocks[k]
			for n := 0; n < len(b.Locals); n++ {
				replaceUsedTypesInVar(&b.Locals[n].Var, typeRemap)
			}
		}
	}
}

func (p *Parser) ReplaceUsedTypes(typeRemap map[c.Type]c.Type) {
	p.replaceUsedTypesInStructs(typeRemap)
	p.replaceUsedTypesInUnions(typeRemap)
	p.replaceUsedTypesInTypedefs(typeRemap)
	// Default overlay
	p.replaceUsedVarTypesInInOverlay(p.Overlay, typeRemap)
	p.replaceUsedFuncTypesInInOverlay(p.Overlay, typeRemap)
	// Other overlays
	for _, overlay := range p.Overlays {
		p.replaceUsedVarTypesInInOverlay(overlay, typeRemap)
		p.replaceUsedFuncTypesInInOverlay(overlay, typeRemap)
	}
}

// MakeNamesUnique goes through parsed symbols and renames duplicate names.
func (p *Parser) MakeNamesUnique() {
	if p.opts.Verbose {
		fmt.Printf("Making names unique...\n")
	}
	p.makeStructsUnique()
	p.makeUnionsUnique()
	p.makeEnumsUnique()
	// Default overlay
	p.makeVarNamesUniqueInOverlay(p.Overlay)
	p.makeFuncNamesUniqueInOverlay(p.Overlay)
	// Other overlays
	for _, overlay := range p.Overlays {
		p.makeVarNamesUniqueInOverlay(overlay)
		p.makeFuncNamesUniqueInOverlay(overlay)
	}
}

// makeVarNamesUniqueInOverlay goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeVarNamesUniqueInOverlay(overlay *Overlay) {
	for _, variables := range overlay.VarNames {
		// Do not rename extern declarations, only real variables
		real_len := 0
		for i := 0; i < len(variables); i++ {
			v := variables[i]
			if v.Class == c.Extern {
				continue
			}
			real_len++
		}
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(variables); i++ {
			v := variables[i]
			if v.Class == c.Extern {
				continue
			}
			v.Var.Name = UniqueVarName(overlay.VarNames, v)
		}
	}
}

// makeFuncNamesUniqueInOverlay goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeFuncNamesUniqueInOverlay(overlay *Overlay) {
	for _, funcs := range overlay.FuncNames {
		// Do not rename extern declarations
		real_len := len(funcs)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(funcs); i++ {
			f := funcs[i]
			f.Var.Name = UniqueFuncName(overlay.FuncNames, f)
		}
	}
}

// makeStructsUnique goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeStructsUnique() {
	for _, structs := range p.StructTags {
		real_len := len(structs)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(structs); i++ {
			t := structs[i]
			t.Tag = UniqueStructTag(p.StructTags, t)
		}
	}
}

// makeUnionsUnique goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeUnionsUnique() {
	for _, unions := range p.UnionTags {
		real_len := len(unions)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(unions); i++ {
			t := unions[i]
			t.Tag = UniqueUnionTag(p.UnionTags, t)
		}
	}
}

// makeEnumsUnique goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeEnumsUnique() {
	for _, enums := range p.EnumTags {
		real_len := len(enums)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(enums); i++ {
			t := enums[i]
			t.Tag = UniqueEnumTag(p.EnumTags, t)
		}
	}
}

// UniqueName returns a unique name based on the given name and address.
func UniqueName(name string, addr uint32) string {
	return fmt.Sprintf("%s_addr_%08X", name, addr)
}

// UniqueTag returns a unique tag based on the given tag and duplicate index.
func UniqueTag(tag string, typ string, idx int) string {
	return fmt.Sprintf("%s_duplicate_%s%d", tag, typ, idx)
}

// UniqueVarName returns a unique variable name based on the given variable
// and set of present variables mapped by names.
func UniqueVarName(varNames map[string][]*c.VarDecl, v *c.VarDecl) string {
	newName := v.Var.Name
	newName = UniqueName(newName, v.Addr)
	return newName
}

// UniqueFuncName returns a unique function name based on the given function
// and set of present functions mapped by names.
func UniqueFuncName(funcNames map[string][]*c.FuncDecl, f *c.FuncDecl) string {
	newName := f.Var.Name
	newName = UniqueName(newName, f.Addr)
	return newName
}

// UniqueStructTag returns a unique struct tag based on the given struct
// and set of present structs mapped by tags.
func UniqueStructTag(structTags map[string][]*c.StructType, t *c.StructType) string {
	newTag := t.Tag
	for {
		structs, ok := structTags[newTag]
		if !ok {
			break
		} // the tag is unique - done
		k := SliceIndex(len(structs), func(i int) bool { return structs[i] == t })
		if k < 0 {
			k = len(structs)
		}
		newTag = UniqueTag(newTag, "s", k)
	}
	return newTag
}

// UniqueUnionTag returns a unique union tag based on the given union
// and set of present unions mapped by tags.
func UniqueUnionTag(unionTags map[string][]*c.UnionType, t *c.UnionType) string {
	newTag := t.Tag
	for {
		unions, ok := unionTags[newTag]
		if !ok {
			break
		} // the tag is unique - done
		k := SliceIndex(len(unions), func(i int) bool { return unions[i] == t })
		if k < 0 {
			k = len(unions)
		}
		newTag = UniqueTag(newTag, "u", k)
	}
	return newTag
}

// UniqueEnumTag returns a unique enum tag based on the given enum
// and set of present enums mapped by tags.
func UniqueEnumTag(EnumTags map[string][]*c.EnumType, t *c.EnumType) string {
	newTag := t.Tag
	for {
		enums, ok := EnumTags[newTag]
		if !ok {
			break
		} // the tag is unique - done
		k := SliceIndex(len(enums), func(i int) bool { return enums[i] == t })
		if k < 0 {
			k = len(enums)
		}
		newTag = UniqueTag(newTag, "e", k)
	}
	return newTag
}
