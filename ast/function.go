package ast

import (
	"fmt"
	"strings"
)

type CreateEnclosureExpression struct {
	Body StatementNode
	// Symbols 指明了在这个函数内有哪些变量，它们各是什么身份
	// 各种 Symbol 内部，各种 Symbol 之间都不允许重名
	// 代表参数，但是实际上和 LocalSymbols 有差不多的表现
	ParameterSymbols []Identifier
	// 代表在这个函数内声明的变量，不允许与其他变量（包括非 local 的变量）重名
	LocalSymbols []Identifier
	// 代表在这个函数外部声明的局部变量，一般作为闭包出现
	FreeSymbols []Identifier
	// 代表这个函数外部声明的变量，但是和全局变量不同之处在于，编译的时候并不知道是否存在，也无法知道其索引
	ImportSymbols []Identifier
	// 指明这个函数是否完全禁用异步模式
	NonAsync bool
}

func (fe CreateEnclosureExpression) NumArgs() int {
	return len(fe.ParameterSymbols)
}

func (fe CreateEnclosureExpression) ExpressionNode() {}
func (fe CreateEnclosureExpression) Simplify() ExpressionNode {
	return fe
}
func (fe CreateEnclosureExpression) String() string {
	return fe.IdentString(0)
}
func (fe CreateEnclosureExpression) IdentString(identLevel int) string {
	genSymbolList := func(symbols []Identifier) string {
		symbolStrings := []string{}
		for _, s := range symbols {
			symbolStrings = append(symbolStrings, s.String())
		}
		return strings.Join(symbolStrings, ",")
	}
	start := "function"
	if fe.NonAsync {
		start = "sync_function"
	}
	identLevel += 1
	return fmt.Sprintf(
		start+"(%v){\n%simports %v;\n%sfrees %v;\n%slocals %v;\n%v%s}",
		genSymbolList(fe.ParameterSymbols),
		IdentToString(identLevel),
		genSymbolList(fe.ImportSymbols),
		IdentToString(identLevel),
		genSymbolList(fe.FreeSymbols),
		IdentToString(identLevel),
		genSymbolList(fe.LocalSymbols),
		fe.Body.IdentString(identLevel),
		IdentToString(identLevel-1),
	)
}

type CallExpression struct {
	Function  ExpressionNode
	Arguments []ExpressionNode
}

func (ce CallExpression) ExpressionNode() {}
func (ce CallExpression) Simplify() ExpressionNode {
	ce.Function = ce.Function.Simplify()
	ce.Arguments = simplifyExpressionSlice(ce.Arguments)
	return ce
}
func (ce CallExpression) String() string { return ce.IdentString(0) }
func (ce CallExpression) IdentString(identLevel int) string {
	ps := []string{}
	for _, p := range ce.Arguments {
		ps = append(ps, p.IdentString(identLevel))
	}
	return ce.Function.IdentString(identLevel) + "(" +
		strings.Join(ps, ",") +
		")"
}
