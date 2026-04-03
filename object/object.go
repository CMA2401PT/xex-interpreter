package object

import (
	"fmt"
	"reflect"
	ir_operator "xex/ast/operator"
	"xex/async"
)

// 由于自由变量传递都是使用引用传递，故使用 *RefOrValue 来作为引用
// 其中 Value 为真正的值
// 通过这种方式，实现将 gc 寄生在宿主语言上，
// 同时使得自由变量和局部变量可以指向同一个目标
type RefOrValue struct {
	Value Box
	Ref   *Ref
}

type Ref struct {
	Value Box
}

const (
	BasicTypNil = byte(iota)
	BasicTypBool
	BasicTypInt
	BasicTypStr
	BasicTypObjList
	BasicTypObjMap
	BasicTypEnclosure
	BasicTypCustom
	BasicTypeType
)
const BasicLitEnd = BasicTypStr

type Box struct {
	BasicType byte
	numData   uint64
	aux       any
}

type TypeInfo struct {
	BasicType      byte
	customSelfDesc interface {
		String() string
	}
	customRT reflect.Type
}

var Nil = Box{}
var Empty = Nil
var True = Box{
	BasicType: BasicTypBool,
	numData:   1,
}
var False = Box{
	BasicType: BasicTypBool,
}

func Type(obj Box) TypeInfo {
	switch obj.BasicType {
	default:
		panic(fmt.Errorf("should not happen: actual obj %T:%v", obj.aux, obj.aux))
	case BasicTypNil, BasicTypBool, BasicTypInt, BasicTypStr, BasicTypObjList, BasicTypObjMap, BasicTypEnclosure, BasicTypeType:
		return TypeInfo{BasicType: obj.BasicType}
	case BasicTypCustom:
		if selfDesc, ok := obj.aux.(interface {
			Type() interface {
				String() string
			}
		}); ok {
			return TypeInfo{BasicType: BasicTypCustom, customSelfDesc: selfDesc.Type()}
		}
		return TypeInfo{BasicType: BasicTypCustom, customRT: reflect.TypeOf(obj.aux)}
	}
}

func UnBoxCustom(obj Box) any {
	if obj.BasicType != BasicTypCustom {
		panic(fmt.Errorf("%v is not custom type", obj))
	}
	return obj.aux
}

func BoxCustom(custom any) Box {
	return Box{
		BasicType: BasicTypCustom,
		aux:       custom,
	}
}

func UnBoxCustomType[T any](obj Box) T {
	if obj.BasicType != BasicTypCustom {
		panic(fmt.Errorf("%v is not custom type", obj))
	}
	return obj.aux.(T)
}

func BoxCustomType[T any](custom T) Box {
	return Box{
		BasicType: BasicTypCustom,
		aux:       custom,
	}
}

func UnBoxBool(obj Box) bool {
	if obj.BasicType != BasicTypBool {
		panic(fmt.Errorf("%v is not bool type", obj))
	}
	return obj.numData == 1
}

func BoxBool(b bool) Box {
	if b {
		return True
	}
	return False
}

func UnBoxInt(obj Box) int {
	if obj.BasicType != BasicTypInt {
		panic(fmt.Errorf("%v is not int type", obj))
	}
	return int(obj.numData)
}

func BoxInt(b int) Box {
	return Box{
		BasicType: BasicTypInt,
		numData:   (uint64(b)),
	}
}

func UnBoxString(obj Box) string {
	if obj.BasicType != BasicTypStr {
		panic(fmt.Errorf("%v is not string type", obj))
	}
	return obj.aux.(string)
}

func BoxString(b string) Box {
	return Box{
		BasicType: BasicTypStr,
		aux:       b,
	}
}

func UnBoxObjList(obj Box) List {
	if obj.BasicType != BasicTypObjList {
		panic(fmt.Errorf("%v is not list type", obj))
	}
	return obj.aux.(List)
}

func BoxObjList(b List) Box {
	return Box{
		BasicType: BasicTypObjList,
		aux:       b,
	}
}

func UnBoxObjMap(obj Box) Map {
	if obj.BasicType != BasicTypObjMap {
		panic(fmt.Errorf("%v is not map type", obj))
	}
	return obj.aux.(Map)
}

func BoxObjMap(b Map) Box {
	return Box{
		BasicType: BasicTypObjMap,
		aux:       b,
	}
}

func UnBoxObjEnclosure[T any](obj Box) T {
	if obj.BasicType != BasicTypEnclosure {
		panic(fmt.Errorf("%v is not enclosure type", obj))
	}
	return obj.aux.(T)
}

func BoxObjEnclosure[T any](b T) Box {
	// 强制确保符合 enclosure 签名
	var i any
	i = b
	_ = i.(Enclosure)
	return Box{
		BasicType: BasicTypEnclosure,
		aux:       b,
	}
}

type List struct {
}

func NewList() List {
	panic("not implement")
}

func (l List) AppendItem(elem Box) {
	panic("not implement")
}

func BoxList(l List) Box {
	panic("not implement")
}

type Map struct {
}

func NewMap() Map {
	panic("not implement")
}

func (m Map) SetKeyValue(key Box, value Box) {
	panic("not implement")
}

func BoxMap(l Map) Box {
	panic("not implement")
}

type AsyncHandleType = async.Handle[[]Box, Box]
type AsyncYieldReason = async.YieldReason[[]Box, Box]
type NormalHostFn = func(args []Box) Box
type AsyncHostFn = func(handle AsyncHandleType, args []Box) AsyncYieldReason

func AsyncYieldFinish(ret Box) async.YieldByFinish[[]Box, Box] {
	return async.YieldByFinish[[]Box, Box]{
		Result: ret,
	}
}

var AsyncShim = func(ret Box) []Box { return []Box{ret} }

func GetCallable(box Box) func(args []Box) Box {
	switch v := UnBoxAny(box).(type) {
	default:
		panic("not implement")
	case func(args []Box) Box:
		return v
	}
}

func TryGetAsyncCallable(box Box) (AsyncHostFn, bool) {
	switch v := UnBoxAny(box).(type) {
	default:
		return nil, false
	case AsyncHostFn:
		return v, true
	}
}

func SetArrtibute(obj Box, attr string, elem Box) {
	panic("not implement")
}

func SetItem(obj Box, index Box, elem Box) {
	panic("not implement")
}

func DispatchPrefixOp(op ir_operator.Operator) func(Box) Box {
	switch op {
	default:
		panic("not implement")
	case ir_operator.BANG:
		return BangOp
	case ir_operator.MINUS:
		return NegateOp
	}
}

func DispatchInfixOp(op ir_operator.Operator) func(Box, Box) Box {
	switch op {
	default:
		panic("not implement")
	case ir_operator.PLUS:
		return PlusOp
	case ir_operator.MINUS:
		return MinusOp
	case ir_operator.EQ:
		return EqualOp
	case ir_operator.NOT_EQ:
		return NotEqualOp
	}
}

func BangOp(operand Box) Box {
	switch operand.BasicType {
	default:
		panic("not implement")
	case BasicTypNil:
		return True
	case BasicTypBool:
		return BoxBool(!UnBoxBool(operand))
	}
}

func NegateOp(operand Box) Box {
	switch operand.BasicType {
	default:
		panic("not implement")
	case BasicTypInt:
		return BoxInt(-UnBoxInt(operand))
	}
}

func PlusOp(left Box, right Box) Box {
	switch left.BasicType {
	default:
		panic(fmt.Errorf("not implement: %T:%v", left, left))
	case BasicTypInt:
		return BoxInt(UnBoxInt(left) + UnBoxInt(right))
	}
}

func MinusOp(left Box, right Box) Box {
	switch left.BasicType {
	default:
		panic("not implement")
	case BasicTypInt:
		return BoxInt(UnBoxInt(left) - UnBoxInt(right))
	}
}

func EqualOp(left Box, right Box) Box {
	if left.BasicType == BasicTypInt && right.BasicType == BasicTypInt {
		if left.numData == right.numData {
			return True
		} else {
			return False
		}
	}
	if left.BasicType != right.BasicType {
		panic("not comparable")
	}
	if left.BasicType > BasicLitEnd {
		panic("not implement")
	}
	if left == right {
		return True
	} else {
		return False
	}
}

func NotEqualOp(left Box, right Box) Box {
	if left.BasicType == BasicTypInt && right.BasicType == BasicTypInt {
		if left.numData == right.numData {
			return False
		} else {
			return True
		}
	}
	if left.BasicType != right.BasicType {
		panic("not comparable")
	}
	if left.BasicType > BasicLitEnd {
		panic("not implement")
	}
	if left != right {
		return True
	} else {
		return False
	}
}
