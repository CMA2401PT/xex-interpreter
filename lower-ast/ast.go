package lower_ast

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	ast "xex/ast"
)

// 每个 SuffixOperate 都消耗 ConsumeOperand() 个操作数并生成一个操作数作为结果
type SuffixOperate interface {
	// 消耗的操作数个数
	ConsumeOperand() int32
	// 打印当前操作,其前面不应该有空格或者Indent
	String() string
	// 返回稳定的结构化 hash
	Hash() uint64
}

// SuffixOperatesSequence 是一串后缀表达式构成的序列
// 最终应该能产生一个值
// SuffixOperatesSequence 不涉及控制流
type SuffixOperatesSequence []SuffixOperate

// CFGReturnTarget 代表 return 语句跳转位置
const CFGReturnTarget int32 = -1

type Sequence interface {
	IdentString(level int) string
	Hash() uint64
}

type CFGBlock[S Sequence] struct {
	// 无条件执行的一系列expression，除最后一个结果外，都将被丢弃
	Sequence S
	// 跳转控制语句
	IsNoCondJump bool
	// 跳转目标 (当为 -1 时，为函数的返回)
	// 当 IsNoCondJump 为 True 时，总是跳转到指定的 CFGBlock
	// 当 IsNoCondJump 为 False 时，如果最后一个结果为 True，跳转到 JumpTo 的 CFGBlock
	JumpTo int32
	// 当 IsNoCondJump 为 False 时，且最后一个结果为 False 时，跳转到 Aux 的 CFGBlock
	Aux int32
}

type CFGraph[S Sequence] []CFGBlock[S]
type LowerAst CFGraph[SuffixOperatesSequence]

type FunctionProto struct {
	// Symbols 指明了在这个函数内有哪些变量，它们各是什么身份
	// 各种 Symbol 内部，各种 Symbol 之间都不允许重名
	// 代表参数，但是实际上和 LocalSymbols 有差不多的表现
	ParameterSymbols []ast.Identifier
	// 代表在这个函数内声明的变量，不允许与其他变量（包括非 local 的变量）重名
	LocalSymbols []ast.Identifier
	// 代表在这个函数外部声明的局部变量，一般作为闭包出现
	FreeSymbols []ast.Identifier
	// 代表这个函数外部声明的变量，但是和全局变量不同之处在于，编译的时候并不知道是否存在，也无法知道其索引
	ImportSymbols []ast.Identifier

	// 转换为 CFG 的函数内部实现
	Graph     LowerAst
	HashValue uint64

	// 指明这个函数是否完全禁用异步模式
	NonAsync bool
}

func (block CFGBlock[S]) Hash() uint64 {
	return newHash(`hashTagCFGBlock`).
		Uint64(block.Sequence.Hash()).
		Literial(block.IsNoCondJump).
		Literial(block.JumpTo).
		Literial(block.Aux).
		Sum64()
}

func (graph CFGraph[S]) Hash() uint64 {
	builder := newHash(`hashTagCFGraph`)
	builder.Uint64(uint64(len(graph)))
	for _, block := range graph {
		builder.Uint64(block.Hash())
	}
	return builder.Sum64()
}

func (graph CFGraph[Sequence]) IdentString(level int) string {
	out := ""
	for i, e := range graph {
		out += fmt.Sprintf("%v%v:\n", IdentToString(level), i)
		out += e.Sequence.IdentString(level + 1)
		// for _, en := range e.Sequence {
		// 	out += fmt.Sprintf("%v%v\n", identToString(level+1), en.IdentString(level+1))
		// }
		if e.IsNoCondJump {
			if e.JumpTo == CFGReturnTarget {
				out += fmt.Sprintf("%vreturn\n", IdentToString(level+1))
				continue
			}
			out += fmt.Sprintf("%vgoto %v\n", IdentToString(level+1), e.JumpTo)
		} else {
			out += fmt.Sprintf("%vgoto %v if True else %v\n", IdentToString(level+1), e.JumpTo, e.Aux)
		}
	}
	return out
}

func (graph CFGraph[Sequence]) String() string {
	return graph.IdentString(0)
}

func (graph LowerAst) Hash() uint64 {
	return CFGraph[SuffixOperatesSequence](graph).Hash()
}

func NewFuncProto(
	ParameterSymbols, LocalSymbols, FreeSymbols, ImportSymbols []ast.Identifier,
	Graph LowerAst, NonAsync bool,
) FunctionProto {
	fp := FunctionProto{
		ParameterSymbols: ParameterSymbols,
		LocalSymbols:     LocalSymbols,
		FreeSymbols:      FreeSymbols,
		ImportSymbols:    ImportSymbols,
		Graph:            Graph,
		NonAsync:         NonAsync,
	}
	builder := newHash(`hashTagFunctionProto`)
	builder.String(`params`)
	for _, v := range fp.ParameterSymbols {
		builder.String(v.Value)
	}
	builder.String(`locals`)
	for _, v := range fp.LocalSymbols {
		builder.String(v.Value)
	}
	builder.String(`frees`)
	for _, v := range fp.FreeSymbols {
		builder.String(v.Value)
	}
	builder.String(`imports`)
	for _, v := range fp.ImportSymbols {
		builder.String(v.Value)
	}
	builder.Uint64(fp.Graph.Hash())
	builder.Literial(fp.NonAsync)
	hv := builder.Sum64()
	fp.HashValue = hv
	return fp
}

func (fp FunctionProto) Hash() uint64 {
	if fp.HashValue == 0 {
		panic("should not happen")
	}
	return fp.HashValue
}

func (fp FunctionProto) GetAllImports() (imports []string) {
	occurs := map[string]struct{}{}
	for _, importName := range fp.ImportSymbols {
		occurs[importName.Value] = struct{}{}
	}
	for _, b := range fp.Graph {
		for _, op := range b.Sequence {
			if opt, ok := op.(CreateEnclosureExpression); ok {
				for _, importName := range opt.FunctionProto.GetAllImports() {
					occurs[importName] = struct{}{}
				}
			}
		}
	}
	return slices.Collect(maps.Keys(occurs))
}

func IdentToString(level int) string {
	out := strings.Builder{}
	for range level {
		out.WriteString("\t")
	}
	return out.String()
}

func (s SuffixOperatesSequence) IdentString(level int) string {
	out := ""
	out += IdentToString(level)
	// 是不是这行的第一个，不是的话需要加逗号
	first := true
	for _, op := range s {
		if op == nil {
			out += "\n" + IdentToString(level)
			first = true
			continue
		}
		if _, ok := op.(DropOperate); ok {
			out += "\n" + IdentToString(level)
			first = true
			continue
		}
		if first {
			first = false
		} else {
			out += ","
		}
		out += op.String()
	}
	return out
}

func (s SuffixOperatesSequence) Hash() uint64 {
	builder := newHash(`SuffixOperatesSequence`).Uint64(uint64(len(s)))
	for _, op := range s {
		if op == nil {
			builder.Nil()
			continue
		}
		builder.Uint64(op.Hash())
	}
	return builder.Sum64()
}
