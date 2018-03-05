package hades

import "strings"

func FromDBName(dbname string) string {
	var resTokens []string
	tokens := strings.Split(dbname, "_")
	for _, token := range tokens {
		resToken := strings.ToUpper(token[:1]) + token[1:]
		if resToken == "Id" {
			resToken = "ID"
		}
		resTokens = append(resTokens, resToken)
	}
	return strings.Join(resTokens, "")
}
