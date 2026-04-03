package ast

import (
	"fmt"
	"strings"
)

// Statements 仅仅能代替一系列 Statement 序列，
// 其他什么都代表不了，不代表这是个作用域
// 也不代表这里的 symbol table 就要变或者里面变量声明需要重命名
// 那些功能应该在这个阶段之前实现，这个包里不允许在局部重新声明变量
type StatementsStatement struct {
	Statements []StatementNode
}

func (st StatementsStatement) StatementNode() {}
func (st StatementsStatement) String() string { return st.IdentString(0) }
func (st StatementsStatement) IdentString(identLevel int) string {
	out := ""
	for _, s := range st.Statements {
		out += s.IdentString(identLevel)
	}
	return out
}

// 什么都不做，只是有时侯作为占位符出现，和 为 nil 的表现完全一致
type EmptyStatement struct{}

func (EmptyStatement) StatementNode() {}
func (EmptyStatement) String() string { return "" }
func (EmptyStatement) IdentString(identLevel int) string {
	return ""
}

type ReturnStatement struct {
	ReturnValue ExpressionNode
}

func (rs ReturnStatement) StatementNode() {}
func (rs ReturnStatement) String() string { return rs.IdentString(0) }
func (rs ReturnStatement) IdentString(identLevel int) string {
	return IdentToString(identLevel) + "return " + rs.ReturnValue.IdentString(identLevel) + ";\n"
}

type BreakStatement struct{}

func (rs BreakStatement) StatementNode() {}
func (rs BreakStatement) String() string { return rs.IdentString(0) }
func (rs BreakStatement) IdentString(identLevel int) string {
	return IdentToString(identLevel) + "break;\n"
}

type ContinueStatement struct{}

func (rs ContinueStatement) StatementNode() {}
func (rs ContinueStatement) String() string { return rs.IdentString(0) }
func (rs ContinueStatement) IdentString(identLevel int) string {
	return IdentToString(identLevel) + "continue;\n"
}

type ExpressionStatement struct {
	Expression ExpressionNode
}

func (rs ExpressionStatement) StatementNode() {}
func (rs ExpressionStatement) String() string { return rs.IdentString(0) }
func (rs ExpressionStatement) IdentString(identLevel int) string {
	return IdentToString(identLevel) + rs.Expression.IdentString(identLevel) + ";\n"
}

type ConditionAndConsequence struct {
	Cond ExpressionNode
	Do   StatementNode
}

type IfStatement struct {
	ConditionAndConsequence []ConditionAndConsequence
	Alternative             StatementNode
}

func (is IfStatement) StatementNode() {}
func (is IfStatement) String() string { return is.IdentString(0) }
func (is IfStatement) IdentString(identLevel int) string {
	branchs := []string{}
	for i, cc := range is.ConditionAndConsequence {
		prefix := IdentToString(identLevel)
		keyword := "if"
		if i > 0 {
			prefix = ""
			keyword = "else if"
		}
		branchs = append(branchs, fmt.Sprintf("%s%s %v {\n%v%s}", prefix, keyword, cc.Cond.IdentString(identLevel), cc.Do.IdentString(identLevel+1), IdentToString(identLevel)))
	}
	branchsString := strings.Join(branchs, " ")
	if isEmptyStatement(is.Alternative) {
		return branchsString + ";\n"
	}
	return branchsString + " else {\n" +
		is.Alternative.IdentString(identLevel+1) +
		IdentToString(identLevel) + "};\n"
}

// 类似 for ; ; {} 但是不是 for initStatement;condition;afterEachLoop { Loop }
// 而是 for condition;afterEachLoop { Loop }
type LoopStatement struct {
	Condition     ExpressionNode // TODO: 优化 Condition 为 Literial{Value: true} 的情况
	LoopBody      StatementNode
	AfterEachLoop StatementNode
}

func (is LoopStatement) StatementNode() {}
func (is LoopStatement) String() string { return is.IdentString(0) }
func (is LoopStatement) IdentString(identLevel int) string {
	afterEach := ""
	if !isEmptyStatement(is.AfterEachLoop) {
		afterEach = strings.TrimSuffix(strings.TrimSuffix(is.AfterEachLoop.String(), "\n"), ";")
	}
	return IdentToString(identLevel) + "for " + is.Condition.IdentString(identLevel) + ";" + afterEach +
		"{\n" +
		is.LoopBody.IdentString(identLevel+1) +
		IdentToString(identLevel) + "};\n"
}

func isEmptyStatement(statement StatementNode) bool {
	if statement == nil {
		return true
	}
	_, ok := statement.(EmptyStatement)
	return ok
}
