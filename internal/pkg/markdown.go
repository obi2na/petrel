package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"reflect"
	"regexp"
	"strings"
)

// ----- Interfaces and Types -----

type Parser interface {
	Parse(markdown string) (ast.Node, []byte, error)
}

type DefaultMarkdownParser struct {
	engine goldmark.Markdown
}

func NewDefaultMarkdownParser() *DefaultMarkdownParser {
	return &DefaultMarkdownParser{
		engine: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		),
	}
}

func (p *DefaultMarkdownParser) Parse(markdown string) (ast.Node, []byte, error) {
	source := []byte(markdown)
	reader := text.NewReader(source)
	node := p.engine.Parser().Parse(reader)

	if node == nil || node.ChildCount() == 0 {
		return nil, nil, errors.New("invalid or empty markdown")
	}

	return node, source, nil
}

type LintWarning struct {
	Line    int    `json:"Line"`
	Message string `json:"Message"`
}

type MarkdownLinter interface {
	Lint(doc ast.Node, source []byte) ([]LintWarning, error)
}

type PetrelMarkdownLinter struct {
	ruleMap map[reflect.Type]func(ast.Node, []byte, []int) []LintWarning
}

// ----- Global Regex -----

var (
	headingNoSpaceRe        = regexp.MustCompile(`^#{1,6}[^\s#]`)
	trailingHashInHeadingRe = regexp.MustCompile(`^#{1,6}.*[^#]\s*#+$`)
	unclosedParenLinkRe     = regexp.MustCompile(`$begin:math:display$[^$end:math:display$]+\]\([^)]+$`)
	unclosedAsteriskRe      = regexp.MustCompile(`\*[^*]*$`)
)

// ----- Linter Core -----

func NewPetrelMarkdownLinter() *PetrelMarkdownLinter {
	l := &PetrelMarkdownLinter{
		ruleMap: make(map[reflect.Type]func(ast.Node, []byte, []int) []LintWarning),
	}
	l.registerRules()
	return l
}

func (l *PetrelMarkdownLinter) registerRules() {
	l.ruleMap[reflect.TypeOf(&ast.Heading{})] = headingRules
	l.ruleMap[reflect.TypeOf(&ast.List{})] = listRules
	l.ruleMap[reflect.TypeOf(&ast.Text{})] = textRules
	l.ruleMap[reflect.TypeOf(&ast.Link{})] = linkRules
}

func extractHeadings(doc ast.Node, source []byte) {
	switch node := doc.(type) {
	case *ast.Document:
		fmt.Println("Document node found")

		for n := node.FirstChild(); n != nil; n = n.NextSibling() {
			if heading, ok := n.(*ast.Heading); ok {
				fmt.Println("Heading found")
				var headingText strings.Builder

				// Case 2: Use inline child nodes (formatted heading)
				for c := heading.FirstChild(); c != nil; c = c.NextSibling() {
					switch t := c.(type) {
					case *ast.Text:
						headingText.Write(t.Segment.Value(source))
					}
				}

				fmt.Printf("Heading (level %d): %s\n", heading.Level, headingText.String())
			}
		}
	}
}

func (l *PetrelMarkdownLinter) Lint(doc ast.Node, source []byte) ([]LintWarning, error) {
	var warnings []LintWarning
	lineOffsets := buildLineOffsets(source)

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if ruleFn, ok := l.ruleMap[reflect.TypeOf(n)]; ok {
			warnings = append(warnings, ruleFn(n, source, lineOffsets)...)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return warnings, err
	}

	return warnings, nil
}

// ----- Rule Implementations -----

func headingRules(n ast.Node, source []byte, lineOffsets []int) []LintWarning {
	var warnings []LintWarning
	heading := n.(*ast.Heading)

	if heading.Level > 3 && heading.Lines().Len() > 0 {
		line := getLine(heading.Lines().At(0).Start, lineOffsets)
		warnings = append(warnings, LintWarning{
			Line:    line,
			Message: "Avoid using deeply nested headings (h4 or deeper)",
		})
	}

	return warnings
}

func listRules(n ast.Node, _ []byte, lineOffsets []int) []LintWarning {
	var warnings []LintWarning
	list := n.(*ast.List)
	if !list.IsTight && list.Lines().Len() > 0 {
		line := getLine(list.Lines().At(0).Start, lineOffsets)
		warnings = append(warnings, LintWarning{
			Line:    line,
			Message: "Loose lists may reduce readability",
		})
	}
	return warnings
}

func textRules(n ast.Node, source []byte, lineOffsets []int) []LintWarning {
	var warnings []LintWarning
	textNode := n.(*ast.Text)
	txt := string(textNode.Segment.Value(source))
	line := getLine(textNode.Segment.Start, lineOffsets)

	if strings.Contains(txt, "TODO") {
		warnings = append(warnings, LintWarning{Line: line, Message: "Contains unfinished content (TODO)"})
	}
	if strings.Contains(txt, "  ") {
		warnings = append(warnings, LintWarning{Line: line, Message: "Avoid multiple consecutive spaces"})
	}
	if strings.Count(txt, "*")%2 != 0 || unclosedAsteriskRe.MatchString(txt) {
		warnings = append(warnings, LintWarning{Line: line, Message: "Unclosed italic/bold formatting"})
	}
	if headingNoSpaceRe.MatchString(txt) {
		warnings = append(warnings, LintWarning{Line: line, Message: "Missing space after hash in heading"})
	}
	if trailingHashInHeadingRe.MatchString(txt) {
		warnings = append(warnings, LintWarning{Line: line, Message: "Avoid trailing '#' in heading"})
	}
	if unclosedParenLinkRe.MatchString(txt) {
		warnings = append(warnings, LintWarning{Line: line, Message: "Malformed link (missing closing parenthesis)"})
	}

	return warnings
}

func linkRules(n ast.Node, _ []byte, lineOffsets []int) []LintWarning {
	var warnings []LintWarning
	link := n.(*ast.Link)
	if !strings.HasPrefix(string(link.Destination), "http") && link.Lines().Len() > 0 {
		line := getLine(link.Lines().At(0).Start, lineOffsets)
		warnings = append(warnings, LintWarning{
			Line:    line,
			Message: "Link does not have a valid URL scheme",
		})
	}
	return warnings
}

// ----- Utility -----

func buildLineOffsets(source []byte) []int {
	lines := bytes.Split(source, []byte("\n"))
	offset := 0
	offsets := make([]int, len(lines))
	for i, line := range lines {
		offsets[i] = offset
		offset += len(line) + 1
	}
	return offsets
}

func getLine(offset int, lineOffsets []int) int {
	for i := len(lineOffsets) - 1; i >= 0; i-- {
		if offset >= lineOffsets[i] {
			return i + 1
		}
	}
	return 1
}
