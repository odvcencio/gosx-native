package grammar

import (
	gotreesitter "github.com/odvcencio/gotreesitter"
)

type swiftScanner struct {
	lang      *gotreesitter.Language
	textIndex int
}

func newSwiftScanner(lang *gotreesitter.Language) *swiftScanner {
	return &swiftScanner{
		lang:      lang,
		textIndex: externalIndex(lang, "jsx_text"),
	}
}

func (s *swiftScanner) Create() any { return nil }

func (s *swiftScanner) Destroy(payload any) {}

func (s *swiftScanner) Serialize(payload any, buf []byte) int { return 0 }

func (s *swiftScanner) Deserialize(payload any, buf []byte) {}

func (s *swiftScanner) SupportsIncrementalReuse() bool { return true }

func (s *swiftScanner) Scan(payload any, lexer *gotreesitter.ExternalLexer, validSymbols []bool) bool {
	if s == nil || s.lang == nil {
		return false
	}
	if swiftValid(validSymbols, s.textIndex) && s.scanGSXText(lexer) {
		return true
	}
	return false
}

func (s *swiftScanner) scanGSXText(lexer *gotreesitter.ExternalLexer) bool {
	consumed := 0
	for {
		ch := lexer.Lookahead()
		if ch == 0 || ch == '<' || ch == '{' {
			break
		}
		lexer.Advance(false)
		consumed++
	}
	if consumed == 0 {
		return false
	}
	lexer.MarkEnd()
	lexer.SetResultSymbol(s.lang.ExternalSymbols[s.textIndex])
	return true
}

func externalIndex(lang *gotreesitter.Language, name string) int {
	if lang == nil {
		return -1
	}
	for i, sym := range lang.ExternalSymbols {
		if int(sym) < len(lang.SymbolNames) && lang.SymbolNames[sym] == name {
			return i
		}
	}
	return -1
}

func swiftValid(vs []bool, idx int) bool { return idx >= 0 && idx < len(vs) && vs[idx] }

func StripSwiftGSXAttributeExpression(text string) string {
	if len(text) >= 2 && text[0] == '{' && text[len(text)-1] == '}' {
		return text[1 : len(text)-1]
	}
	return text
}
