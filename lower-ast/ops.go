package lower_ast

import (
	"fmt"
	ast "xex/ast"
	ir_operator "xex/ast/operator"
)

// DropOperate 代表丢弃前面一个操作数。
// 它不应该出现在单个 ExpressionNode 的 lowering 结果里，
// 只用于“顺序执行多个 expression，但只保留最后一个值”的上层场景。
type DropOperate struct{}

func (DropOperate) ConsumeOperand() int32 { return 1 }
func (DropOperate) String() string        { return "" }
func (DropOperate) Hash() uint64 {
	return newHash(`hashTagDropOperate`).Sum64()
}

type IdentifierExpression struct {
	IdentifierName string
	Scope          ast.IdentifierScope
}

func (ie IdentifierExpression) ConsumeOperand() int32 { return 0 }
func (ie IdentifierExpression) String() string {
	return fmt.Sprintf("(identifier,%v,%v)", ie.Scope, ie.IdentifierName)
}
func (ie IdentifierExpression) Hash() uint64 {
	return newHash(`hashTagIdentifierExpression`).String(ie.IdentifierName).Byte(byte(ie.Scope)).Sum64()
}

type LiterialExpression struct {
	Value any
}

func (le LiterialExpression) ConsumeOperand() int32 { return 0 }
func (le LiterialExpression) String() string {
	return fmt.Sprintf("(literal,%v)", le.Value)
}
func (le LiterialExpression) Hash() uint64 {
	return newHash(`hashTagLiteralExpression`).Literial(le.Value).Sum64()
}

type PrefixExpression struct {
	Operator ir_operator.Operator
}

func (pe PrefixExpression) ConsumeOperand() int32 { return 1 }
func (pe PrefixExpression) String() string {
	return fmt.Sprintf("(prefixExp,%v)", pe.Operator.String())
}
func (pe PrefixExpression) Hash() uint64 {
	return newHash(`hashTagPrefixExpression`).Byte(byte(pe.Operator)).Sum64()
}

type InfixExpression struct {
	Operator ir_operator.Operator
}

func (ie InfixExpression) ConsumeOperand() int32 { return 2 }
func (ie InfixExpression) String() string {
	return fmt.Sprintf("(infixExp,%v)", ie.Operator.String())
}
func (ie InfixExpression) Hash() uint64 {
	return newHash(`hashTagInfixExpression`).Byte(byte(ie.Operator)).Sum64()
}

type ListExpression struct {
	ElementsCount int32
}

func (le ListExpression) ConsumeOperand() int32 { return le.ElementsCount }
func (le ListExpression) String() string {
	return fmt.Sprintf("(list,%v)", le.ElementsCount)
}
func (le ListExpression) Hash() uint64 {
	return newHash(`hashTagListExpression`).Literial(le.ElementsCount).Sum64()
}

type MapExpression struct {
	PairsCount int32
}

func (me MapExpression) ConsumeOperand() int32 { return me.PairsCount * 2 }
func (me MapExpression) String() string {
	return fmt.Sprintf("(map,%v)", me.PairsCount)
}
func (me MapExpression) Hash() uint64 {
	return newHash(`hashTagMapExpression`).Literial(me.PairsCount).Sum64()
}

type AssignmentIdentifierExpression struct {
	ast.Identifier
	Scope ast.IdentifierScope
}

func (aie AssignmentIdentifierExpression) ConsumeOperand() int32 { return 1 }
func (aie AssignmentIdentifierExpression) String() string {
	return fmt.Sprintf("(setIdentifier,%v,%v)", aie.Scope, aie.Identifier.Value)
}
func (aie AssignmentIdentifierExpression) Hash() uint64 {
	return newHash(`hashTagAssignmentIdentifierExpression`).String(aie.Identifier.Value).Byte(byte(aie.Scope)).Sum64()
}

type AssignmentAttributeExpression struct {
	Attribute string
}

// 第一个操作数是被设置 attr 的对象，第二个是设置的值
func (aae AssignmentAttributeExpression) ConsumeOperand() int32 { return 2 }
func (aae AssignmentAttributeExpression) String() string {
	return fmt.Sprintf("(setAttr,%v)", aae.Attribute)
}
func (aae AssignmentAttributeExpression) Hash() uint64 {
	return newHash(`hashTagAssignmentAttributeExpression`).String(aae.Attribute).Sum64()
}

type AssignmentIndexExpression struct{}

// 第一个操作数是被设置 index 的对象，第二个是index，第三个是值
func (aie AssignmentIndexExpression) ConsumeOperand() int32 { return 3 }
func (aie AssignmentIndexExpression) String() string {
	return "(setIndex)"
}
func (AssignmentIndexExpression) Hash() uint64 {
	return newHash(`hashTagAssignmentIndexExpression`).Sum64()
}

// 从原型函数构造闭包函数
type CreateEnclosureExpression struct {
	// 原型函数，其缺乏 Free 未绑定，
	// CreateEnclosureExpression 将 Free 与环境相绑定
	FunctionProto
}

func (cee CreateEnclosureExpression) ConsumeOperand() int32 { return 0 }
func (cee CreateEnclosureExpression) String() string {
	return fmt.Sprintf("(createProto,%v)", cee.FunctionProto.Hash())
}
func (cee CreateEnclosureExpression) Hash() uint64 {
	return newHash(`hashTagCreateEnclosureExpression`).Uint64(cee.FunctionProto.Hash()).Sum64()
}

type CallExpression struct {
	ArgumentsCount int32
}

// 第一个操作数是 闭包，后续操作数是参数
func (ce CallExpression) ConsumeOperand() int32 { return ce.ArgumentsCount + 1 }
func (ce CallExpression) String() string {
	return fmt.Sprintf("(call,%v)", ce.ArgumentsCount)
}
func (ce CallExpression) Hash() uint64 {
	return newHash(`hashTagCallExpression`).Literial(ce.ArgumentsCount).Sum64()
}

type YieldExpression struct{}

func (YieldExpression) ConsumeOperand() int32 { return 1 }
func (YieldExpression) String() string        { return "(yield)" }
func (YieldExpression) Hash() uint64 {
	return newHash(`hashTagYieldExpression`).Sum64()
}
