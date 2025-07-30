package utils

import (
	"errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type ASTNode interface {
}

type Parser interface {
	Parse(markdown string) (ASTNode, error)
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

func (p *DefaultMarkdownParser) Parse(markdown string) (ASTNode, error) {
	source := []byte(markdown)
	reader := text.NewReader(source)
	node := p.engine.Parser().Parse(reader)

	if node == nil || node.ChildCount() == 0 {
		return nil, errors.New("invalid or empty markdown")
	}

	return node, nil
}
