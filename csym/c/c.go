// Package c provides an AST for a subset of C.
package c

import (
	"fmt"
	"strings"
)

// A VarDecl is a variable declaration.
type VarDecl struct {
	// Address, frame pointer delta, or register depending on storage class
	// (optional).
	Addr uint32
	// Size (optional).
	Size uint32
	// Storage class.
	Class StorageClass
	// Underlying variable.
	Var
}

var _Registers = [...]string{
	"$zero",
	"$at",
	"$v0", "$v1",
	"$a0", "$a1", "$a2", "$a3",
	"$t0", "$t1", "$t2", "$t3", "$t4", "$t5", "$t6", "$t7",
	"$s0", "$s1", "$s2", "$s3", "$s4", "$s5", "$s6", "$s7",
	"$t8", "$t9",
	"$k0", "$k1",
	"$gp",
	"$sp",
	"$fp",
	"$ra"}

// String returns the string representation of the variable declaration.
func (v *VarDecl) String() string {
	return v.Name
}

// Def returns the C syntax representation of the definition of the variable
// declaration.
func (v *VarDecl) Def() string {
	buf := &strings.Builder{}
	switch v.Class {
	case Register:
		fmt.Fprintf(buf, "// register: %s\n", _Registers[v.Addr])
	default:
		if v.Addr > 0 {
			if v.Addr > 0x80000000 && v.Addr < 0x90000000 {
				fmt.Fprintf(buf, "// address: 0x%08X\n", v.Addr)
			} else {
				fmt.Fprintf(buf, "// address: %d\n", int32(v.Addr))
			}
		}
	}
	if v.Size > 0 {
		fmt.Fprintf(buf, "// size: 0x%X\n", v.Size)
	}
	if v.Class == 0 {
		fmt.Fprintf(buf, "%s", v.Var)
	} else {
		fmt.Fprintf(buf, "%s %s", v.Class, v.Var)
	}
	return buf.String()
}

//go:generate stringer -linecomment -type StorageClass

// A StorageClass is a storage class.
type StorageClass uint8

// Base types.
const (
	Auto     StorageClass = iota + 1 // auto
	Extern                           // extern
	Static                           // static
	Register                         // register
	Typedef                          // typedef
)

// A FuncDecl is a function declaration.
type FuncDecl struct {
	// Source file.
	Path string
	// Address (optional).
	Addr uint32
	// Size (optional).
	Size uint32
	// Start line number.
	LineStart uint32
	// End line number.
	LineEnd uint32
	// Underlying function variable.
	Var
	// Scope blocks.
	Blocks []*Block
}

// String returns the string representation of the function declaration.
func (f *FuncDecl) String() string {
	return f.Name
}

// Def returns the C syntax representation of the definition of the function
// declaration.
func (f *FuncDecl) Def() string {
	// TODO: Print storage class.
	buf := &strings.Builder{}
	fmt.Fprintf(buf, "// path: %s\n", f.Path)
	if f.Addr > 0 {
		fmt.Fprintf(buf, "// address: 0x%08X\n", f.Addr)
	}
	if f.Size > 0 {
		fmt.Fprintf(buf, "// size: 0x%X\n", f.Size)
	}
	fmt.Fprintf(buf, "// line start: %d\n", f.LineStart)
	fmt.Fprintf(buf, "// line end:   %d\n", f.LineEnd)
	if len(f.Blocks) == 0 {
		fmt.Fprintf(buf, "%s;", f.Var)
		return buf.String()
	}
	fmt.Fprintf(buf, "%s\n", f.Var)
	for i, block := range f.Blocks {
		indent := strings.Repeat("\t", block.Depth)
		fmt.Fprintf(buf, "%s{ //line: %d\n", indent, block.LineStart)
		for _, local := range block.Locals {
			indent := strings.Repeat("\t", block.Depth+1)
			l := strings.Replace(local.Def(), "\n", "\n"+indent, -1)
			fmt.Fprintf(buf, "%s%s;\n", indent, l)
		}
		if i < len(f.Blocks)-1 {
			for j := i; j >= 0; j-- {
				if f.Blocks[i+1].Depth <= f.Blocks[j].Depth && !f.Blocks[j].Closed {
					f.Blocks[j].Closed = true
					indent := strings.Repeat("\t", f.Blocks[j].Depth)
					fmt.Fprintf(buf, "%s} //line: %d\n", indent, f.Blocks[j].LineEnd)
				}
			}
		}
	}
	for i := len(f.Blocks) - 1; i >= 0; i-- {
		if !f.Blocks[i].Closed {
			f.Blocks[i].Closed = true
			indent := strings.Repeat("\t", f.Blocks[i].Depth)
			fmt.Fprintf(buf, "%s} //line: %d\n", indent, f.Blocks[i].LineEnd)
		}
	}
	return buf.String()
}

// A Block encapsulates a block scope.
type Block struct {
	// Start line number (relative to the function).
	LineStart uint32
	// End line number (relative to the function).
	LineEnd uint32
	// Local variables.
	Locals []*VarDecl
	Closed bool
	Depth  int
}
