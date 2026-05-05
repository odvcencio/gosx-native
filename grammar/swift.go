package grammar

import (
	"sync"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammargen"
)

var (
	swiftGSXOnce sync.Once
	swiftGSXLang *gotreesitter.Language
	swiftGSXErr  error
)

// SwiftGSXGrammar extends Swift with JSX-style markup as expressions.
// File extension: .swift.gsx.
func SwiftGSXGrammar() *grammargen.Grammar {
	return grammargen.ExtendGrammar("swift_gsx", grammargen.SwiftGrammar(), func(g *grammargen.Grammar) {
		g.SetExternals(grammargen.Sym("jsx_text"))

		g.Define("source_file",
			grammargen.Repeat(grammargen.Choice(
				grammargen.Sym("struct_declaration"),
				grammargen.Sym("function_declaration"),
			)))

		g.Define("struct_declaration",
			grammargen.Seq(
				grammargen.Str("struct"),
				grammargen.Field("name", grammargen.Sym("type_identifier")),
				grammargen.Str("{"),
				grammargen.Repeat(grammargen.Field("members", grammargen.Sym("property_declaration"))),
				grammargen.Str("}"),
			))

		g.Define("function_declaration",
			grammargen.Seq(
				grammargen.Str("func"),
				grammargen.Field("name", grammargen.Sym("simple_identifier")),
				grammargen.Sym("function_value_parameters"),
				grammargen.Optional(grammargen.Seq(
					grammargen.Str("->"),
					grammargen.Field("return_type", grammargen.Sym("type_identifier")),
				)),
				grammargen.Field("body", grammargen.Sym("function_body")),
			))

		g.Define("function_value_parameters",
			grammargen.Seq(
				grammargen.Str("("),
				grammargen.Optional(grammargen.CommaSep1(grammargen.Sym("function_parameter"))),
				grammargen.Str(")"),
			))

		g.Define("function_parameter",
			grammargen.Seq(
				grammargen.Field("name", grammargen.Sym("simple_identifier")),
				grammargen.Str(":"),
				grammargen.Field("type", grammargen.Sym("type_identifier")),
			))

		g.Define("function_body",
			grammargen.Seq(
				grammargen.Str("{"),
				grammargen.Repeat(grammargen.Sym("_local_statement")),
				grammargen.Str("}"),
			))

		g.Define("_local_statement",
			grammargen.Choice(
				grammargen.Sym("property_declaration"),
				grammargen.Sym("control_transfer_statement"),
				grammargen.Sym("_expression"),
			))

		g.Define("property_declaration",
			grammargen.Choice(
				grammargen.Seq(
					grammargen.Str("var"),
					grammargen.Field("name", grammargen.Sym("simple_identifier")),
					grammargen.Str(":"),
					grammargen.Field("type", grammargen.Sym("type_identifier")),
				),
				grammargen.Seq(
					grammargen.Str("let"),
					grammargen.Field("name", grammargen.Sym("simple_identifier")),
					grammargen.Str("="),
					grammargen.Field("value", grammargen.Sym("_expression")),
				),
			))

		g.Define("control_transfer_statement",
			grammargen.Seq(
				grammargen.Str("return"),
				grammargen.Field("result", grammargen.Sym("_expression")),
			))

		g.Define("_expression",
			grammargen.Choice(
				grammargen.Sym("jsx_element"),
				grammargen.Sym("jsx_self_closing_element"),
				grammargen.Sym("jsx_fragment"),
				grammargen.Sym("binary_expression"),
				grammargen.Sym("call_expression"),
				grammargen.Sym("navigation_expression"),
				grammargen.Sym("integer_literal"),
				grammargen.Sym("string_literal"),
				grammargen.Sym("simple_identifier"),
			))

		g.Define("binary_expression",
			grammargen.PrecLeft(1,
				grammargen.Seq(
					grammargen.Field("left", grammargen.Sym("_expression")),
					grammargen.Field("operator", grammargen.Choice(
						grammargen.Str("+"),
						grammargen.Str("-"),
					)),
					grammargen.Field("right", grammargen.Sym("_expression")),
				),
			))

		g.Define("navigation_expression",
			grammargen.PrecLeft(3,
				grammargen.Seq(
					grammargen.Field("target", grammargen.Sym("_expression")),
					grammargen.Str("."),
					grammargen.Field("suffix", grammargen.Sym("simple_identifier")),
				),
			))

		g.Define("call_expression",
			grammargen.PrecLeft(4,
				grammargen.Seq(
					grammargen.Field("function", grammargen.Sym("_expression")),
					grammargen.Str("("),
					grammargen.Optional(grammargen.CommaSep1(grammargen.Field("argument", grammargen.Sym("_expression")))),
					grammargen.Str(")"),
				),
			))

		// The generated CST keeps jsx_* node names for compatibility with gosx's
		// existing lowering and tooling conventions.
		g.Define("jsx_identifier",
			grammargen.Pat(`[a-zA-Z_][a-zA-Z0-9_]*`))

		g.Define("jsx_html_tag_name",
			grammargen.Pat(`[a-z][a-zA-Z0-9_-]*`))

		g.Define("jsx_dotted_name",
			grammargen.Seq(
				grammargen.Field("object", grammargen.Sym("jsx_identifier")),
				grammargen.Str("."),
				grammargen.Field("property", grammargen.Sym("jsx_identifier")),
			))

		g.Define("jsx_tag_name",
			grammargen.Choice(
				grammargen.Sym("jsx_dotted_name"),
				grammargen.Sym("jsx_html_tag_name"),
				grammargen.Sym("jsx_identifier"),
			))

		g.Define("jsx_string_literal",
			grammargen.Token(grammargen.Seq(
				grammargen.Str(`"`),
				grammargen.Pat(`[^"]*`),
				grammargen.Str(`"`),
			)))

		g.Define("jsx_expression_container",
			grammargen.Seq(
				grammargen.Str("{"),
				grammargen.Field("expression", grammargen.Sym("_expression")),
				grammargen.Str("}"),
			))

		g.Define("jsx_attr_name",
			grammargen.Pat(`[a-zA-Z_][a-zA-Z0-9_:-]*`))

		g.Define("jsx_attribute",
			grammargen.Choice(
				grammargen.Seq(
					grammargen.Field("name", grammargen.Sym("jsx_attr_name")),
					grammargen.Str("="),
					grammargen.Field("value", grammargen.Sym("jsx_expression_container")),
				),
				grammargen.Seq(
					grammargen.Field("name", grammargen.Sym("jsx_attr_name")),
					grammargen.Str("="),
					grammargen.Field("value", grammargen.Sym("jsx_string_literal")),
				),
				grammargen.Seq(
					grammargen.Field("name", grammargen.Sym("jsx_attr_name")),
				),
			))

		g.Define("jsx_spread_attribute",
			grammargen.Seq(
				grammargen.Str("{"),
				grammargen.Str("..."),
				grammargen.Field("expression", grammargen.Sym("_expression")),
				grammargen.Str("}"),
			))

		g.Define("_jsx_child",
			grammargen.Choice(
				grammargen.Sym("jsx_element"),
				grammargen.Sym("jsx_self_closing_element"),
				grammargen.Sym("jsx_expression_container"),
				grammargen.Sym("jsx_fragment"),
				grammargen.Sym("jsx_text"),
			))

		g.Define("jsx_opening_element",
			grammargen.Seq(
				grammargen.Str("<"),
				grammargen.Field("name", grammargen.Sym("jsx_tag_name")),
				grammargen.Repeat(grammargen.Field("attributes", grammargen.Choice(
					grammargen.Sym("jsx_attribute"),
					grammargen.Sym("jsx_spread_attribute"),
				))),
				grammargen.Str(">"),
			))

		g.Define("jsx_closing_element",
			grammargen.Seq(
				grammargen.Str("<"),
				grammargen.Str("/"),
				grammargen.Field("name", grammargen.Sym("jsx_tag_name")),
				grammargen.Str(">"),
			))

		g.Define("jsx_element",
			grammargen.Seq(
				grammargen.Field("open", grammargen.Sym("jsx_opening_element")),
				grammargen.Repeat(grammargen.Field("children", grammargen.Sym("_jsx_child"))),
				grammargen.Field("close", grammargen.Sym("jsx_closing_element")),
			))

		g.Define("jsx_self_closing_element",
			grammargen.Seq(
				grammargen.Str("<"),
				grammargen.Field("name", grammargen.Sym("jsx_tag_name")),
				grammargen.Repeat(grammargen.Field("attributes", grammargen.Choice(
					grammargen.Sym("jsx_attribute"),
					grammargen.Sym("jsx_spread_attribute"),
				))),
				grammargen.Str("/"),
				grammargen.Str(">"),
			))

		g.Define("jsx_fragment",
			grammargen.Seq(
				grammargen.Str("<"),
				grammargen.Str(">"),
				grammargen.Repeat(grammargen.Field("children", grammargen.Sym("_jsx_child"))),
				grammargen.Str("<"),
				grammargen.Str("/"),
				grammargen.Str(">"),
			))

	})
}

func SwiftGSXLanguage() (*gotreesitter.Language, error) {
	swiftGSXOnce.Do(func() {
		swiftGSXLang, swiftGSXErr = grammargen.GenerateLanguage(SwiftGSXGrammar())
		if swiftGSXErr == nil && swiftGSXLang != nil {
			swiftGSXLang.ExternalScanner = newSwiftScanner(swiftGSXLang)
		}
	})
	return swiftGSXLang, swiftGSXErr
}
