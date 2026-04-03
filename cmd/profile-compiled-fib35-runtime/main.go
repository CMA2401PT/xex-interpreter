package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	ast "xex/ast"
	ir_operator "xex/ast/operator"
	"xex/engine"
	enginecompile "xex/engine/compile"
	convert_ast "xex/lower-ast/convert-ast"
)

func main() {
	iterations := flag.Int("n", 3, "number of fib(35) runtime executions to profile")
	outDir := flag.String("out", "pprof/runtime-fib35", "directory to write runtime-only profile artifacts")
	enableCPU := flag.Bool("cpu", true, "write CPU profile artifacts")
	enableMem := flag.Bool("mem", false, "write memory profile artifacts")
	flag.Parse()

	if *iterations <= 0 {
		fmt.Fprintf(os.Stderr, "-n must be > 0, got %d\n", *iterations)
		os.Exit(2)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	root := compileRootEnclosure(buildCompiledFib35RootFn())
	objectFn := root.GetCallable()

	cpuProfilePath := filepath.Join(*outDir, "cpu.prof")
	cpuSVGPath := filepath.Join(*outDir, "cpu.svg")
	cpuTopPath := filepath.Join(*outDir, "cpu.top.txt")
	cpuTreePath := filepath.Join(*outDir, "cpu.tree.txt")
	memProfilePath := filepath.Join(*outDir, "mem.prof")
	memAllocSpaceSVGPath := filepath.Join(*outDir, "mem.alloc_space.svg")
	memAllocObjectsSVGPath := filepath.Join(*outDir, "mem.alloc_objects.svg")
	memInuseSpaceSVGPath := filepath.Join(*outDir, "mem.inuse_space.svg")
	memAllocSpaceTopPath := filepath.Join(*outDir, "mem.alloc_space.top.txt")
	memAllocObjectsTopPath := filepath.Join(*outDir, "mem.alloc_objects.top.txt")
	memInuseSpaceTopPath := filepath.Join(*outDir, "mem.inuse_space.top.txt")

	runtime.MemProfileRate = 1

	var cpuFile *os.File
	var err error
	if *enableCPU {
		cpuFile, err = os.Create(cpuProfilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create cpu profile: %v\n", err)
			os.Exit(1)
		}

		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			fmt.Fprintf(os.Stderr, "start cpu profile: %v\n", err)
			os.Exit(1)
		}
	}

	// var last any
	runDurations := make([]time.Duration, 0, *iterations)
	for i := 0; i < *iterations; i++ {
		start := time.Now()
		objectFn(nil)
		runDurations = append(runDurations, time.Since(start))
	}

	if *enableCPU {
		pprof.StopCPUProfile()
		if err := cpuFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close cpu profile: %v\n", err)
			os.Exit(1)
		}
	}

	if *enableMem {
		runtime.GC()
		memFile, err := os.Create(memProfilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create memory profile: %v\n", err)
			os.Exit(1)
		}
		if err := pprof.WriteHeapProfile(memFile); err != nil {
			_ = memFile.Close()
			fmt.Fprintf(os.Stderr, "write memory profile: %v\n", err)
			os.Exit(1)
		}
		if err := memFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close memory profile: %v\n", err)
			os.Exit(1)
		}
	}

	// if last != 9227465 {
	// 	fmt.Fprintf(os.Stderr, "fib(35)=%v, want 9227465\n", last)
	// 	os.Exit(1)
	// }

	if *enableCPU {
		if err := runPprof("-top", "-output", cpuTopPath, os.Args[0], cpuProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write cpu top: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-tree", "-output", cpuTreePath, os.Args[0], cpuProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write cpu tree: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-svg", "-output", cpuSVGPath, os.Args[0], cpuProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write cpu svg: %v\n", err)
			os.Exit(1)
		}
	}
	if *enableMem {
		if err := runPprof("-sample_index=alloc_space", "-top", "-output", memAllocSpaceTopPath, os.Args[0], memProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write mem alloc_space top: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-sample_index=alloc_objects", "-top", "-output", memAllocObjectsTopPath, os.Args[0], memProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write mem alloc_objects top: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-sample_index=inuse_space", "-top", "-output", memInuseSpaceTopPath, os.Args[0], memProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write mem inuse_space top: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-sample_index=alloc_space", "-svg", "-output", memAllocSpaceSVGPath, os.Args[0], memProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write mem alloc_space svg: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-sample_index=alloc_objects", "-svg", "-output", memAllocObjectsSVGPath, os.Args[0], memProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write mem alloc_objects svg: %v\n", err)
			os.Exit(1)
		}
		if err := runPprof("-sample_index=inuse_space", "-svg", "-output", memInuseSpaceSVGPath, os.Args[0], memProfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "write mem inuse_space svg: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("runtime-only fib(35) profile written to %s\n", *outDir)
	for i, elapsed := range runDurations {
		fmt.Printf("run %d: %s\n", i+1, elapsed)
	}
	if *enableCPU {
		fmt.Printf("cpu profile: %s\n", cpuProfilePath)
		fmt.Printf("cpu svg: %s\n", cpuSVGPath)
		fmt.Printf("cpu top: %s\n", cpuTopPath)
		fmt.Printf("cpu tree: %s\n", cpuTreePath)
	}
	if *enableMem {
		fmt.Printf("mem profile: %s\n", memProfilePath)
		fmt.Printf("mem alloc_space svg: %s\n", memAllocSpaceSVGPath)
		fmt.Printf("mem alloc_objects svg: %s\n", memAllocObjectsSVGPath)
		fmt.Printf("mem inuse_space svg: %s\n", memInuseSpaceSVGPath)
		fmt.Printf("mem alloc_space top: %s\n", memAllocSpaceTopPath)
		fmt.Printf("mem alloc_objects top: %s\n", memAllocObjectsTopPath)
		fmt.Printf("mem inuse_space top: %s\n", memInuseSpaceTopPath)
	}
}

func runPprof(args ...string) error {
	cmd := exec.Command("go", append([]string{"tool", "pprof"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func compileRootEnclosure(fn ast.CreateEnclosureExpression) *engine.Enclosure {
	proto := convert_ast.ConvertFn(fn)
	fp, _ := enginecompile.CompileFnProto(proto, nil, nil)
	return fp.ToEnclosure(nil, nil)
}

func buildCompiledFib35RootFn() ast.CreateEnclosureExpression {
	fibName := ast.Identifier{Value: "fibonacci"}
	xName := ast.Identifier{Value: "x"}
	fibFn := ast.CreateEnclosureExpression{
		ParameterSymbols: []ast.Identifier{xName},
		LocalSymbols:     []ast.Identifier{xName},
		FreeSymbols:      []ast.Identifier{fibName},
		Body: ast.IfStatement{
			ConditionAndConsequence: []ast.ConditionAndConsequence{
				{
					Cond: ast.InfixExpression{
						Operator: ir_operator.EQ,
						Left:     newLocalIdentifier("x"),
						Right:    ast.Literial{Value: 0},
					},
					Do: ast.ReturnStatement{ReturnValue: ast.Literial{Value: 0}},
				},
			},
			Alternative: ast.IfStatement{
				ConditionAndConsequence: []ast.ConditionAndConsequence{
					{
						Cond: ast.InfixExpression{
							Operator: ir_operator.EQ,
							Left:     newLocalIdentifier("x"),
							Right:    ast.Literial{Value: 1},
						},
						Do: ast.ReturnStatement{ReturnValue: ast.Literial{Value: 1}},
					},
				},
				Alternative: ast.ReturnStatement{
					ReturnValue: ast.InfixExpression{
						Operator: ir_operator.PLUS,
						Left: ast.CallExpression{
							Function: newFreeIdentifier("fibonacci"),
							Arguments: []ast.ExpressionNode{
								ast.InfixExpression{
									Operator: ir_operator.MINUS,
									Left:     newLocalIdentifier("x"),
									Right:    ast.Literial{Value: 1},
								},
							},
						},
						Right: ast.CallExpression{
							Function: newFreeIdentifier("fibonacci"),
							Arguments: []ast.ExpressionNode{
								ast.InfixExpression{
									Operator: ir_operator.MINUS,
									Left:     newLocalIdentifier("x"),
									Right:    ast.Literial{Value: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	return ast.CreateEnclosureExpression{
		LocalSymbols: []ast.Identifier{fibName},
		Body: ast.StatementsStatement{
			Statements: []ast.StatementNode{
				newLocalAssignStatement("fibonacci", fibFn),
				ast.ReturnStatement{
					ReturnValue: ast.CallExpression{
						Function: newLocalIdentifier("fibonacci"),
						Arguments: []ast.ExpressionNode{
							ast.Literial{Value: 35},
						},
					},
				},
			},
		},
	}
}

func newLocalIdentifier(name string) ast.IdentifierExpression {
	return ast.IdentifierExpression{
		Identifier: ast.Identifier{Value: name},
		Scope:      ast.IdentifierScopeLocal,
	}
}

func newFreeIdentifier(name string) ast.IdentifierExpression {
	return ast.IdentifierExpression{
		Identifier: ast.Identifier{Value: name},
		Scope:      ast.IdentifierScopeFree,
	}
}

func newLocalLeftIdentifier(name string) ast.LeftIdentifier {
	return ast.LeftIdentifier{
		Identifier: ast.Identifier{Value: name},
		Scope:      ast.IdentifierScopeLocal,
	}
}

func newLocalAssignStatement(name string, right ast.ExpressionNode) ast.ExpressionStatement {
	return ast.ExpressionStatement{
		Expression: ast.AssignmentExpression{
			Left:  newLocalLeftIdentifier(name),
			Right: right,
		},
	}
}
