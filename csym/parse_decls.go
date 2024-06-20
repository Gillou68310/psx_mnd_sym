package csym

import (
	"fmt"
	"log"

	sym "github.com/mefistotelis/psx_mnd_sym"
	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

// parseSymbol parses a symbol and its associated address.
func (p *Parser) parseSymbol(addr uint32, name string) {
	// TODO: name = validName(name)?
	symbol := &Symbol{
		Addr: addr,
		Name: name,
	}
	p.CurOverlay.Symbols = append(p.CurOverlay.Symbols, symbol)
}

// parseLineNumbers parses a line numbers sequence of symbols.
func (p *Parser) parseLineNumbers(addr uint32, body *sym.SetSLD2, syms []*sym.Symbol) (n int) {
	curLine := Line{
		Path: body.Path,
		Line: body.Line,
	}
	line := &Line{
		Addr: addr,
		Path: curLine.Path,
		Line: curLine.Line,
	}
	p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
	for n = 0; n < len(syms); n++ {
		s := syms[n]
		switch body := s.Body.(type) {
		case *sym.IncSLD:
			curLine.Line++
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
		case *sym.IncSLDByte:
			curLine.Line += uint32(body.Inc)
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
		case *sym.IncSLDWord:
			curLine.Line += uint32(body.Inc)
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
		case *sym.SetSLD:
			curLine.Line = body.Line
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
		case *sym.SetSLD2:
			curLine.Path = body.Path
			curLine.Line = body.Line
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
		case *sym.EndSLD:
			return n + 1
		default:
			panic("")
			// Symbol type not handled by parseLineNumber, re-parse.
			//return n
		}
	}
	panic("unreachable")
}

// emptyFunc creates an empty/dummy function declaration when real one is missing.
func (p *Parser) emptyFunc(name string, addr uint32) *c.FuncDecl {
	f := &c.FuncDecl{
		Addr: addr,
		Var: c.Var{
			Name: name,
			Type: &c.FuncType{RetType: c.Void},
		},
	}
	p.CurOverlay.Funcs = append(p.CurOverlay.Funcs, f)
	p.CurOverlay.FuncNames[name] = append(p.CurOverlay.FuncNames[name], f)
	return f
}

// parseFunc parses a function sequence of symbols.
func (p *Parser) parseFunc(addr uint32, body *sym.FuncStart, syms []*sym.Symbol) (n int) {
	f, funcType := findFunc(p, body.Name, addr)
	// Ignore duplicate function (already parsed).
	if f.LineStart != 0 {
		panic("")
		/*for n = 0; n < len(syms); n++ {
			if _, ok := syms[n].Body.(*sym.FuncEnd); ok {
				return n + 1
			}
		}*/
	}
	f.Path = body.Path
	// Parse function declaration.
	f.LineStart = body.Line
	curLine := Line{
		Path: body.Path,
		Line: body.Line,
	}
	line := &Line{
		Addr: addr,
		Path: curLine.Path,
		Line: curLine.Line,
	}
	p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
	var blocks blockStack
	var curBlock *c.Block
	var depth = 0
	for n = 0; n < len(syms); n++ {
		s := syms[n]
		switch body := s.Body.(type) {
		case *sym.FuncEnd:
			f.LineEnd = body.Line
			return n + 1
		case *sym.BlockStart:
			if curBlock != nil {
				blocks.push(curBlock)
			}
			block := &c.Block{
				LineStart: body.Line,
				Closed:    false,
				Depth:     depth,
			}
			f.Blocks = append(f.Blocks, block)
			curBlock = block
			curBlock.LineStartAddr = s.Hdr.Value
			curLine.Line += body.Line - 1
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
			depth++
		case *sym.BlockEnd:
			curBlock.LineEnd = body.Line
			curBlock.LineEndAddr = s.Hdr.Value
			if !blocks.empty() {
				curBlock = blocks.pop()
			} else {
				curBlock = nil
			}
			curLine.Line += body.Line - 1
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.CurOverlay.Lines = append(p.CurOverlay.Lines, line)
			depth--
		case *sym.Def:
			switch body.Class {
			case sym.ClassSTAT, sym.ClassREG, sym.ClassREGPARM, sym.ClassARG, sym.ClassLABEL, sym.ClassAUTO:
				t := p.parseType(body.Type, nil, "", body.Size)
				v := p.parseLocalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
				if curBlock != nil {
					addLocal(curBlock, v)
				} else {
					addParam(funcType, v)
				}
			default:
				panic("")
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassTPDEF:
				p.parseTypedef(body.Type, body.Dims, body.Tag, body.Name, body.Size)
			case sym.ClassSTAT, sym.ClassREG, sym.ClassREGPARM, sym.ClassARG, sym.ClassAUTO:
				t := p.parseType(body.Type, body.Dims, body.Tag, body.Size)
				v := p.parseLocalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
				if curBlock != nil {
					addLocal(curBlock, v)
				} else {
					addParam(funcType, v)
				}
			default:
				panic("")
			}
		default:
			panic(fmt.Errorf("support for symbol type %T not yet implemented", body))
		}
	}
	panic("unreachable")
}

func tagRefStructUnion(t c.Type) {
	switch tt := t.(type) {
	case *c.StructType:
		tt.IsRef = true
	case *c.UnionType:
		tt.IsRef = true
	case *c.PointerType:
		tagRefStructUnion(tt.Elem)
	case *c.ArrayType:
		tagRefStructUnion(tt.Elem)
	default:
		break
	}
}

// parseLocalDecl parses a local declaration symbol.
func (p *Parser) parseLocalDecl(addr, size uint32, class sym.Class, t c.Type, name string) *c.VarDecl {
	tagRefStructUnion(t)
	validateSize(t, size, 1)
	name = validName(name)
	v := &c.VarDecl{
		Addr:  addr,
		Size:  size,
		Class: parseClass(class),
		Var: c.Var{
			Type: t,
			Name: name,
		},
	}
	return v
}

// TODO: consider rewriting FuncDecl as:
//
//    type FuncDecl struct {
//       *VarDecl
//       Blocks []*Block
//    }

// parseGlobalDecl parses a global declaration symbol.
func (p *Parser) parseGlobalDecl(addr, size uint32, class sym.Class, t c.Type, name string) {
	tagRefStructUnion(t)
	validateSize(t, size, 1)
	name = validName(name)
	if _, ok := t.(*c.FuncType); ok {
		f := &c.FuncDecl{
			Addr: addr,
			Size: size,
			Var: c.Var{
				Type: t,
				Name: name,
			},
		}
		p.CurOverlay.Funcs = append(p.CurOverlay.Funcs, f)
		p.CurOverlay.FuncNames[name] = append(p.CurOverlay.FuncNames[name], f)
		return
	}
	v := &c.VarDecl{
		Addr:  addr,
		Size:  size,
		Class: parseClass(class),
		Var: c.Var{
			Type: t,
			Name: name,
		},
	}
	p.CurOverlay.Vars = append(p.CurOverlay.Vars, v)
	p.CurOverlay.VarNames[name] = append(p.CurOverlay.VarNames[name], v)
}

// parseOverlay parses an overlay symbol.
func (p *Parser) parseOverlay(addr uint32, body *sym.Overlay) {
	overlay := &Overlay{
		Addr:       addr,
		ID:         body.ID,
		Length:     body.Length,
		VarNames:   make(map[string][]*c.VarDecl),
		FuncNames:  make(map[string][]*c.FuncDecl),
		StructTags: make(map[string][]*c.StructType),
		UnionTags:  make(map[string][]*c.UnionType),
		EnumTags:   make(map[string][]*c.EnumType),
		Types:      make(map[string][]*c.VarDecl),
	}
	p.Overlays = append(p.Overlays, overlay)
	p.overlayIDs[overlay.ID] = overlay
}

// ### [ Helper functions ] ####################################################

// findFunc returns the function with the given name and address.
func findFunc(p *Parser, name string, addr uint32) (*c.FuncDecl, *c.FuncType) {
	name = validName(name)
	var f *c.FuncDecl = nil
	nameExists := false
	funcs, ok := p.CurOverlay.FuncNames[name]
	if ok {
		nameExists = len(funcs) > 0
		for i := 0; i < len(funcs); i++ {
			tf := funcs[i]
			if tf.Addr != addr {
				continue
			}
			if f != nil {
				continue
			}
			f = tf
		}
	}
	if f == nil {
		f = p.emptyFunc(name, addr)
		if nameExists {
			panic("")
			//f.Var.Name = UniqueFuncName(p.CurOverlay.FuncNames, f)
		}
		log.Printf("unable to locate function %q, created void", name)
	}
	funcType, ok := f.Type.(*c.FuncType)
	if !ok {
		panic(fmt.Errorf("invalid function type; expected *c.FuncType, got %T", f.Type))
	}
	return f, funcType
}

// parseClass parses the symbol class into an equivalent C storage class.
func parseClass(class sym.Class) c.StorageClass {
	switch class {
	case sym.ClassAUTO:
		return c.Auto
	case sym.ClassEXT:
		return c.Extern
	case sym.ClassSTAT:
		return c.Static
	case sym.ClassREG:
		return c.Register
	case sym.ClassLABEL:
		return c.Label
	case sym.ClassARG:
		return 0
	case sym.ClassTPDEF:
		return c.Typedef
	case sym.ClassREGPARM:
		return c.Register
	default:
		panic(fmt.Errorf("support for symbol class %v not yet implemented", class))
	}
}

// blockStack is a stack of blocks.
type blockStack []*c.Block

// push pushes the block onto the stack.
func (b *blockStack) push(block *c.Block) {
	*b = append(*b, block)
}

// pop pops the top block of the stack.
func (b *blockStack) pop() *c.Block {
	if b.empty() {
		panic("invalid call to pop; empty stack")
	}
	n := len(*b)
	block := (*b)[n-1]
	*b = (*b)[:n-1]
	return block
}

// empty reports whether the stack is empty.
func (b *blockStack) empty() bool {
	return len(*b) == 0
}

// addLocal adds the local variable to the block if not already present.
func addLocal(block *c.Block, local *c.VarDecl) {
	for _, v := range block.Locals {
		if v.Name == local.Name {
			return
		}
	}
	block.Locals = append(block.Locals, local)
}

// addParam adds the function parameter to the function type if not already
// present.
func addParam(t *c.FuncType, param *c.VarDecl) {
	for _, p := range t.Params {
		if p.Name == param.Name {
			return
		}
	}
	t.Params = append(t.Params, param)
}
