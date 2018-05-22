package wizcompiler

import "github.com/itchio/wizardry/wizardry/wizparser"

func computePagesUsage(book wizparser.Spellbook) map[string]*PageUsage {
	// look at all rules to see which pages are used, and whether they're used
	// in normal endianness or swapped endianness
	usages := make(map[string]*PageUsage)
	usages[""] = &PageUsage{
		EmitNormal: true,
	}

	for _, rules := range book {
		for _, rule := range rules {
			if rule.Kind.Family == wizparser.KindFamilyUse {
				uk, _ := rule.Kind.Data.(*wizparser.UseKind)
				var usage *PageUsage
				var ok bool
				if usage, ok = usages[uk.Page]; !ok {
					usage = &PageUsage{}
					usages[uk.Page] = usage
				}

				if uk.SwapEndian {
					usage.EmitSwapped = true
				} else {
					usage.EmitNormal = true
				}
			}
		}
	}

	return usages
}
