package xex

import (
	"maps"
	"slices"
	"xex/ast"
	"xex/engine"
	"xex/engine/compile"
	lower_ast "xex/lower-ast"
	convert_ast "xex/lower-ast/convert-ast"
	"xex/object"
)

func CompileRootFunction(
	fn ast.CreateEnclosureExpression,
	globals map[string]object.Box,
) *engine.Enclosure {
	proto := convert_ast.ConvertFn(fn)
	globalNames := slices.Collect(maps.Keys(globals))
	fp, globalLookup := compile.CompileFnProto(proto, globalNames, nil)
	globalValues := make([]object.Box, len(globalLookup))
	for name, value := range globals {
		slot := globalLookup[lower_ast.IdentifierExpression{
			IdentifierName: name,
			Scope:          ast.IdentifierScopeGlobal,
		}]
		globalValues[slot] = value
	}
	return fp.ToEnclosure(globalValues, nil)
}
