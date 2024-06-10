// Package csym translates Playstation 1 symbol information to C declarations.
package csym

import (
	sym "github.com/mefistotelis/psx_mnd_sym"
	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

// Parser tracks type information used for parsing.
type Parser struct {
	// Type information.

	// Struct maps from struct tag to struct types.
	StructTags map[string][]*c.StructType
	// Unions maps from union tag to union types.
	UnionTags map[string][]*c.UnionType
	// Enums maps from enum tag to enum types.
	EnumTags map[string][]*c.EnumType
	// types maps from type name to underlying type definition.
	Types map[string]c.Type
	// Structs in order of occurrence in SYM file.
	Structs []*c.StructType
	// Unions in order of occurrence in SYM file.
	Unions []*c.UnionType
	// Enums in order of occurrence in SYM file.
	Enums []*c.EnumType
	// Type definitions in order of occurrence in SYM file.
	Typedefs []c.Type
	// Tracks unique enum member names.
	enumMembers map[string]bool

	// Declarations.
	*Overlay // default binary

	// Overlays.
	Overlays []*Overlay
	// overlayIDs maps from overlay ID to overlay.
	overlayIDs map[uint32]*Overlay

	// Current overlay.
	curOverlay *Overlay

	// Option switches.
	opts *sym.Options
}

// NewParser returns a new parser.
func NewParser(opts *sym.Options) *Parser {
	overlay := &Overlay{
		ID:        0,
		Addr:      0,
		Length:    0,
		VarNames:  make(map[string][]*c.VarDecl),
		FuncNames: make(map[string][]*c.FuncDecl),
	}
	parser := &Parser{
		StructTags:  make(map[string][]*c.StructType),
		UnionTags:   make(map[string][]*c.UnionType),
		EnumTags:    make(map[string][]*c.EnumType),
		Types:       make(map[string]c.Type),
		enumMembers: make(map[string]bool),
		Overlay:     overlay,
		overlayIDs:  make(map[uint32]*Overlay),
		curOverlay:  overlay,
		opts:        opts,
	}
	parser.overlayIDs[overlay.ID] = overlay
	return parser
}

// An Overlay is an overlay appended to the end of the executable.
type Overlay struct {
	// Base address at which the overlay is loaded.
	Addr uint32
	// Overlay ID.
	ID uint32
	// Overlay length in bytes.
	Length uint32

	// Variable delcarations.
	Vars []*c.VarDecl
	// Function delcarations.
	Funcs []*c.FuncDecl
	// VarNames maps from variable name to variable declaration.
	VarNames map[string][]*c.VarDecl
	// FuncNames maps from function name to function declaration.
	FuncNames map[string][]*c.FuncDecl

	// Symbols.
	Symbols []*Symbol
	// Source file line numbers.
	Lines []*Line
}

// A Symbol associates a symbol name with an address.
type Symbol struct {
	// Symbol address.
	Addr uint32
	// Symbol name.
	Name string
}

// A Line associates a line number in a source file with an address.
type Line struct {
	// Address.
	Addr uint32
	// Source file name.
	Path string
	// Line number.
	Line uint32
}
