package ast

type AssignmentExpression struct {
	Left  LeftValueNode
	Right RightValue
}

func (as AssignmentExpression) String() string { return as.IdentString(0) }
func (as AssignmentExpression) IdentString(identLevel int) string {
	return as.Left.IdentString(identLevel) + " = " + as.Right.IdentString(identLevel)
}

func (as AssignmentExpression) ExpressionNode() {}
func (as AssignmentExpression) Simplify() ExpressionNode {
	as.Left = simplifyLeftValue(as.Left)
	as.Right = as.Right.Simplify()
	return as
}

// 将值接收到这个标识符
type LeftIdentifier struct {
	Identifier
	Scope IdentifierScope
}

func (li LeftIdentifier) LeftValueNode() {}
func (li LeftIdentifier) String() string { return li.IdentString(0) }
func (li LeftIdentifier) IdentString(identLevel int) string {
	return li.Identifier.String()
}

// 将值接收到 CanSetIndex[Index]
type LeftSetIndex struct {
	CanSetIndex ExpressionNode
	Index       ExpressionNode
}

func (li LeftSetIndex) LeftValueNode() {}

func (ls LeftSetIndex) String() string { return ls.IdentString(0) }
func (ls LeftSetIndex) IdentString(identLevel int) string {
	return ls.CanSetIndex.IdentString(identLevel) + "[" + ls.Index.IdentString(identLevel) + "]"
}

// 将值接收到 CanSetAttribute.Attribute
// Attribute 只能为 string
type LeftSetAttribute struct {
	CanSetAttribute ExpressionNode
	Attribute       string
}

func (li LeftSetAttribute) LeftValueNode() {}
func (ls LeftSetAttribute) String() string { return ls.IdentString(0) }
func (ls LeftSetAttribute) IdentString(identLevel int) string {
	return ls.CanSetAttribute.IdentString(identLevel) + "." + ls.Attribute
}

func simplifyLeftValue(left LeftValueNode) LeftValueNode {
	switch left := left.(type) {
	default:
		return left
	case LeftIdentifier:
		return left
	case LeftSetIndex:
		left.CanSetIndex = left.CanSetIndex.Simplify()
		left.Index = left.Index.Simplify()
		return left
	case LeftSetAttribute:
		left.CanSetAttribute = left.CanSetAttribute.Simplify()
		return left
	}
}
