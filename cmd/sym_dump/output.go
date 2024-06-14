package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mefistotelis/psx_mnd_sym/csym"
	"github.com/mefistotelis/psx_mnd_sym/csym/c"
	"github.com/pkg/errors"
	"github.com/rickypai/natsort"
)

// --- [ Type definitions ] ----------------------------------------------------

// Type definitions header file name.
const typesName = "types.h"

func dumpUnion(u *c.UnionType, f *os.File) error {
	if u.Emitted {
		return nil
	}
	for _, g := range u.Fields {
		switch t := g.Type.(type) {
		case *c.StructType:
			dumpStruct(t, f)
		case *c.ArrayType:
			switch tt := t.Elem.(type) {
			case *c.StructType:
				dumpStruct(tt, f)
			default:
				break
			}
		case *c.UnionType:
			dumpUnion(t, f)
		default:
			break
		}
	}
	u.Emitted = true
	if _, err := fmt.Fprintf(f, "%s;\n\n", u.Def()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func dumpStruct(s *c.StructType, f *os.File) error {
	if s.Emitted {
		return nil
	}
	for _, g := range s.Fields {
		switch t := g.Type.(type) {
		case *c.StructType:
			dumpStruct(t, f)
		case *c.ArrayType:
			switch tt := t.Elem.(type) {
			case *c.StructType:
				dumpStruct(tt, f)
			default:
				break
			}
		case *c.UnionType:
			dumpUnion(t, f)
		default:
			break
		}
	}
	s.Emitted = true
	if _, err := fmt.Fprintf(f, "%s\n\n", s.Def()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// dumpTypes outputs the type information recorded by the parser to a C header
// stored in the output directory.
func dumpTypes(p *csym.Parser, outputDir string) error {
	// Create output file.
	typesPath := filepath.Join(outputDir, typesName)
	fmt.Println("creating:", typesPath)
	f, err := os.Create(typesPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	// Print predeclared identifiers.
	if def, ok := p.Overlay.Types["bool"]; ok {
		if _, err := fmt.Fprintf(f, "%s;\n", def[0].Def()); err != nil {
			return errors.WithStack(err)
		}
	}
	fmt.Fprintf(f, "typedef signed char s8;\n")
	fmt.Fprintf(f, "typedef unsigned char u8;\n")
	fmt.Fprintf(f, "typedef short s16;\n")
	fmt.Fprintf(f, "typedef unsigned short u16;\n")
	fmt.Fprintf(f, "typedef long s32;\n")
	fmt.Fprintf(f, "typedef unsigned long u32;\n")
	fmt.Fprintf(f, "typedef float f32;\n\n")

	// Print enums.
	for _, t := range p.Overlay.Enums {
		if _, err := fmt.Fprintf(f, "%s;\n\n", t.Def()); err != nil {
			return errors.WithStack(err)
		}
	}
	// Print structs.
	for _, t := range p.Overlay.Structs {
		dumpStruct(t, f)
	}
	// Print unions.
	for _, t := range p.Overlay.Unions {
		dumpUnion(t, f)
	}
	// Print typedefs.
	for _, def := range p.Overlay.Typedefs {
		if !def.Emitted {
			if _, err := fmt.Fprintf(f, "%s;\n\n", def.Def()); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	for _, overlay := range p.Overlays {
		// Create output file.
		typeName := fmt.Sprintf("types_%x.h", overlay.ID)
		typesPath := filepath.Join(outputDir, typeName)
		fmt.Println("creating:", typesPath)
		f, err := os.Create(typesPath)
		if err != nil {
			return errors.Wrapf(err, "unable to create declarations header %q", typesPath)
		}
		defer f.Close()
		// Print predeclared identifiers.
		if def, ok := overlay.Types["bool"]; ok {
			if _, err := fmt.Fprintf(f, "%s;\n", def[0].Def()); err != nil {
				return errors.WithStack(err)
			}
		}
		fmt.Fprintf(f, "typedef signed char s8;\n")
		fmt.Fprintf(f, "typedef unsigned char u8;\n")
		fmt.Fprintf(f, "typedef short s16;\n")
		fmt.Fprintf(f, "typedef unsigned short u16;\n")
		fmt.Fprintf(f, "typedef long s32;\n")
		fmt.Fprintf(f, "typedef unsigned long u32;\n")
		fmt.Fprintf(f, "typedef float f32;\n\n")

		// Print enums.
		fmt.Fprintf(f, "// Overlay %d\n", overlay.ID)
		for _, t := range overlay.Enums {
			if _, err := fmt.Fprintf(f, "%s;\n\n", t.Def()); err != nil {
				return errors.WithStack(err)
			}
		}
		// Print structs.
		for _, t := range overlay.Structs {
			dumpStruct(t, f)
		}
		// Print unions.
		for _, t := range overlay.Unions {
			dumpUnion(t, f)
		}
		// Print typedefs.
		for _, def := range overlay.Typedefs {
			if !def.Emitted {
				if _, err := fmt.Fprintf(f, "%s;\n\n", def.Def()); err != nil {
					return errors.WithStack(err)
				}
			}
		}
	}
	return nil
}

// --- [ Global declarations ] -------------------------------------------------

const (
	// Declarations header file name.
	declsName = "decls.h"
	// Overlay header file name format string.
	overlayNameFormat = "overlay_%x.h"
	symbolNameFormat  = "symbols.%x.txt"
)

// dumpDecls outputs the declarations recorded by the parser to C headers stored
// in the output directory.
func dumpDecls(p *csym.Parser, outputDir string) error {
	// Create output file.
	declsPath := filepath.Join(outputDir, declsName)
	fmt.Println("creating:", declsPath)
	f, err := os.Create(declsPath)
	if err != nil {
		return errors.Wrapf(err, "unable to create declarations header %q", declsPath)
	}
	defer f.Close()
	// Store declarations of default binary.
	if err := dumpOverlay(f, p.Overlay); err != nil {
		return errors.WithStack(err)
	}
	// Store declarations of overlays.
	for _, overlay := range p.Overlays {
		overlayName := fmt.Sprintf(overlayNameFormat, overlay.ID)
		overlayPath := filepath.Join(outputDir, overlayName)
		fmt.Println("creating:", overlayPath)
		f, err := os.Create(overlayPath)
		if err != nil {
			return errors.Wrapf(err, "unable to create overlay header %q", overlayPath)
		}
		defer f.Close()
		if err := dumpOverlay(f, overlay); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func dumpSymbols(p *csym.Parser, outputDir string) error {
	// Create output file.
	symbolName := fmt.Sprintf(symbolNameFormat, p.Overlay.ID)
	symbolPath := filepath.Join(outputDir, symbolName)
	fmt.Println("creating:", symbolPath)
	f, err := os.Create(symbolPath)
	if err != nil {
		return errors.Wrapf(err, "unable to create symbol file %q", symbolPath)
	}
	defer f.Close()
	// Store symbols for default binary.
	if err := dumpOverlaySymbols(f, p.Overlay); err != nil {
		return errors.WithStack(err)
	}
	// Store symbols for overlays.
	for _, overlay := range p.Overlays {
		symbolName := fmt.Sprintf(symbolNameFormat, overlay.ID)
		symbolPath := filepath.Join(outputDir, symbolName)
		fmt.Println("creating:", symbolPath)
		f, err := os.Create(symbolPath)
		if err != nil {
			return errors.Wrapf(err, "unable to create symbol file %q", symbolPath)
		}
		defer f.Close()
		if err := dumpOverlaySymbols(f, overlay); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func dumpOverlaySymbols(w io.Writer, overlay *csym.Overlay) error {
	if overlay.Addr != 0 || overlay.ID != 0 || overlay.Length != 0 {
		if _, err := fmt.Fprintf(w, "\n//Overlay_%X = 0x%08X; // size: 0x%X\n", overlay.ID, overlay.Addr, overlay.Length); err != nil {
			return errors.WithStack(err)
		}
	}
	sort.Slice(overlay.Symbols, func(i, j int) bool {
		return overlay.Symbols[i].Addr < overlay.Symbols[j].Addr
	})
	var last_path = ""
	// Print symbols.
	for _, s := range overlay.Symbols {
		var v = overlay.VarNames[s.Name]
		var f = overlay.FuncNames[s.Name]

		for _, ff := range f {
			if ff.Addr == s.Addr && ff.Path != last_path {
				if _, err := fmt.Fprintf(w, "//%s\n", ff.Path); err != nil {
					return errors.WithStack(err)
				}
				last_path = ff.Path
				break
			}
		}

		if strings.HasSuffix(s.Name, "_size") && s.Addr < 0x80000000 ||
			strings.HasSuffix(s.Name, "_org") ||
			strings.HasSuffix(s.Name, "_orgend") ||
			strings.HasSuffix(s.Name, "_obj") ||
			strings.HasSuffix(s.Name, "_objend") {
			if _, err := fmt.Fprintf(w, "//"); err != nil {
				return errors.WithStack(err)
			}
		}
		if _, err := fmt.Fprintf(w, "%s = 0x%08X;", s.Name, s.Addr); err != nil {
			return errors.WithStack(err)
		}
		for _, vv := range v {
			if vv.Addr == s.Addr && vv.Size != 0 {
				if _, err := fmt.Fprintf(w, " // size:0x%X", vv.Size); err != nil {
					return errors.WithStack(err)
				}
				break
			}
		}

		for _, ff := range f {
			if ff.Addr == s.Addr {
				if _, err := fmt.Fprintf(w, " // type:func"); err != nil {
					return errors.WithStack(err)
				}
				break
			}
		}

		if _, err := fmt.Fprintf(w, "\n"); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// dumpOverlay outputs the declarations of the overlay, writing to w.
func dumpOverlay(w io.Writer, overlay *csym.Overlay) error {
	// Add types.h include directory.
	if _, err := fmt.Fprintf(w, "#include %q\n\n", typesName); err != nil {
		return errors.WithStack(err)
	}
	if overlay.Addr != 0 || overlay.ID != 0 || overlay.Length != 0 {
		if _, err := fmt.Fprintf(w, "// === [ Overlay ID %x ] ===\n\n", overlay.ID); err != nil {
			return errors.WithStack(err)
		}
	}
	// Print variable declarations.
	sort.Slice(overlay.Vars, func(i, j int) bool {
		return overlay.Vars[i].Addr < overlay.Vars[j].Addr
	})
	for _, v := range overlay.Vars {
		if _, err := fmt.Fprintf(w, "%s;\n\n", v.Def()); err != nil {
			return errors.WithStack(err)
		}
	}
	// Print function declarations.
	sort.Slice(overlay.Funcs, func(i, j int) bool {
		return overlay.Funcs[i].Addr < overlay.Funcs[j].Addr
	})
	for _, f := range overlay.Funcs {
		if _, err := fmt.Fprintf(w, "%s\n\n", f.Def()); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// --- [ Source files ] --------------------------------------------------------

// A SourceFile is a source file.
type SourceFile struct {
	// Source file path.
	Path string
	// Variable declarations.
	vars []*c.VarDecl
	// Function declarations.
	funcs []*c.FuncDecl
}

// dumpSourceFiles outputs the source files recorded by the parser to the output
// directory.
func dumpSourceFiles(p *csym.Parser, outputDir string) error {
	srcs := getSourceFiles(p)
	for _, src := range srcs {
		// Create source file directory.
		path := strings.ToLower(src.Path)
		path = strings.Replace(path, `\`, "/", -1)
		if strings.HasPrefix(path[1:], ":/") {
			path = path[len("c:/"):]
		}
		path = filepath.Join(outputDir, path)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.WithStack(err)
		}
		fmt.Println("creating:", path)
		f, err := os.Create(path)
		if err != nil {
			return errors.WithStack(err)
		}
		defer f.Close()
		if err := dumpSourceFile(f, src); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// dumpSourceFile outputs the declarations of the source file, writing to w.
func dumpSourceFile(w io.Writer, src *SourceFile) error {
	if _, err := fmt.Fprintf(w, "// %s\n\n", src.Path); err != nil {
		return errors.WithStack(err)
	}
	// Add types.h include directory.
	if _, err := fmt.Fprintf(w, "#include %q\n\n", typesName); err != nil {
		return errors.WithStack(err)
	}
	// Handle duplicate identifiers.
	names := make(map[string]bool)
	for _, v := range src.vars {
		if names[v.Name] {
			v.Name = csym.UniqueName(v.Name, v.Addr)
		}
		names[v.Name] = true
	}
	for _, f := range src.funcs {
		if names[f.Name] {
			f.Name = csym.UniqueName(f.Name, f.Addr)
		}
		names[f.Name] = true
	}
	// Print variable declarations.
	for _, v := range src.vars {
		if _, err := fmt.Fprintf(w, "%s;\n\n", v.Def()); err != nil {
			return errors.WithStack(err)
		}
	}
	// Print function declarations.
	for _, f := range src.funcs {
		if _, err := fmt.Fprintf(w, "%s\n\n", f.Def()); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// --- [ IDA scripts ] ---------------------------------------------------------

// dumpIDAScripts outputs the declarations recorded by the parser to IDA scripts
// stored in the output directory.
func dumpIDAScripts(p *csym.Parser, outputDir string) error {
	// Create scripts for declarations of default binary.
	if err := dumpIDAOverlay(p.Overlay, outputDir); err != nil {
		return errors.WithStack(err)
	}
	// Create scripts for declarations of overlays.
	for _, overlay := range p.Overlays {
		if err := dumpIDAOverlay(overlay, outputDir); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// IDA script names.
const (
	// Scripts mapping addresses to identifiers.
	idaIdentsName = "make_psx.py"
	// Scripts adding function signatures to identifiers.
	idaFuncsName = "set_funcs.py"
	// Scripts adding global variable types to identifiers.
	idaVarsName = "set_vars.py"
)

// dumpIDAOverlay outputs the declarations of the overlay to IDA scripts.
func dumpIDAOverlay(overlay *csym.Overlay, outputDir string) error {
	// Create scripts for mapping addresses to identifiers.
	dir := outputDir
	if overlay.ID != 0 {
		overlayDir := fmt.Sprintf("overlay_%x", overlay.ID)
		dir = filepath.Join(outputDir, overlayDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.WithStack(err)
		}
	}
	identsPath := filepath.Join(dir, idaIdentsName)
	fmt.Println("creating:", identsPath)
	w, err := os.Create(identsPath)
	if err != nil {
		return errors.Wrapf(err, "unable to create declarations IDA script %q", identsPath)
	}
	defer w.Close()
	for _, f := range overlay.Funcs {
		if _, err := fmt.Fprintf(w, "set_name(0x%08X, %q, SN_NOWARN)\n", f.Addr, f.Name); err != nil {
			return errors.WithStack(err)
		}
	}
	for _, v := range overlay.Vars {
		if _, err := fmt.Fprintf(w, "set_name(0x%08X, %q, SN_NOWARN)\n", v.Addr, v.Name); err != nil {
			return errors.WithStack(err)
		}
	}
	// Create scripts for adding function signatures to identifiers.
	funcsPath := filepath.Join(dir, idaFuncsName)
	fmt.Println("creating:", funcsPath)
	w, err = os.Create(funcsPath)
	if err != nil {
		return errors.Wrapf(err, "unable to create function signatures IDA script %q", funcsPath)
	}
	defer w.Close()
	for _, f := range overlay.Funcs {
		if _, err := fmt.Fprintf(w, "del_items(0x%08X)\n", f.Addr); err != nil {
			return errors.WithStack(err)
		}
		if _, err := fmt.Fprintf(w, "SetType(0x%08X, %q)\n", f.Addr, f.Var); err != nil {
			return errors.WithStack(err)
		}
	}
	// Create scripts adding global variable types to identifiers.
	varsPath := filepath.Join(dir, idaVarsName)
	fmt.Println("creating:", varsPath)
	w, err = os.Create(varsPath)
	if err != nil {
		return errors.Wrapf(err, "unable to create global variables IDA script %q", varsPath)
	}
	defer w.Close()
	for _, v := range overlay.Vars {
		if _, err := fmt.Fprintf(w, "del_items(0x%08X)\n", v.Addr); err != nil {
			return errors.WithStack(err)
		}
		if _, err := fmt.Fprintf(w, "SetType(0x%08X, %q)\n", v.Addr, v.Var); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// ### [ Helper functions ] ####################################################

// getSourceFiles returns the source files recorded by the parser.
func getSourceFiles(p *csym.Parser) []*SourceFile {
	// Record source file information from overlays.
	overlays := append(p.Overlays, p.Overlay)
	// sources maps from source path to source file.
	sources := make(map[string]*SourceFile)
	for _, overlay := range overlays {
		for _, v := range overlay.Vars {
			srcPath := fmt.Sprintf("global_%x.cpp", overlay.ID)
			src, ok := sources[srcPath]
			if !ok {
				src = &SourceFile{
					Path: srcPath,
				}
				sources[srcPath] = src
			}
			src.vars = append(src.vars, v)
		}
		for _, f := range overlay.Funcs {
			srcPath := f.Path
			src, ok := sources[srcPath]
			if !ok {
				src = &SourceFile{
					Path: srcPath,
				}
				sources[srcPath] = src
			}
			src.funcs = append(src.funcs, f)
		}
	}
	var srcs []*SourceFile
	for _, src := range sources {
		srcs = append(srcs, src)
	}
	less := func(i, j int) bool {
		return natsort.Less(srcs[i].Path, srcs[j].Path)
	}
	sort.Slice(srcs, less)
	return srcs
}
