// The sym_dump tool converts Playstation 1 MND/SYM files to C headers (*.sym ->
// *.h) and scripts for importing symbol information into IDA.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	sym "github.com/mefistotelis/psx_mnd_sym"
	"github.com/mefistotelis/psx_mnd_sym/csym"
	"github.com/pkg/errors"
)

// usage prints usage information.
func usage() {
	const use = `
Convert Playstation 1 MND/SYM files to C headers (*.sym -> *.h) and scripts for importing symbol information into IDA.
`
	fmt.Println(use[1:])
	flag.PrintDefaults()
}

// Default output directory.
const dumpDir = "_dump_"

func main() {
	// Command line flags.
	var (
		// Output C types and declarations.
		outputC bool
		// Output directory.
		outputDir string
		// Output IDA scripts.
		outputIDA bool
		// Split output into source files.
		splitSrc bool
		// Output C types.
		outputTypes bool
		// Output symbols.
		outputSyms bool
		// Verbosity level.
		opts sym.Options
	)
	flag.BoolVar(&outputC, "c", false, "output C types and declarations")
	flag.StringVar(&outputDir, "dir", dumpDir, "output directory")
	flag.BoolVar(&outputIDA, "ida", false, "output IDA scripts")
	flag.BoolVar(&splitSrc, "src", false, "split output into source files")
	flag.BoolVar(&outputTypes, "types", false, "output C types")
	flag.BoolVar(&opts.Verbose, "v", false, "show verbose messages")
	flag.BoolVar(&outputSyms, "s", false, "output symbols")
	flag.Usage = usage
	flag.Parse()

	// Parse SYM files.
	for _, path := range flag.Args() {
		// Parse SYM file.
		f, err := sym.ParseFile(path, &opts)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		switch {
		case outputC, outputIDA, outputSyms, outputTypes:
			// Parse C types and declarations.
			p := csym.NewParser(&opts)
			p.ParseTypesDecls(f.Syms)

			// Default overlay
			p.CurOverlay = p.Overlay
			p.RemoveDuplicateTypes(p.Overlay)
			// Other overlays
			for _, overlay := range p.Overlays {
				p.CurOverlay = overlay
				p.RemoveDuplicateTypes(overlay)
				p.RemoveDuplicateTypes(p.Overlay)
			}

			// Default overlay
			p.CurOverlay = p.Overlay
			p.MakeNamesUnique()
			// Other overlays
			for _, overlay := range p.Overlays {
				p.CurOverlay = overlay
				p.MakeNamesUnique()
			}

			// Output once for each files if not in merge mode.
			if err := dump(p, outputDir, outputC, outputTypes, outputIDA, splitSrc, outputSyms); err != nil {
				log.Fatalf("%+v", err)
			}
		default:
			// Output in Psy-Q DUMPSYM.EXE format.
			// Note, we never merge the Psy-Q output.
			fmt.Print(f)
		}
	}
}

// dump dumps the declarations of the parser to the given output directory, in
// the format specified.
func dump(p *csym.Parser, outputDir string, outputC, outputTypes, outputIDA, splitSrc, outputSyms bool) error {
	if err := initOutputDir(outputDir); err != nil {
		return errors.WithStack(err)
	}

	switch {
	case outputSyms:
		if err := dumpSymbols(p, outputDir); err != nil {
			return errors.WithStack(err)
		}
	case outputC:
		// Output C types and declarations.
		if err := dumpTypes(p, outputDir); err != nil {
			return errors.WithStack(err)
		}
		if splitSrc {
			if err := dumpSourceFiles(p, outputDir); err != nil {
				return errors.WithStack(err)
			}
		} else {
			if err := dumpDecls(p, outputDir); err != nil {
				return errors.WithStack(err)
			}
		}
	case outputTypes:
		// Output C types.
		if err := dumpTypes(p, outputDir); err != nil {
			return errors.WithStack(err)
		}
	case outputIDA:
		// Output IDA scripts.
		if err := dumpIDAScripts(p, outputDir); err != nil {
			return errors.WithStack(err)
		}
		// Delete bool and __int64 types as they cause issues with IDA.
		/*delete(p.CurOverlay.Types, "bool")
		for i, def := range p.CurOverlay.Typedefs {
			if v, ok := def.(*c.VarDecl); ok {
				if v.Name == "__int64" {
					defs := append(p.CurOverlay.Typedefs[:i], p.CurOverlay.Typedefs[i+1:]...)
					p.CurOverlay.Typedefs = defs
					break
				}
			}
		}
		delete(p.CurOverlay.Types, "__int64")
		if err := dumpTypes(p, outputDir); err != nil {
			return errors.WithStack(err)
		}*/
	}
	return nil
}

// initOutputDir initializes the output directory.
func initOutputDir(outputDir string) error {
	// Only remove output directory if set to default. Otherwise, let user remove
	// output directory as a safety precaution.
	if outputDir == dumpDir {
		if err := os.RemoveAll(outputDir); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
