package wizcompiler

import (
	"fmt"

	"github.com/itchio/wizardry/wizardry/wizparser"
)

func switchify(node *ruleNode) *ruleNode {
	var lastChild *ruleNode
	var streak []*ruleNode

	var newChildren []*ruleNode

	endStreak := func() {
		switch len(streak) {
		case 0:
			return
		case 1:
			newChildren = append(newChildren, streak[0])
		default:
			model := streak[0].rule.Kind.Data.(*wizparser.IntegerKind)
			sk := &wizparser.SwitchKind{
				ByteWidth:  model.ByteWidth,
				Endianness: model.Endianness,
				Signed:     model.Signed,
			}
			for _, child := range streak {
				ik := child.rule.Kind.Data.(*wizparser.IntegerKind)
				sk.Cases = append(sk.Cases, &wizparser.SwitchCase{
					Description: child.rule.Description,
					Value:       ik.Value,
				})
			}
			newChildren = append(newChildren, &ruleNode{
				id: streak[0].id,
				rule: wizparser.Rule{
					Kind: wizparser.Kind{
						Family: wizparser.KindFamilySwitch,
						Data:   sk,
					},
					Level:  streak[0].rule.Level,
					Offset: streak[0].rule.Offset,
					Line:   fmt.Sprintf("(switch generated from %d integer tests)", len(streak)),
				},
			})
		}
		streak = nil
	}

	for _, childIn := range node.children {
		child := switchify(childIn)

		candidate := false

		if child.rule.Kind.Family == wizparser.KindFamilyInteger && len(child.children) == 0 {
			ik, _ := child.rule.Kind.Data.(*wizparser.IntegerKind)
			if ik.IntegerTest == wizparser.IntegerTestEqual && !ik.DoAnd && ik.AdjustmentType == wizparser.AdjustmentNone {
				candidate = true
			}
		}

		if !candidate {
			endStreak()
			newChildren = append(newChildren, child)
		} else {
			if len(streak) > 0 {
				if !lastChild.rule.Offset.Equals(child.rule.Offset) {
					endStreak()
				}
				ik, _ := child.rule.Kind.Data.(*wizparser.IntegerKind)
				jk, _ := lastChild.rule.Kind.Data.(*wizparser.IntegerKind)
				if ik.ByteWidth != jk.ByteWidth {
					endStreak()
				}
				if ik.Signed != jk.Signed {
					endStreak()
				}
			}
			streak = append(streak, child)
		}

		lastChild = child
	}

	endStreak()

	node.children = newChildren

	return node
}
