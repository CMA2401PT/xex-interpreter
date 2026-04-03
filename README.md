# xex-interpreter

`xex-interpreter` 是一个围绕 `xex` 解释器引擎搭出来的实验性(玩具)项目。它不从源码字符串开始工作，而是接受已经构造好的、并且已经带有作用域/符号信息的 AST，然后把它逐步降低为：

1. 结构化 AST
2. CFG
3. 后缀表达式序列
4. 可执行的 `xex` 混合操作序列

最终执行器是一个基于 Go 的无 JIT 解释器，但它不是典型的“纯 eval”，也不是典型的“纯字节码 VM”。仓库的核心思路，是把两者各取一部分：普通表达式尽量编译成可直接求值的图，只有在 `call` / `yield` 这些真正会打断控制流的地方，才退化为显式辅助栈和操作队列。

虽然仓库里已经没有早期 Monkey 语言实现的代码了，但这个项目的起点，确实是 Thorsten Ball 的《使用Go语言实现解释器》和《使用Go语言实现编译器》两本书里的 Monkey 语言。这两本书很好，尤其适合作为解释器、编译器和运行时实现的入门起点。`xex-interpreter` 后面走到现在这条路，已经和书里的实现差得很远，但最初的出发点就是从那里开始的。

## 概括
概括一下，这个项目做的事是：

“把已经带好语义信息的 AST 编译成 CFG 和后缀表达式，再进一步编译成一种图-栈混合的 `xex` 执行模型，用 Go 实现一个尽量减少解释器调度开销、同时原生支持可挂起协程的无 JIT 引擎。”

## 项目目标

这个项目关心的不是“再写一个普通(玩具) Go 解释器”，而是两个更具体的问题：

1. 无 JIT 前提下，解释器还能不能继续从执行模型上拿到明显收益。
2. 不依赖 goroutine，能不能做出一套可挂起、可恢复、接近无栈协程效果的执行系统，而且调用代码仍然保持普通函数风格，而不是强迫整条链路都写成 `async/await`。

围绕这个目标，`xex` 想做的事情主要有这些：

- 普通表达式不强制拆成一条条细碎指令，而是尽量合并成 Go 闭包图直接求值，减少解释器调度开销。
- 只有 `call` / `yield` 才退化成显式辅助槽位，因此比纯栈机少了很多机械性的入栈/出栈。
- 闭包捕获通过 `RefOrValue` 实现，局部变量和自由变量共享引用语义，直接依赖宿主语言 GC。
- 异步路径和同步路径共用大部分编译结果，但异步执行器额外维护可恢复的 frame 栈，因此可以在对象语言里直接写“像同步代码一样”的挂起逻辑。

## 性能

这个仓库目前没有成体系的 benchmark 套件，但已经有一个可以直接拿来对比的点：`fib(35)`。

在同一台 M4 机器上：

- [uGo](https://github.com/ozanh/ugo/tree/main) 在 [ugobenchfib](https://github.com/ozanh/ugobenchfib) 中给出的 `fib(35)` 耗时是 `1.23s`
- `xex` 的同步模式 `fib(35)` 耗时约为 `980ms`，也就是 `0.98s`

只看这一个点，`xex` 同步模式大约快了 `20%`。这个结果只能说明当前这条执行路径在这个测试上有优势，但不能直接外推成“所有场景都更快”，也不能替代系统化 benchmark。

## 核心执行路线

### 1. AST 不是解析器产物的末端，而是引擎输入

项目的公开入口在 [xex.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/xex.go)。`CompileRootFunction` 接受一个 `ast.CreateEnclosureExpression` 和全局变量表，说明这个仓库当前关注的是“如何执行一棵已经准备好的 AST”，不是“如何把源码解析成 AST”。

AST 本身要求已经带有这些信息：

- 变量作用域：`local` / `free` / `import` / `global`
- 函数的参数、局部变量、自由变量、导入符号列表
- 是否强制同步：`NonAsync`

这部分定义在 [ast](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast)。

### 2. 先把语句树压成 CFG，把表达式压成后缀序列

[lower-ast/convert-statement](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-statement) 负责把 `if` / `loop` / `break` / `continue` / `return` 这些语句结构改写成 CFG。

[lower-ast/convert-expression](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-expression) 负责把表达式改写成 `SuffixOperatesSequence`。这个序列满足“每个操作消费 N 个操作数，生成 1 个结果”的约束，便于后续编译。

对应的数据结构在：

- [lower-ast/ast.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/ast.go)
- [lower-ast/ops.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/ops.go)

### 3. `xex` 编译阶段：普通表达式并成图，特殊点退化成栈

[engine/compile](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile) 是项目的核心。

这里不是把所有节点一视同仁编译成统一字节码，而是分成两类情况：

- 普通表达式：编译成 `NormalSuffixOperate`，本质是 `func(env Env) object.Box`
- 特殊表达式：`call` / `yield` 会被拆出来，落入 `HybirdOperation`

`CompileSeq` 和 `lowerSpecialOperands` 的策略很关键：

- 能留在图里的表达式，尽量留在图里。
- 一旦遇到 `call` 或 `yield`，才把必须保序的操作数 fallback 到辅助槽位。
- 后续计算通过桥接函数从辅助槽位读取值，继续拼图。

所以这个执行模型不是“纯图”，也不是“纯栈”，而是图和显式辅助栈的混合体。

### 4. 运行时分同步和异步两条执行器

[engine/model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/model.go) 实现同步执行器。

它的特点是：

- 函数调用时复用局部槽和辅助槽，尽量少分配。
- 如果整条链路都是同步函数，`call` 可以直接内联到图求值逻辑里。

[engine/async_model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/async_model.go) 实现异步执行器。

它额外维护：

- `pc`
- frame 栈
- 每层函数自己的 `LocalAndFreeVars`
- 每层函数自己的 `AuxSlots`

这样 `yield` 时可以把当前状态完整挂起，恢复后继续从原位置执行。

## 协程模型

[async](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async) 是独立的一套小型调度框架。它不是直接绑定 goroutine，而是把协程拆成 `Step -> YieldReason -> Resume` 这个模型。

关键点：

- `YieldBySleep` 表示挂起到某个时间点。
- `YieldByAwait` 表示等待另一个异步任务完成。
- `YieldByOuterResume` 表示由外部事件恢复。
- `YieldByFinish` 表示任务结束。

事件循环在 [async/loop_model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async/loop_model.go) 和 [async/runner.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async/runner.go)。

这套设计不是为了“模拟 goroutine”，而是为了给 `xex` 引擎一个宿主可控、可嵌入、可外部恢复的协程执行模型。对象语言里可以直接出现 `yield` 表达式，但宿主仍然能决定何时恢复、如何调度、如何等待子任务。

## 不足与边界

这个项目现在更准确的定位是“执行模型原型”，还不是完整语言产品。

目前已经很明确的不足主要有这些：

- 没有 parser，也没有完整前端；输入是手工构造或外部提供的 AST。
- `object` 层还有不少未完成部分，尤其是列表、映射、属性/索引写入以及部分操作符实现仍然是 `panic("not implement")`。
- 同步和异步核心链路已经能跑，但外围对象系统还不完整，离“可用语言运行时”还有距离。
- 仓库里有性能实验入口，但 benchmark 还不系统，当前能稳定拿出来说的主要还是 `fib(35)` 这一类单点测试。
- 仓库当前最有价值的部分，是 `xex` 的编译/执行模型本身，而不是外围语言工具链。

所以这里要先说清楚：这个仓库现在不是一个可以直接拿来跑脚本语言源码的成熟解释器，而是一个把执行模型做深、把协程语义和无 JIT 执行方式做细的实验项目。

## 如何阅读这个仓库

如果你的目标是理解项目主线，建议按这个顺序看：

1. [xex.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/xex.go)
2. [ast](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast)
3. [lower-ast/convert-ast](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-ast)
4. [lower-ast/convert-statement](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-statement)
5. [lower-ast/convert-expression](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-expression)
6. [engine/compile](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile)
7. [engine/model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/model.go)
8. [engine/async_model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/async_model.go)
9. [async](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async)

## 目录结构

### 根目录

- [xex.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/xex.go)
  对外入口。把根函数 AST 编译成可执行的 `engine.Enclosure`。
- [go.mod](/Users/dai/Projects/用go语言自制编译器/neo-monkey/go.mod)
  模块定义。当前模块名就是 `xex`。

### `ast/`

对象语言前端 AST 定义，但这里只包含“结构”和少量简化逻辑，不包含 parser。

- [ast/ast.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/ast.go)
  基础节点接口、标识符、作用域定义。
- [ast/expression.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/expression.go)
  表达式节点。
- [ast/statements.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/statements.go)
  语句节点和控制流结构。
- [ast/function.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/function.go)
  函数/闭包定义与调用表达式。
- [ast/assignment.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/assignment.go)
  赋值及左值节点。
- [ast/simplify.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/simplify.go)
  常量折叠。
- [ast/operator](/Users/dai/Projects/用go语言自制编译器/neo-monkey/ast/operator)
  运算符枚举。

### `lower-ast/`

中间层。把结构化 AST 变成更利于执行器编译的形式。

- [lower-ast/ast.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/ast.go)
  `FunctionProto`、CFG、后缀序列等核心数据结构。
- [lower-ast/ops.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/ops.go)
  lowering 之后的操作定义。
- [lower-ast/hash.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/hash.go)
  结构化 hash，给函数原型缓存和复用服务。
- [lower-ast/convert-ast](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-ast)
  总入口，把 AST 节点分发给 statement / expression lowering。
- [lower-ast/convert-statement](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-statement)
  语句转 CFG。
- [lower-ast/convert-expression](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/convert-expression)
  表达式转后缀序列。
- [lower-ast/re-arrange](/Users/dai/Projects/用go语言自制编译器/neo-monkey/lower-ast/re-arrange)
  CFG block 重排，让 false 分支尽量顺序落在后面，减少额外跳转。

### `engine/`

运行时执行器。

- [engine/model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/model.go)
  同步执行模型、环境、闭包对象和混合操作定义。
- [engine/async_model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/async_model.go)
  异步执行模型，支持 `yield`、await 子任务和恢复。
- [engine/compile](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile)
  从 `lower-ast` 到 `HybirdOperation` 的编译器，是项目最关键的一层。

`engine/compile` 内部再细分为：

- [engine/compile/compile_fn.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/compile_fn.go)
  函数原型编译、符号槽位布局、闭包 free mapping。
- [engine/compile/compile_ast.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/compile_ast.go)
  序列、CFG、整棵函数图的编译主流程。
- [engine/compile/compile_op.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/compile_op.go)
  单个 lowering 操作如何变成可执行图节点。
- [engine/compile/compile_aux.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/compile_aux.go)
  辅助槽桥接、fallback、lookup 生成。
- [engine/compile/compile_prefix_gen.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/compile_prefix_gen.go)
  生成的前缀操作编译代码。
- [engine/compile/compile_infix_gen.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/compile_infix_gen.go)
  生成的中缀操作编译代码。
- [engine/compile/internal/gen_prefix](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/internal/gen_prefix)
  `go generate` 用的模板生成器。
- [engine/compile/internal/gen_infix](/Users/dai/Projects/用go语言自制编译器/neo-monkey/engine/compile/internal/gen_infix)
  `go generate` 用的模板生成器。

### `async/`

协程/事件循环基础设施，不依赖 `xex` AST，本身可以单独理解。

- [async/async_func_model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async/async_func_model.go)
  `Step`、`Handle`、`YieldReason`、`Future` 定义。
- [async/loop_model.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async/loop_model.go)
  事件循环和任务推进逻辑。
- [async/runner.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async/runner.go)
  对外更直接的 runner 包装。
- [async/heap.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/async/heap.go)
  sleep 队列用的小根堆。

### `object/`

运行时对象表示和宿主桥接层。

- [object/object.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/object/object.go)
  `Box`、基础对象、闭包/宿主函数桥接、操作符分发。
- [object/go_spec.go](/Users/dai/Projects/用go语言自制编译器/neo-monkey/object/go_spec.go)
  Go 宿主值与 `Box` 的互转。

注意：这一层目前还没有做完。

### `cmd/`

小型实验或分析程序。

- [cmd/profile-compiled-fib35-runtime](/Users/dai/Projects/用go语言自制编译器/neo-monkey/cmd/profile-compiled-fib35-runtime)
  编译并执行 `fib(35)`，生成运行时 pprof 数据，用来观察当前执行模型的热点。

## 现有验证方式

这个仓库现在主要靠测试和实验入口来验证行为：

- `go test ./...`
- `go run ./cmd/profile-compiled-fib35-runtime -n 3`

其中后者不是正式 benchmark，只是当前仓库内用于观察运行时开销和 profile 的入口。


