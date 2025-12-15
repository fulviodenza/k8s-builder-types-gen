package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// builderGenerator holds the state for generating builder code
type builderGenerator struct {
	pkg     string
	imports []string
	types   map[string]*ast.TypeSpec
}

func main() {
	var inputDir string
	var outputDir string

	flag.StringVar(&inputDir, "input-dir", "", "Directory containing API types")
	flag.StringVar(&outputDir, "output-dir", "", "Directory for generated code")
	flag.Parse()

	if inputDir == "" || outputDir == "" {
		fmt.Println("input-dir and output-dir are required")
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Process .go files in input directory
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		return processFile(path, outputDir)
	})

	if err != nil {
		log.Fatalf("Error processing files: %v", err)
	}
}

func processFile(inputPath, outputDir string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input file: %w", err)
	}

	bg := &builderGenerator{
		types: make(map[string]*ast.TypeSpec),
	}

	if err := bg.parseSource(string(content)); err != nil {
		return fmt.Errorf("parsing source: %w", err)
	}

	// If no types with +builder tag found, skip file
	if len(bg.types) == 0 {
		return nil
	}

	output, err := bg.generate() // core function generation
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	baseFileName := filepath.Base(inputPath)
	outputFileName := strings.TrimSuffix(baseFileName, ".go") + "_builder.go"
	outputPath := filepath.Join(outputDir, outputFileName)

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	log.Printf("Generated builder code for %s in %s", inputPath, outputPath)
	return nil
}

func (bg *builderGenerator) parseSource(src string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing source: %w", err)
	}

	bg.pkg = file.Name.Name

	// Collect imports, this has still some issue and requires file type to have v1 instead of metav1 for example
	// and needs a manual change to not make the import be deleted automatically by the linter.
	for _, imp := range file.Imports {
		bg.imports = append(bg.imports, imp.Path.Value)
	}

	// Find types with +builder tag
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			hasBuilderTag := false
			if genDecl.Doc != nil {
				for _, comment := range genDecl.Doc.List {
					if strings.Contains(comment.Text, "+builder") {
						hasBuilderTag = true
						break
					}
				}
			}

			if hasBuilderTag {
				bg.types[typeSpec.Name.Name] = typeSpec
			}
		}
	}

	return nil
}

func (bg *builderGenerator) generate() ([]byte, error) {
	var buf bytes.Buffer

	// Package declaration
	fmt.Fprintf(&buf, "package %s\n\n", bg.pkg)

	// Imports
	if len(bg.imports) > 0 {
		fmt.Fprintln(&buf, "import (")
		for _, imp := range bg.imports {
			fmt.Fprintf(&buf, "\t%s\n", imp)
		}
		fmt.Fprintln(&buf, ")")
		fmt.Fprintln(&buf)
	}

	// Generate builder functions for each type
	for typeName, typeSpec := range bg.types {
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		// New functions
		fmt.Fprintf(&buf, "// New%s returns a %s object with the given options\n", typeName, typeName)
		parentTypeName := typeName
		fmt.Fprintf(&buf, "func New%s(opts ...func(*%s)) *%s {\n", typeName, parentTypeName, parentTypeName)
		fmt.Fprintf(&buf, "\tobj := &%s{\n", typeName)
		if !strings.Contains(typeName, "Spec") && !strings.Contains(typeName, "Status") {
			fmt.Fprintf(&buf, "\t\tTypeMeta: v1.TypeMeta{\n")
			fmt.Fprintf(&buf, "\t\t\tKind:       %q,\n", parentTypeName)
			fmt.Fprintf(&buf, "\t\t\tAPIVersion: %q,\n", "stack.civo.com/v1alpha1")
			fmt.Fprintf(&buf, "\t\t},\n")
		}
		fmt.Fprintln(&buf, "\t}")
		fmt.Fprintln(&buf)

		fmt.Fprintln(&buf, "\tfor _, f := range opts {")
		fmt.Fprintln(&buf, "\t\tf(obj)")
		fmt.Fprintln(&buf, "\t}")
		fmt.Fprintln(&buf)
		fmt.Fprintln(&buf, "\treturn obj")
		fmt.Fprintln(&buf, "}")
		fmt.Fprintln(&buf)

		// Generate With* functions for fields
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				// Handle embedded fields
				switch typeExpr := field.Type.(type) {
				case *ast.SelectorExpr:
					bg.generateEmbeddedFieldFuncs(&buf, typeName, typeExpr)
				}
				continue
			}

			for _, name := range field.Names {
				fieldType := renderType(field.Type)
				fmt.Fprintf(&buf, "// With%s sets the %s of a %s\n", name, name, typeName)
				fmt.Fprintf(&buf, "func With%s(%s %s) func(*%s) {\n", name, strings.ToLower(name.Name), fieldType, typeName)
				fmt.Fprintf(&buf, "\treturn func(obj *%s) {\n", typeName)
				fmt.Fprintf(&buf, "\t\tobj.%s = %s\n", name, strings.ToLower(name.Name))
				fmt.Fprintf(&buf, "\t}\n")
				fmt.Fprintf(&buf, "}\n\n")
			}
		}
	}

	return buf.Bytes(), nil
}

func (bg *builderGenerator) generateEmbeddedFieldFuncs(buf *bytes.Buffer, typeName string, typeExpr *ast.SelectorExpr) {
	if typeExpr.Sel.Name == "ObjectMeta" {
		// WithName
		fmt.Fprintf(buf, "// WithName sets the name of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithName(name string) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.Name = name\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")

		// WithNamespace
		fmt.Fprintf(buf, "// WithNamespace sets the namespace of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithNamespace(namespace string) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.Namespace = namespace\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")

		// WithLabels
		fmt.Fprintf(buf, "// WithLabel sets a label of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithLabel(k, v string) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.Labels[k] = v\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")

		// WithAnnotations
		fmt.Fprintf(buf, "// WithAnnotation sets an annotation of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithAnnotation(k, v string) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.Annotations[k] = v\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")

		// WithFinalizers
		fmt.Fprintf(buf, "// WithFinalizer sets the finalizers of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithFinalizer(f string) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.Finalizers = append(obj.Finalizers, f)\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")

		// WithCreationTimestamp
		fmt.Fprintf(buf, "// WithCreationTimestamp sets the deletion timestamp of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithCreationTimestamp(timestamp v1.Time) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.CreationTimestamp = timestamp\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")

		// WithDeletionTimestamp
		fmt.Fprintf(buf, "// WithDeletionTimestamp sets the deletion timestamp of the %s\n", typeName)
		fmt.Fprintf(buf, "func WithDeletionTimestamp(timestamp *v1.Time) func(*%s) {\n", typeName)
		fmt.Fprintf(buf, "\treturn func(obj *%s) {\n", typeName)
		fmt.Fprintf(buf, "\t\tobj.DeletionTimestamp = timestamp\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "}\n\n")
	}
}

func renderType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + renderType(t.X)
	case *ast.SelectorExpr:
		return renderType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + renderType(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", renderType(t.Key), renderType(t.Value))
	default:
		return fmt.Sprintf("unsupported-%T", expr)
	}
}
