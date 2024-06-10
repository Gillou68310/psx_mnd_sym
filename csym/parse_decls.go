package csym

import (
	"fmt"
	"log"

	sym "github.com/mefistotelis/psx_mnd_sym"
	"github.com/mefistotelis/psx_mnd_sym/csym/c"
)

// ParseDecls parses the symbols into the equivalent C declarations.
func (p *Parser) ParseDecls(syms []*sym.Symbol) {
	if p.opts.Verbose {
		fmt.Printf("Parsing %d symbol tags for declarations...\n", len(syms))
	}
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
			case sym.ClassEXT, sym.ClassSTAT:
				t := p.parseType(body.Type, nil, "")
				p.parseGlobalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
			case sym.ClassMOS, sym.ClassSTRTAG, sym.ClassMOU, sym.ClassUNTAG, sym.ClassTPDEF, sym.ClassENTAG, sym.ClassMOE, sym.ClassFIELD, sym.Class103:
				// nothing to do.
			default:
				panic(fmt.Sprintf("support for symbol class %q not yet implemented", body.Class))
			}
		case *sym.Def2:
			switch body.Class {
			case sym.ClassEXT, sym.ClassSTAT:
				t := p.parseType(body.Type, body.Dims, body.Tag)
				switch t.(type) {
				case *c.PointerType:
					body.Size = 4
				}
				p.parseGlobalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
			case sym.ClassMOS, sym.ClassMOU, sym.ClassTPDEF, sym.ClassMOE, sym.ClassFIELD, sym.ClassEOS:
				// nothing to do.
			default:
				panic(fmt.Sprintf("support for symbol class %q not yet implemented", body.Class))
			}
		case *sym.Overlay:
			p.parseOverlay(s.Hdr.Value, body)
		case *sym.SetOverlay:
			overlay, ok := p.overlayIDs[s.Hdr.Value]
			if !ok {
				panic(fmt.Errorf("unable to locate overlay with ID %x", s.Hdr.Value))
			}
			p.curOverlay = overlay
		default:
			panic(fmt.Sprintf("support for symbol type %T not yet implemented", body))
		}
	}
	if p.opts.Verbose {
		fmt.Printf("Created %d functions, %d global variables\n", len(p.curOverlay.Funcs), len(p.curOverlay.Vars))
	}
}

// parseSymbol parses a symbol and its associated address.
func (p *Parser) parseSymbol(addr uint32, name string) {
	// TODO: name = validName(name)?
	symbol := &Symbol{
		Addr: addr,
		Name: name,
	}
	p.curOverlay.Symbols = append(p.curOverlay.Symbols, symbol)
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
	p.curOverlay.Lines = append(p.curOverlay.Lines, line)
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
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
		case *sym.IncSLDByte:
			curLine.Line += uint32(body.Inc)
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
		case *sym.IncSLDWord:
			curLine.Line += uint32(body.Inc)
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
		case *sym.SetSLD:
			curLine.Line = body.Line
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
		case *sym.SetSLD2:
			curLine.Path = body.Path
			curLine.Line = body.Line
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
		case *sym.EndSLD:
			return n + 1
		default:
			// Symbol type not handled by parseLineNumber, re-parse.
			return n
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
	p.curOverlay.Funcs = append(p.curOverlay.Funcs, f)
	p.curOverlay.FuncNames[name] = append(p.curOverlay.FuncNames[name], f)
	return f
}

// parseFunc parses a function sequence of symbols.
func (p *Parser) parseFunc(addr uint32, body *sym.FuncStart, syms []*sym.Symbol) (n int) {
	f, funcType := findFunc(p, body.Name, addr)
	// Ignore duplicate function (already parsed).
	if f.LineStart != 0 {
		for n = 0; n < len(syms); n++ {
			if _, ok := syms[n].Body.(*sym.FuncEnd); ok {
				return n + 1
			}
		}
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
	p.curOverlay.Lines = append(p.curOverlay.Lines, line)
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
			curLine.Line += body.Line - 1
			line := &Line{
				Addr: s.Hdr.Value,
				Path: curLine.Path,
				Line: curLine.Line,
			}
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
			depth++
		case *sym.BlockEnd:
			curBlock.LineEnd = body.Line
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
			p.curOverlay.Lines = append(p.curOverlay.Lines, line)
			depth--
		case *sym.Def:
			t := p.parseType(body.Type, nil, "")
			v := p.parseLocalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
			if curBlock != nil {
				addLocal(curBlock, v)
			} else {
				addParam(funcType, v)
			}
		case *sym.Def2:
			t := p.parseType(body.Type, body.Dims, body.Tag)
			v := p.parseLocalDecl(s.Hdr.Value, body.Size, body.Class, t, body.Name)
			if curBlock != nil {
				addLocal(curBlock, v)
			} else {
				addParam(funcType, v)
			}
		default:
			panic(fmt.Errorf("support for symbol type %T not yet implemented", body))
		}
	}
	panic("unreachable")
}

// parseLocalDecl parses a local declaration symbol.
func (p *Parser) parseLocalDecl(addr, size uint32, class sym.Class, t c.Type, name string) *c.VarDecl {
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
		p.curOverlay.Funcs = append(p.curOverlay.Funcs, f)
		p.curOverlay.FuncNames[name] = append(p.curOverlay.FuncNames[name], f)
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
	p.curOverlay.Vars = append(p.curOverlay.Vars, v)
	p.curOverlay.VarNames[name] = append(p.curOverlay.VarNames[name], v)
}

// parseOverlay parses an overlay symbol.
func (p *Parser) parseOverlay(addr uint32, body *sym.Overlay) {
	overlay := &Overlay{
		Addr:      addr,
		ID:        body.ID,
		Length:    body.Length,
		VarNames:  make(map[string][]*c.VarDecl),
		FuncNames: make(map[string][]*c.FuncDecl),
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
	funcs, ok := p.curOverlay.FuncNames[name]
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
			f.Var.Name = UniqueFuncName(p.curOverlay.FuncNames, f)
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
		return 0
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
