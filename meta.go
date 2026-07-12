// Package meta is an extension for goldmark.
//
// This extension parses YAML metadata blocks and stores metadata to a
// document.
package meta

import (
	"bytes"
	"fmt"
	"io"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"

	"go.yaml.in/yaml/v4"
)

// An MetaBlock struct represents a meta block node in the AST.
type MetaBlock struct {
	ast.BaseBlock

	// Value holds the raw content of this block.
	Value text.Lines

	data data
}

// Dump implements Node.Dump.
func (n *MetaBlock) Dump(source []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, map[string]any{
		"Value": string(n.Value.Bytes(source)),
	})
}

// KindMetaBlock is a NodeKind of the MetaBlock node.
var KindMetaBlock = ast.NewNodeKind("MetaBlock")

// Kind implements Node.Kind.
func (n *MetaBlock) Kind() ast.NodeKind {
	return KindMetaBlock
}

// NewMetaBlock returns a new MetaBlock node.
func NewMetaBlock() *MetaBlock {
	n := &MetaBlock{}
	n.Init(n)
	return n
}

type data struct {
	Map   map[string]any
	Items yaml.Node
	Error error
	Node  ast.Node
}

var contextKey = parser.NewContextKey()

type blockParser struct{}

var defaultBlockParser = &blockParser{}

// NewParser returns a BlockParser that can parse YAML metadata blocks.
func NewParser() parser.BlockParser {
	return defaultBlockParser
}

func isSeparator(line []byte) bool {
	line = util.TrimRightSpace(util.TrimLeftSpace(line))
	for i := 0; i < len(line); i++ {
		if line[i] != '-' {
			return false
		}
	}
	return true
}

func (b *blockParser) Trigger() []byte {
	return []byte{'-'}
}

func (b *blockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	linenum, _ := reader.Position()
	if linenum != 0 {
		return nil, parser.NoChildren
	}
	line, _ := reader.PeekLine()
	if isSeparator(line) {
		reader.AdvanceToEOL()
		node := NewMetaBlock()
		pc.Set(contextKey, node)

		return node, parser.NoChildren
	}
	return nil, parser.NoChildren
}

func (b *blockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if isSeparator(line) && !util.IsBlank(line) {
		reader.AdvanceToEOL()
		return parser.Close
	}
	node.(*MetaBlock).Value.AppendSegment(segment)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (b *blockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	d := &node.(*MetaBlock).data
	lines := node.(*MetaBlock).Value.Segments()
	var buf bytes.Buffer
	for _, segment := range lines {
		buf.Write(segment.Bytes(reader.Source()))
	}
	meta := map[string]any{}
	if err := yaml.Unmarshal(buf.Bytes(), &meta); err != nil {
		d.Error = err
	} else {
		d.Map = meta
		for k, v := range d.Map {
			node.OwnerDocument().AddMeta(k, v)
		}
		if err := yaml.Unmarshal(buf.Bytes(), &d.Items); err != nil {
			d.Error = err
		}
	}

}

func (b *blockParser) CanInterruptParagraph() bool {
	return false
}

func (b *blockParser) CanAcceptIndentedLine() bool {
	return false
}

type parserExt struct{}

type htmlRendererConfig struct {
	Table *tableConfig
}

// HTMLRendererOption configures the HTML renderer extension.
type HTMLRendererOption func(*htmlRendererConfig)

// TableLayout is a type for specifying the layout of the table rendering.
type TableLayout int

const (
	// TableLayoutColumns specifies that the table should be rendered in a column layout.
	TableLayoutColumns TableLayout = iota

	// TableLayoutRows specifies that the table should be rendered in a row layout.
	TableLayoutRows
)

type tableConfig struct {
	Layout TableLayout
}

// TableOption configures the table rendering options.
type TableOption func(*tableConfig)

// WithLayout sets the layout of the table rendering.
func WithLayout(direction TableLayout) TableOption {
	return func(c *tableConfig) {
		c.Layout = direction
	}
}

// WithTable renders parsed metadata as an HTML table before the document body.
func WithTable(opts ...TableOption) HTMLRendererOption {
	return func(c *htmlRendererConfig) {
		cfg := &tableConfig{
			Layout: TableLayoutColumns,
		}
		for _, opt := range opts {
			opt(cfg)
		}
		c.Table = cfg
	}
}

type metaHTMLRenderer struct {
	config htmlRendererConfig
	writer html.Writer
}

// Parser is a parser extension for goldmark.
var Parser = &parserExt{}

// HTMLRenderer is the default HTML renderer extension for metadata blocks.
var HTMLRenderer = NewHTMLRenderer()

// NewHTMLRenderer returns a new HTML renderer extension.
func NewHTMLRenderer(opts ...HTMLRendererOption) html.Extension {
	cfg := htmlRendererConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &metaHTMLRenderer{config: cfg}
}

func (e *parserExt) ParserOptions(_ *parser.Config) []parser.Option {
	return []parser.Option{
		parser.WithBlockParsers(
			util.Prioritized(NewParser(), 0),
		),
	}
}

func (r *metaHTMLRenderer) RendererOptions(cfg *html.Config) []html.Option {
	var opts []html.Option
	r.writer = cfg.Writer()
	opts = append(opts, html.WithNodeRenderer(KindMetaBlock, html.NodeRendererFunc(r.renderMetaBlock)))
	return opts
}

func (r *metaHTMLRenderer) renderMetaBlock(writer io.Writer, source []byte, n ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if entering {
		w := writer.(util.BufWriter)
		d := &n.(*MetaBlock).data
		if d.Error != nil {
			_, _ = fmt.Fprintf(w, "<!-- %v -->\n", d.Error)
			return ast.WalkSkipChildren, nil
		}

		if r.config.Table != nil {
			var keys []string
			mapping := d.Items.Content[0]
			for i := 0; i < len(mapping.Content); i += 2 {
				keys = append(keys, mapping.Content[i].Value)
			}
			switch r.config.Table.Layout {
			case TableLayoutColumns:
				_, _ = w.WriteString("<table>\n<thead>\n<tr>\n")
				for _, key := range keys {
					_, _ = w.WriteString("<th>")
					r.writer.WriteText(w, fmt.Append(nil, key))
					_, _ = w.WriteString("</th>\n")
				}
				_, _ = w.WriteString("</tr>\n</thead>\n<tbody>\n<tr>\n")
				for _, key := range keys {
					_, _ = w.WriteString("<td>")
					r.writer.WriteHTML(w, []byte(valueToHTML(d.Map[key])))
					_, _ = w.WriteString("</td>\n")
				}
				_, _ = w.WriteString("</tr>\n</tbody>\n</table>\n")
			case TableLayoutRows:
				_, _ = w.WriteString("<table>\n<tbody>\n")
				for _, key := range keys {
					_, _ = w.WriteString("<tr>\n<td>")
					r.writer.WriteText(w, fmt.Append(nil, key))
					_, _ = w.WriteString("</td>\n<td>")
					r.writer.WriteHTML(w, []byte(valueToHTML(d.Map[key])))
					_, _ = w.WriteString("</td>\n</tr>\n")
				}
				_, _ = w.WriteString("</tbody>\n</table>\n")
			}
		}
	}
	return ast.WalkSkipChildren, nil
}

func valueToHTML(v any) []byte {
	switch v := v.(type) {
	case []any:
		var buf bytes.Buffer
		buf.WriteString("<ul>")
		for _, item := range v {
			buf.WriteString("<li>")
			buf.Write(valueToHTML(item))
			buf.WriteString("</li>")
		}
		buf.WriteString("</ul>")
		return buf.Bytes()
	case map[string]any:
		var buf bytes.Buffer
		buf.WriteString("<ul>")
		for k, v := range v {
			buf.WriteString("<li>")
			buf.WriteString(k)
			buf.WriteString(": ")
			buf.Write(valueToHTML(v))
			buf.WriteString("</li>")
		}
		buf.WriteString("</ul>")
		return buf.Bytes()
	default:
		return util.EscapeHTML(fmt.Append(nil, v))
	}

}
