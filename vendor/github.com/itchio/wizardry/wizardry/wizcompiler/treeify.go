package wizcompiler

import "github.com/itchio/wizardry/wizardry/wizparser"

func treeify(rules []wizparser.Rule) []*ruleNode {
	var rootNodes []*ruleNode
	var nodeStack []*ruleNode
	var idSeed int64

	for _, rule := range rules {
		node := &ruleNode{
			id:   idSeed,
			rule: rule,
		}
		idSeed++

		if rule.Level > 0 {
			parent := nodeStack[rule.Level-1]
			parent.children = append(parent.children, node)
		} else {
			rootNodes = append(rootNodes, node)
		}

		nodeStack = append(nodeStack[0:rule.Level], node)
	}

	return rootNodes
}
