package re_arrange

import lower_ast "xex/lower-ast"

type ReArrangedBlock struct {
	Block   lower_ast.CFGBlock[lower_ast.SuffixOperatesSequence]
	BlockID int32
}

// 尝试将一个block的 false 分支跳转直接放在其后
func ReArrangeBlock(astt lower_ast.LowerAst) []ReArrangedBlock {
	if len(astt) == 0 {
		return nil
	}
	blocks := make([]ReArrangedBlock, 0, len(astt))
	arranged := make(map[int32]struct{}, len(astt))

	appendFalseChain := func(start int32) {
		current := start
		for {
			if _, exists := arranged[current]; exists {
				return
			}
			block := astt[current]
			blocks = append(blocks, ReArrangedBlock{Block: block, BlockID: current})
			arranged[current] = struct{}{}
			if block.IsNoCondJump {
				return
			}
			next := block.Aux
			if _, exists := arranged[next]; exists {
				return
			}
			current = next
		}
	}

	appendFalseChain(0)
	for bi, block := range astt {
		if _, exists := arranged[int32(bi)]; exists {
			continue
		}
		if !block.IsNoCondJump {
			appendFalseChain(int32(bi))
		}
	}
	for bi, block := range astt {
		if _, exists := arranged[int32(bi)]; exists {
			continue
		}
		blocks = append(blocks, ReArrangedBlock{Block: block, BlockID: int32(bi)})
		arranged[int32(bi)] = struct{}{}
	}
	return blocks
}
