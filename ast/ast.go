package ast

import "strings"

type ExpressionNode interface {
	// 代表这个节点一定会产生一个值
	ExpressionNode()
	Simplify() ExpressionNode
	String() string
	IdentString(identLevel int) string
}

type StatementNode interface {
	// 代表这个节点不会产生值，但是可能会影响控制流
	// 例如 break，return
	StatementNode()
	String() string
	IdentString(identLevel int) string
}

type LeftValueNode interface {
	// 代表这个节点只会出现在赋值节点的左侧，是接收值
	// 例如 a = 1, list[idx]=2, map["key"] ="value"
	LeftValueNode()
	String() string
	IdentString(identLevel int) string
}

type RightValue interface {
	// 代表这个节点会出现在赋值节点的右侧，可以产生一个值
	ExpressionNode
}

type Identifier struct {
	Value string
}

func (id Identifier) String() string { return id.Value }
func (id Identifier) IdentString(identLevel int) string {
	return id.Value
}

type IdentifierScope byte

const (
	IdentifierScopeInvalid = IdentifierScope(iota)
	IdentifierScopeLocal
	IdentifierScopeFree
	IdentifierScopeImport
	IdentifierScopeGlobal
)

func (is IdentifierScope) String() string {
	switch is {
	default:
		return "invalid"
	case IdentifierScopeLocal:
		return "local"
	case IdentifierScopeFree:
		return "free"
	case IdentifierScopeImport:
		return "import"
	case IdentifierScopeGlobal:
		return "global"
	}
}

// 代表这个是求值节点
type IdentifierExpression struct {
	Identifier
	Scope IdentifierScope
}

func (ie IdentifierExpression) ExpressionNode() {}
func (ie IdentifierExpression) Simplify() ExpressionNode {
	return ie
}
func (ie IdentifierExpression) String() string {
	return ie.IdentString(0)
}
func (ie IdentifierExpression) IdentString(identLevel int) string {
	return ie.Identifier.IdentString(identLevel)
}

func IdentToString(level int) string {
	out := strings.Builder{}
	for range level {
		out.WriteString("\t")
	}
	return out.String()
}
