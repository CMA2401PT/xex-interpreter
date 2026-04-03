package object

import "fmt"

// 代表这些函数都不应该作为函数出现在对象语言中
// 仅仅能被宿主语言使用

func BoxLit(lit any) Box {
	switch v := lit.(type) {
	default:
		panic(fmt.Errorf("not implement: %T:%v", lit, lit))
	case nil:
		return Nil
	case bool:
		return BoxBool(v)
	case int:
		return BoxInt(v)
	case string:
		return BoxString(v)
	}
}

func UnBoxLit(box Box) any {
	switch box.BasicType {
	default:
		panic(fmt.Errorf("%v is not literial", box))
	case BasicTypNil:
		return nil
	case BasicTypBool:
		return UnBoxBool(box)
	case BasicTypInt:
		return UnBoxInt(box)
	case BasicTypStr:
		return UnBoxString(box)
	}
}

// object.Box 辨认 Enclosure 的标识
type Enclosure interface {
	EnclosureObj()
}

func BoxAny(a any) Box {
	switch v := a.(type) {
	default:
		return BoxCustom(v)
	case nil:
		return Nil
	case bool:
		return BoxBool(v)
	case int:
		return BoxInt(v)
	case string:
		return BoxString(v)
	case List:
		return BoxList(v)
	case Map:
		return BoxMap(v)
	case Enclosure:
		return BoxObjEnclosure(v)
	}
}

func UnBoxAny(box Box) any {
	switch box.BasicType {
	default:
		return UnBoxCustom(box)
	case BasicTypNil:
		return nil
	case BasicTypBool:
		return UnBoxBool(box)
	case BasicTypInt:
		return UnBoxInt(box)
	case BasicTypStr:
		return UnBoxString(box)
	case BasicTypObjList:
		return UnBoxObjList(box)
	case BasicTypObjMap:
		return UnBoxObjMap(box)
	case BasicTypEnclosure:
		return box.aux
	}
}
