// package meta is a extension for the goldmark(http://github.com/yuin/goldmark).
//
// This extension parses YAML metadata blocks and store metadata to a
// parser.Context.
package meta

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"gopkg.in/yaml.v3"
)

type data struct {
	Map   map[string]interface{}
	Items *yaml.Node
	Error error
	Node  gast.Node
}

var contextKey = parser.NewContextKey()

// Get returns a YAML metadata.
func Get(pc parser.Context) map[string]interface{} {
	v := pc.Get(contextKey)
	if v == nil {
		return nil
	}
	d := v.(*data)
	return d.Map
}

// TryGet tries to get a YAML metadata.
// If there are YAML parsing errors, then nil and error are returned
func TryGet(pc parser.Context) (map[string]interface{}, error) {
	dtmp := pc.Get(contextKey)
	if dtmp == nil {
		return nil, nil
	}
	d := dtmp.(*data)
	if d.Error != nil {
		return nil, d.Error
	}
	return d.Map, nil
}

// GetItems returns a YAML metadata.
// GetItems preserves defined key order.
func GetItems(pc parser.Context) *yaml.Node {
	v := pc.Get(contextKey)
	if v == nil {
		return nil
	}
	d := v.(*data)
	return d.Items
}

// TryGetItems returns a YAML metadata.
// TryGetItems preserves defined key order.
// If there are YAML parsing errors, then nil and erro are returned.
func TryGetItems(pc parser.Context) (*yaml.Node, error) {
	dtmp := pc.Get(contextKey)
	if dtmp == nil {
		return nil, nil
	}
	d := dtmp.(*data)
	if d.Error != nil {
		return nil, d.Error
	}
	return d.Items, nil
}

type metaParser struct {
}

var defaultMetaParser = &metaParser{}

// NewParser returns a BlockParser that can parse YAML metadata blocks.
func NewParser() parser.BlockParser {
	return defaultMetaParser
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

func (b *metaParser) Trigger() []byte {
	return []byte{'-'}
}

func (b *metaParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	linenum, _ := reader.Position()
	if linenum != 0 {
		return nil, parser.NoChildren
	}
	line, _ := reader.PeekLine()
	if isSeparator(line) {
		return gast.NewTextBlock(), parser.NoChildren
	}
	return nil, parser.NoChildren
}

func (b *metaParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if isSeparator(line) && !util.IsBlank(line) {
		reader.Advance(segment.Len())
		return parser.Close
	}
	node.Lines().Append(segment)
	return parser.Continue | parser.NoChildren
}

func (b *metaParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(reader.Source()))
	}
	d := &data{}
	d.Node = node
	meta := map[string]interface{}{}
	if err := yaml.Unmarshal(buf.Bytes(), &meta); err != nil {
		d.Error = err
	} else {
		d.Map = meta
	}

	metaItems := &yaml.Node{}
	if err := yaml.Unmarshal(buf.Bytes(), metaItems); err != nil {
		d.Error = err
	} else {
		d.Items = metaItems
	}

	pc.Set(contextKey, d)

	if d.Error == nil {
		node.Parent().RemoveChild(node.Parent(), node)
	}
}

func (b *metaParser) CanInterruptParagraph() bool {
	return false
}

func (b *metaParser) CanAcceptIndentedLine() bool {
	return false
}

type astTransformer struct {
}

var defaultASTTransformer = &astTransformer{}

func (a *astTransformer) Transform(node *gast.Document, reader text.Reader, pc parser.Context) {
	dtmp := pc.Get(contextKey)
	if dtmp == nil {
		return
	}
	d := dtmp.(*data)
	if d.Error != nil {
		msg := gast.NewString([]byte(fmt.Sprintf("<!-- %s -->", d.Error)))
		msg.SetCode(true)
		d.Node.AppendChild(d.Node, msg)
		return
	}

	meta := GetItems(pc)
	if meta == nil {
		return
	}
	if meta.Kind == yaml.DocumentNode {
		meta = meta.Content[0]
	}
	if meta.Kind != yaml.MappingNode {
		// only mapping node is supported as root
		return
	}

	table := east.NewTable()
	alignments := []east.Alignment{}
	for i := 1; i == len(meta.Content)%2; i++ {
		alignments = append(alignments, east.AlignNone)
	}
	row := east.NewTableRow(alignments)
	valueNodes := make([]*yaml.Node, 0, len(meta.Content)/2)
	for i := 0; i < len(meta.Content); i = i + 2 {
		keyNode := meta.Content[i]
		valueNodes = append(valueNodes, meta.Content[i+1])

		cell := east.NewTableCell()
		cell.AppendChild(cell, gast.NewString([]byte(valueNodeToString(keyNode))))
		row.AppendChild(row, cell)
	}
	table.AppendChild(table, east.NewTableHeader(row))

	row = east.NewTableRow(alignments)
	for _, item := range valueNodes {
		cell := east.NewTableCell()
		cell.AppendChild(cell, gast.NewString([]byte(valueNodeToString(item))))
		row.AppendChild(row, cell)
	}
	table.AppendChild(table, row)
	node.InsertBefore(node, node.FirstChild(), table)
}

func valueNodeToString(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case yaml.SequenceNode:
		val := make([]string, len(node.Content))
		for i := range node.Content {
			val[i] = valueNodeToString(node.Content[i])
		}
		return fmt.Sprintf("%v", val)

	case yaml.MappingNode:
		if (len(node.Content) % 2) != 0 {
			return "<broken mapping node>"
		}
		val := make(map[string]string, len(node.Content)%2)
		for i := len(node.Content); i > 1; i = i - 2 {
			k := valueNodeToString(node.Content[i-2])
			val[fmt.Sprint(k)] = valueNodeToString(node.Content[i-1])
		}
		return fmt.Sprintf("%v", val)

	case yaml.ScalarNode:
		return node.Value
	}

	return fmt.Sprintf("<do not support yaml node kind '%v'>", node.Kind)
}

// Option is a functional option type for this extension.
type Option func(*meta)

// WithTable is a functional option that renders a YAML metadata as a table.
func WithTable() Option {
	return func(m *meta) {
		m.Table = true
	}
}

type meta struct {
	Table bool
}

// Meta is a extension for the goldmark.
var Meta = &meta{}

// New returns a new Meta extension.
func New(opts ...Option) goldmark.Extender {
	e := &meta{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *meta) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewParser(), 0),
		),
	)
	if e.Table {
		m.Parser().AddOptions(
			parser.WithASTTransformers(
				util.Prioritized(defaultASTTransformer, 0),
			),
		)
	}
}
