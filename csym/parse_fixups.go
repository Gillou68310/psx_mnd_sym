package csym

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

func compareStruct(x, y *c.StructType) bool {

	if strings.HasSuffix(x.Tag, "fake") && strings.HasSuffix(y.Tag, "fake") {
	} else if x.Tag != y.Tag {
		return false
	}
	if x.Size != y.Size {
		return false
	}
	if len(x.Fields) != len(y.Fields) {
		return false
	}
	for i := 0; i < len(x.Fields); i++ {
		if x.Fields[i].Name != y.Fields[i].Name {
			return false
		}
		if x.Fields[i].Offset != y.Fields[i].Offset {
			return false
		}
		if x.Fields[i].Size != y.Fields[i].Size {
			return false
		}
		if !compareVar(x.Fields[i].Var, y.Fields[i].Var) {
			return false
		}
	}
	if len(x.Typedef) != 0 && len(y.Typedef) != 0 {
		if len(x.Typedef) != len(y.Typedef) {
			return false
		}
		for i := 0; i < len(x.Typedef); i++ {
			if !compareTypedef(x.Typedef[i], y.Typedef[i]) {
				return false
			}
		}
	}
	return true
}

func compareUnion(x, y *c.UnionType) bool {

	if strings.HasSuffix(x.Tag, "fake") && strings.HasSuffix(y.Tag, "fake") {
	} else if x.Tag != y.Tag {
		return false
	}
	if x.Size != y.Size {
		return false
	}
	if len(x.Fields) != len(y.Fields) {
		return false
	}
	for i := 0; i < len(x.Fields); i++ {
		if x.Fields[i].Name != y.Fields[i].Name {
			return false
		}
		if x.Fields[i].Offset != y.Fields[i].Offset {
			return false
		}
		if x.Fields[i].Size != y.Fields[i].Size {
			return false
		}
		if !compareVar(x.Fields[i].Var, y.Fields[i].Var) {
			return false
		}
	}
	if len(x.Typedef) != 0 && len(y.Typedef) != 0 {
		if len(x.Typedef) != len(y.Typedef) {
			return false
		}
		for i := 0; i < len(x.Typedef); i++ {
			if !compareTypedef(x.Typedef[i], y.Typedef[i]) {
				return false
			}
		}
	}
	return true
}

func compareTypedef(x, y *c.VarDecl) bool {

	if x.Addr != y.Addr {
		return false
	}
	if x.Size != y.Size {
		return false
	}
	if uint32(x.Class) != uint32(y.Class) {
		return false
	}
	if x.Name != y.Name {
		return false
	}
	if reflect.TypeOf(x.Type) != reflect.TypeOf(y.Type) {
		return false
	}
	return true
}

func compareVar(x, y c.Var) bool {
	if x.Name != y.Name {
		return false
	}
	if reflect.TypeOf(x.Type) != reflect.TypeOf(y.Type) {
		return false
	}
	return true
}

func compareEnum(x, y *c.EnumType) bool {
	if strings.HasSuffix(x.Tag, "fake") && strings.HasSuffix(y.Tag, "fake") {
	} else if x.Tag != y.Tag {
		return false
	}
	if len(x.Members) != len(y.Members) {
		return false
	}
	for i := 0; i < len(x.Members); i++ {
		if x.Members[i].Name != y.Members[i].Name {
			return false
		}
		if x.Members[i].Value != y.Members[i].Value {
			return false
		}
	}
	if len(x.Typedef) != 0 && len(y.Typedef) != 0 {
		if len(x.Typedef) != len(y.Typedef) {
			return false
		}
		for i := 0; i < len(x.Typedef); i++ {
			if !compareTypedef(x.Typedef[i], y.Typedef[i]) {
				return false
			}
		}
	}
	return true
}

// RemoveDuplicateTypes goes through parsed types and marks exact duplicates.
func (p *Parser) RemoveDuplicateTypes() {
	if p.opts.Verbose {
		fmt.Printf("Remove duplicate types...\n")
	}
	p.removeStructsDuplicates()
	p.removeUnionsDuplicates()
	p.removeEnumsDuplicates()
	p.removeTypedefsDuplicates()
}

func typedefExists(t *c.VarDecl, l []*c.VarDecl) bool {
	for i := 0; i < len(l); i++ {
		if compareTypedef(t, l[i]) {
			return true
		}
	}
	return false
}

// removeStructsDuplicates goes through parsed symbols and marks exact duplicates.
func (p *Parser) removeStructsDuplicates() {
	// Create a type replacing map
	typeRemap := make(map[c.Type]c.Type)
	for i := 0; i < len(p.CurOverlay.Structs); i++ {
		t1 := p.CurOverlay.Structs[i]
		if _, ok := typeRemap[t1]; ok {
			continue
		}
		for k := i + 1; k < len(p.CurOverlay.Structs); k++ {
			t2 := p.CurOverlay.Structs[k]
			if _, ok := typeRemap[t2]; ok {
				continue
			}
			if !compareStruct(t2, t1) {
				continue
			}
			for l := 0; l < len(t2.Typedef); l++ {
				if !typedefExists(t2.Typedef[l], t1.Typedef) {
					t1.Typedef = append(t1.Typedef, t2.Typedef[l])
				}
			}
			typeRemap[t2] = t1
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
	for i := 0; i < len(p.CurOverlay.Unions); i++ {
		t1 := p.CurOverlay.Unions[i]
		if _, ok := typeRemap[t1]; ok {
			continue
		}
		for k := i + 1; k < len(p.CurOverlay.Unions); k++ {
			t2 := p.CurOverlay.Unions[k]
			if _, ok := typeRemap[t2]; ok {
				continue
			}
			if !compareUnion(t2, t1) {
				continue
			}
			for l := 0; l < len(t2.Typedef); l++ {
				if !typedefExists(t2.Typedef[l], t1.Typedef) {
					t1.Typedef = append(t1.Typedef, t2.Typedef[l])
				}
			}
			typeRemap[t2] = t1
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
	for i := 0; i < len(p.CurOverlay.Enums); i++ {
		t1 := p.CurOverlay.Enums[i]
		if _, ok := typeRemap[t1]; ok {
			continue
		}
		for k := i + 1; k < len(p.CurOverlay.Enums); k++ {
			t2 := p.CurOverlay.Enums[k]
			if _, ok := typeRemap[t2]; ok {
				continue
			}
			if !compareEnum(t2, t1) {
				continue
			}
			for l := 0; l < len(t2.Typedef); l++ {
				if !typedefExists(t2.Typedef[l], t1.Typedef) {
					t1.Typedef = append(t1.Typedef, t2.Typedef[l])
				}
			}
			typeRemap[t2] = t1
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

func (p *Parser) removeTypedefsDuplicates() {
	// Create a type replacing map
	typeRemap := make(map[c.Type]c.Type)
	for _, typedefs := range p.CurOverlay.Types {
		for i := 0; i < len(typedefs); i++ {
			t1 := typedefs[i]
			if _, ok := typeRemap[t1]; ok {
				continue
			}
			for k := i + 1; k < len(typedefs); k++ {
				t2 := typedefs[k]
				if _, ok := typeRemap[t2]; ok {
					continue
				}
				if !compareTypedef(t2, t1) {
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
	p.ReplaceTypedefs(typeRemap)
	p.RmNilTypedefs()
	if p.opts.Verbose {
		fmt.Printf("Removed typedefs: %d\n", len(typeRemap))
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
	for i := 0; i < len(p.CurOverlay.Structs); i++ {
		t := p.CurOverlay.Structs[i]
		for k := 0; k < len(t.Fields); k++ {
			replaceUsedTypesInVar(&t.Fields[k].Var, typeRemap)
		}
		for k := 0; k < len(t.Methods); k++ {
			replaceUsedTypesInVar(&t.Methods[k].Var, typeRemap)
		}
		for k := 0; k < len(t.Typedef); k++ {
			replaceUsedTypesInVar(&t.Typedef[k].Var, typeRemap)
		}
	}
}

func (p *Parser) replaceUsedTypesInUnions(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.CurOverlay.Unions); i++ {
		t := p.CurOverlay.Unions[i]
		for k := 0; k < len(t.Fields); k++ {
			replaceUsedTypesInVar(&t.Fields[k].Var, typeRemap)
		}
		for k := 0; k < len(t.Typedef); k++ {
			replaceUsedTypesInVar(&t.Typedef[k].Var, typeRemap)
		}
	}
}
func (p *Parser) replaceUsedTypesInTypedefs(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.CurOverlay.Typedefs); i++ {
		t := p.CurOverlay.Typedefs[i]
		// Do not replace the typedef itself, only uses of types within
		replaceUsedSubtypesInType(t, typeRemap)
	}
}

func (p *Parser) replaceUsedVarTypes(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.CurOverlay.Vars); i++ {
		t := p.CurOverlay.Vars[i]
		replaceUsedTypesInVar(&t.Var, typeRemap)
	}
}

func (p *Parser) replaceUsedFuncTypes(typeRemap map[c.Type]c.Type) {
	for i := 0; i < len(p.CurOverlay.Funcs); i++ {
		t := p.CurOverlay.Funcs[i]
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
	p.replaceUsedVarTypes(typeRemap)
	p.replaceUsedFuncTypes(typeRemap)
}

// MakeNamesUnique goes through parsed symbols and renames duplicate names.
func (p *Parser) MakeNamesUnique() {
	if p.opts.Verbose {
		fmt.Printf("Making names unique...\n")
	}
	p.makeStructsUnique()
	p.makeUnionsUnique()
	p.makeEnumsUnique()
	p.makeVarNamesUnique()
	p.makeFuncNamesUnique()
}

// makeVarNamesUniqueInOverlay goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeVarNamesUnique() {
	for _, variables := range p.CurOverlay.VarNames {
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
			v.Var.Name = UniqueVarName(p.CurOverlay.VarNames, v)
		}
	}
}

// makeFuncNamesUniqueInOverlay goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeFuncNamesUnique() {
	for _, funcs := range p.CurOverlay.FuncNames {
		// Do not rename extern declarations
		real_len := len(funcs)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(funcs); i++ {
			f := funcs[i]
			f.Var.Name = UniqueFuncName(p.CurOverlay.FuncNames, f)
		}
	}
}

// makeStructsUnique goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeStructsUnique() {
	for _, structs := range p.CurOverlay.StructTags {
		real_len := len(structs)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(structs); i++ {
			t := structs[i]
			t.Tag = UniqueStructTag(p.CurOverlay.StructTags, t)
		}
	}
}

// makeUnionsUnique goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeUnionsUnique() {
	for _, unions := range p.CurOverlay.UnionTags {
		real_len := len(unions)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(unions); i++ {
			t := unions[i]
			t.Tag = UniqueUnionTag(p.CurOverlay.UnionTags, t)
		}
	}
}

// makeEnumsUnique goes through parsed symbols and renames duplicate ones.
func (p *Parser) makeEnumsUnique() {
	for _, enums := range p.CurOverlay.EnumTags {
		real_len := len(enums)
		if real_len < 2 {
			continue
		}
		for i := 0; i < len(enums); i++ {
			t := enums[i]
			t.Tag = UniqueEnumTag(p.CurOverlay.EnumTags, t)
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
