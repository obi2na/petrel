package notion

import (
	"context"
	"fmt"
	"github.com/jomei/notionapi"
	"github.com/obi2na/petrel/internal/logger"
	utils "github.com/obi2na/petrel/internal/pkg"
	"github.com/yuin/goldmark/ast"
	"go.uber.org/zap"
	"reflect"
	"strings"
)

type BlockWithChildren struct {
	Block    notionapi.Block
	Children []*BlockWithChildren
}

func (b *BlockWithChildren) AddChild(ctx context.Context, child *BlockWithChildren) {
	if b == nil {
		logger.With(ctx).Warn("Attempted to add child to nil parent block")
		return
	}
	logger.With(ctx).Debug("Adding child block")
	b.Children = append(b.Children, child)
}

type mappingContext struct {
	currentParent *BlockWithChildren               // current parent container (nil if non-parent container)
	stack         *utils.Stack[*BlockWithChildren] // stack to track nested parent container
	result        []*BlockWithChildren             // Final list of top-level blocks
}

func newMappingContext() *mappingContext {
	return &mappingContext{
		stack:  utils.NewStack[*BlockWithChildren](),
		result: []*BlockWithChildren{},
	}
}

func (c *mappingContext) addBlock(ctx context.Context, b *BlockWithChildren) {
	if b == nil {
		logger.With(ctx).Warn("Attempted to add nil block to context")
		return
	}
	if c.currentParent != nil {
		c.currentParent.AddChild(ctx, b)
	} else {
		c.result = append(c.result, b)
	}
}

func (c *mappingContext) pushParent(ctx context.Context, b *BlockWithChildren) {
	if b == nil {
		logger.With(ctx).Warn("Attempted to push nil parent to stack")
		return
	}
	logger.With(ctx).Debug("Pushing parent to stack")
	c.stack.Push(b)
	c.currentParent = b
}

func (c *mappingContext) popParent() {
	c.stack.Pop()
	if top, ok := c.stack.Peek(); ok {
		c.currentParent = top
	} else {
		c.currentParent = nil
	}
}

type MarkdownToNotionMapper interface {
	Map(ctx context.Context, doc ast.Node, source []byte) ([]*BlockWithChildren, error)
}

type MapperFunc func(ctx context.Context, node ast.Node, source []byte, mc *mappingContext) error
type PetrelMarkdownToNotionMapper struct {
	mapperMap map[reflect.Type]MapperFunc
}

func NewPetrelMarkdownToNotionMapper() *PetrelMarkdownToNotionMapper {
	return &PetrelMarkdownToNotionMapper{
		mapperMap: make(map[reflect.Type]MapperFunc),
	}
}

func (p *PetrelMarkdownToNotionMapper) RegisterMappers() {
	// TODO: register mappers here
	p.mapperMap[reflect.TypeOf(&ast.Document{})] = mapDocument
	p.mapperMap[reflect.TypeOf(&ast.Heading{})] = mapHeading
	p.mapperMap[reflect.TypeOf(&ast.Paragraph{})] = mapParagraph
	p.mapperMap[reflect.TypeOf(&ast.ListItem{})] = mapList
	p.mapperMap[reflect.TypeOf(&ast.Blockquote{})] = mapQuote
	p.mapperMap[reflect.TypeOf(&ast.FencedCodeBlock{})] = mapCodeBlock
	p.mapperMap[reflect.TypeOf(&ast.List{})] = mapDocument
}

func (p *PetrelMarkdownToNotionMapper) Map(ctx context.Context, doc ast.Node, source []byte) ([]*BlockWithChildren, error) {
	logger.With(ctx).Info("Mapping markdown to Notion blocks")
	mapCtx := newMappingContext()

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			if isParentBlock(n) {
				mapCtx.popParent()
			}
			return ast.WalkContinue, nil
		}

		fn, ok := p.mapperMap[reflect.TypeOf(n)]
		if !ok {
			// Don’t log inline types as unsupported
			logger.With(ctx).Debug("Unsupported block-level node", zap.String("node", n.Kind().String()))
			return ast.WalkContinue, nil
		}

		if err := fn(ctx, n, source, mapCtx); err != nil {
			logger.With(ctx).Error("Error mapping node", zap.Error(err), zap.String("node", n.Kind().String()))
		}

		return ast.WalkContinue, nil
	})
	if err != nil {
		logger.With(ctx).Error("Error walking AST", zap.Error(err))
		return nil, err
	}

	return mapCtx.result, nil
}

func isParentBlock(n ast.Node) bool {
	switch n.(type) {
	case *ast.List, *ast.ListItem, *ast.Blockquote:
		return true
	default:
		return false
	}
}

func extractText(n ast.Node, source []byte) string {
	var textBuilder strings.Builder

	var walk func(ast.Node)
	walk = func(node ast.Node) {
		switch v := node.(type) {
		case *ast.Text:
			textBuilder.Write(v.Segment.Value(source))

		case *ast.Emphasis:
			for child := v.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child)
			}

		case *ast.CodeSpan:
			text := string(v.Text(source))
			textBuilder.WriteString(text)

		case *ast.Link, *ast.Image:
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child)
			}

		default:
			// Generic fallback for any inline container node
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child)
			}
		}
	}

	walk(n)
	return textBuilder.String()
}

func mapHeading(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	heading, ok := node.(*ast.Heading)
	if !ok {
		err := fmt.Errorf("expected *ast.Heading but got %T", node)
		logger.With(ctx).Error("error casting", zap.Error(err))
		return err
	}
	text := extractText(heading, source)

	var block notionapi.Block
	switch heading.Level {
	case 1:
		block = &notionapi.Heading1Block{
			BasicBlock: notionapi.BasicBlock{
				Type:   notionapi.BlockTypeHeading1,
				Object: notionapi.ObjectTypeBlock,
			},
			Heading1: notionapi.Heading{
				RichText: []notionapi.RichText{
					{
						Type: notionapi.ObjectTypeText,
						Text: &notionapi.Text{
							Content: text,
						},
					},
				},
			},
		}
	case 2:
		block = &notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Type:   notionapi.BlockTypeHeading2,
				Object: notionapi.ObjectTypeBlock,
			},
			Heading2: notionapi.Heading{
				RichText: []notionapi.RichText{
					{
						Type: notionapi.ObjectTypeText,
						Text: &notionapi.Text{
							Content: text,
						},
					},
				},
			},
		}
	default:
		block = &notionapi.Heading3Block{
			BasicBlock: notionapi.BasicBlock{
				Type:   notionapi.BlockTypeHeading3,
				Object: notionapi.ObjectTypeBlock,
			},
			Heading3: notionapi.Heading{
				RichText: []notionapi.RichText{
					{
						Type: notionapi.ObjectTypeText,
						Text: &notionapi.Text{
							Content: text,
						},
					},
				},
			},
		}
	}

	ctxMap.addBlock(ctx, &BlockWithChildren{Block: block})
	return nil
}

func mapParagraph(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	text := extractText(node, source)

	block := &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeParagraph,
		},
		Paragraph: notionapi.Paragraph{
			RichText: []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{
						Content: text,
					},
				},
			},
		},
	}

	ctxMap.addBlock(ctx, &BlockWithChildren{Block: block})
	return nil
}

func mapBulletedList(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	item, ok := node.(*ast.ListItem)
	if !ok {
		err := fmt.Errorf("expected *ast.ListItem but got %T", node)
		logger.With(ctx).Error("error casting", zap.Error(err))
		return err
	}
	text := extractText(item, source)

	block := &notionapi.BulletedListItemBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeBulletedListItem,
		},
		BulletedListItem: notionapi.ListItem{
			RichText: []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{
						Content: text,
					},
				},
			},
		},
	}

	b := &BlockWithChildren{Block: block}
	ctxMap.addBlock(ctx, b)

	// If this item contains nested content, push to stack
	if isParentBlock(item) {
		ctxMap.pushParent(ctx, b)
	}
	return nil
}

func mapQuote(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	text := extractText(node, source)

	block := &notionapi.QuoteBlock{
		BasicBlock: notionapi.BasicBlock{
			Type:   notionapi.BlockTypeQuote,
			Object: notionapi.ObjectTypeBlock,
		},
		Quote: notionapi.Quote{
			RichText: []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{
						Content: text,
					},
				},
			},
		},
	}

	ctxMap.addBlock(ctx, &BlockWithChildren{Block: block})
	return nil
}

func mapCodeBlock(ctx context.Context, node ast.Node, source []byte, mapCtx *mappingContext) error {
	codeBlock, ok := node.(*ast.FencedCodeBlock)
	if !ok {
		err := fmt.Errorf("error casting codeblock to FencedCodeBlock")
		logger.With(ctx).Error("casting error", zap.Error(err))
		return err
	}

	// Extract the code content
	var contentBuilder strings.Builder
	for i := 0; i < codeBlock.Lines().Len(); i++ {
		line := codeBlock.Lines().At(i)
		contentBuilder.Write(line.Value(source))
	}
	codeContent := contentBuilder.String()
	language := string(codeBlock.Language(source))

	// Create Notion code block
	block := &BlockWithChildren{

		Block: &notionapi.CodeBlock{
			BasicBlock: notionapi.BasicBlock{
				Type:   notionapi.BlockTypeCode,
				Object: notionapi.ObjectTypeBlock,
			},
			Code: notionapi.Code{
				RichText: []notionapi.RichText{
					{
						Type: "text",
						Text: &notionapi.Text{
							Content: codeContent,
						},
					},
				},
				Language: language,
			},
		},
	}

	// Add the block to the mapping context
	mapCtx.addBlock(ctx, block)
	return nil
}

func mapNumberedList(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	item, ok := node.(*ast.ListItem)
	if !ok {
		err := fmt.Errorf("expected *ast.ListItem but got %T", node)
		logger.With(ctx).Error("casting error", zap.Error(err))
		return err
	}
	text := extractText(item, source)

	block := &notionapi.NumberedListItemBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeNumberedListItem,
		},
		NumberedListItem: notionapi.ListItem{
			RichText: []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{
						Content: text,
					},
				},
			},
		},
	}

	b := &BlockWithChildren{Block: block}
	ctxMap.addBlock(ctx, b)

	if isParentBlock(item) {
		ctxMap.pushParent(ctx, b)
	}

	return nil
}

func mapList(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	item, ok := node.(*ast.ListItem)
	if !ok {
		err := fmt.Errorf("expected *ast.ListItem, got %T", node)
		logger.With(ctx).Error("invalid list item node", zap.Error(err))
		return err
	}

	parent := item.Parent()
	if parent == nil {
		err := fmt.Errorf("list item has no parent")
		logger.With(ctx).Error("list item missing parent", zap.Error(err))
		return err
	}

	list, ok := parent.(*ast.List)
	if !ok {
		err := fmt.Errorf("list item parent is not *ast.List: %T", parent)
		logger.With(ctx).Error("invalid parent type for list item", zap.Error(err))
		return err
	}

	if list.IsOrdered() {
		return mapNumberedList(ctx, node, source, ctxMap)
	}
	return mapBulletedList(ctx, node, source, ctxMap)
}

func mapDocument(ctx context.Context, node ast.Node, source []byte, ctxMap *mappingContext) error {
	// No-op mapper — just ensures children are walked
	return nil
}
