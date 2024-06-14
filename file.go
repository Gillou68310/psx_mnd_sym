// Package sym provides access to Playstation 1 symbol files (*.SYM).
package sym

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"
)

// A File is PS1 symbol file.
type File struct {
	// File header.
	Hdr *FileHeader
	// Symbols.
	Syms []*Symbol
	// Parser options.
	Opts *Options
}

// String returns the string representation of the symbol file.
func (f *File) String() string {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, f.Hdr)
	var line int
	for _, sym := range f.Syms {
		bodyStr := sym.Body.String()
		switch body := sym.Body.(type) {
		case *IncSLD:
			if line == 0 {
				panic(fmt.Errorf("cannot use IncSLD symbol before associated SetSLD symbol"))
			}
			line++
			bodyStr = fmt.Sprintf("Inc SLD linenum (to %d)", line)
		case *IncSLDByte:
			if line == 0 {
				panic(fmt.Errorf("cannot use IncSLDByte symbol before associated SetSLD symbol"))
			}
			line += int(body.Inc)
			bodyStr = fmt.Sprintf("Inc SLD linenum by byte %d (to %d)", body.Inc, line)
		case *IncSLDWord:
			if line == 0 {
				panic(fmt.Errorf("cannot use IncSLDWord symbol before associated SetSLD symbol"))
			}
			line += int(body.Inc)
			bodyStr = fmt.Sprintf("Inc SLD linenum by word %d (to %d)", body.Inc, line)
		case *SetSLD:
			line = int(body.Line)
		case *SetSLD2:
			line = int(body.Line)
		}
		if len(bodyStr) == 0 {
			// Symbol without body.
			fmt.Fprintf(buf, "%06x: %s\n", sym.Offset, sym.Hdr)

		} else {
			fmt.Fprintf(buf, "%06x: %s %s\n", sym.Offset, sym.Hdr, bodyStr)
		}
	}
	return buf.String()
}

// A FileHeader is a PS1 symbol file header.
type FileHeader struct {
	// File signature; MND.
	Signature [3]byte `struc:"[3]byte"`
	// File format version.
	Version uint8 `struc:"uint8"`
	// Target unit.
	TargetUnit uint32 `struc:"uint32,little"`
}

// String returns the string representation of the symbol file header.
func (hdr *FileHeader) String() string {
	const format = `
Header : %s version %d
Target unit %d`
	return fmt.Sprintf(format, hdr.Signature, hdr.Version, hdr.TargetUnit)
}

// ParseFile parses the given PS1 symbol file.
func ParseFile(path string, opts *Options) (*File, error) {
	if opts.Verbose {
		fmt.Printf("Opening '%s'...\n", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()
	return Parse(f, opts)
}

// ParseBytes parses the given PS1 symbol file, reading from b.
func ParseBytes(b []byte, opts *Options) (*File, error) {
	return Parse(bytes.NewReader(b), opts)
}

// Parse parses the given PS1 symbol file, reading from r.
func Parse(r io.Reader, opts *Options) (*File, error) {
	// Parse file header.
	f := &File{}
	br := bufio.NewReader(r)
	hdr, err := parseFileHeader(br)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	f.Hdr = hdr
	f.Opts = opts

	if f.Opts.Verbose {
		fmt.Printf("Parsing flattened tags...\n")
	}
	// Parse symbols.
	offset := 0
	offset += binary.Size(*f.Hdr)
	for {
		sym, err := parseSymbol(br)
		if err != nil {
			if errors.Cause(err) == io.EOF {
				break
			}
			return f, errors.WithStack(err)
		}
		sym.Offset = offset
		f.Syms = append(f.Syms, sym)
		offset += sym.Size()
	}
	if f.Opts.Verbose {
		fmt.Printf("Created %d symbol tags.\n", len(f.Syms))
	}
	return f, nil
}

// parseFileHeader parses and returns a PS1 symbol file header.
func parseFileHeader(r io.Reader) (*FileHeader, error) {
	hdr := &FileHeader{}
	if err := struc.Unpack(r, hdr); err != nil {
		return nil, errors.WithStack(err)
	}
	// Verify Smacker signature.
	switch string(hdr.Signature[:]) {
	case "MND":
		// valid signature.
	default:
		return nil, errors.Errorf(`invalid SYM signature; expected "MND", got %q`, string(hdr.Signature[:]))
	}
	return hdr, nil
}
