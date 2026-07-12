package meta

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

func parseAndRender(t *testing.T, source string, rendererExts ...html.Extension) (*ast.Document, string) {
	t.Helper()

	sourceBytes := []byte(source)
	p := parser.New(parser.WithExtensions(Parser))
	document := p.Parse(sourceBytes).OwnerDocument()
	r := html.New(html.WithExtensions(append([]html.Extension{NewHTMLRenderer()}, rendererExts...)...))

	var buf bytes.Buffer
	if err := r.Render(&buf, sourceBytes, document); err != nil {
		t.Fatal(err)
	}

	return document, buf.String()
}

func TestMeta(t *testing.T) {
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---

# Hello goldmark-meta
`

	document, output := parseAndRender(t, source)
	title := document.Metadata()["Title"]
	s, ok := title.(string)
	if !ok {
		t.Error("Title not found in meta data or is not a string")
	}
	if s != "goldmark-meta" {
		t.Errorf("Title must be %s, but got %v", "goldmark-meta", s)
	}
	if output != "<h1>Hello goldmark-meta</h1>\n" {
		t.Errorf("should render '<h1>Hello goldmark-meta</h1>', but '%s'", output)
	}
	tags, ok := document.Metadata()["Tags"].([]any)
	if !ok {
		t.Error("Tags not found in meta data or is not a slice")
	}
	if len(tags) != 2 {
		t.Error("Tags must be a slice that has 2 elements")
	}
	if tags[0] != "markdown" {
		t.Errorf("Tag#1 must be 'markdown', but got %s", tags[0])
	}
	if tags[1] != "goldmark" {
		t.Errorf("Tag#2 must be 'goldmark', but got %s", tags[1])
	}
}

func TestMetaTable(t *testing.T) {
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---

# Hello goldmark-meta
`

	_, output := parseAndRender(t, source, NewHTMLRenderer(WithTable()))
	if output != `<table>
<thead>
<tr>
<th>Title</th>
<th>Summary</th>
<th>Tags</th>
</tr>
</thead>
<tbody>
<tr>
<td>goldmark-meta</td>
<td>Add YAML metadata to the document</td>
<td><ul><li>markdown</li><li>goldmark</li></ul></td>
</tr>
</tbody>
</table>
<h1>Hello goldmark-meta</h1>
` {
		t.Error("invalid table output")
	}
}

func TestMetaError(t *testing.T) {
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
  - : {
  }
    - markdown
    - goldmark
---

# Hello goldmark-meta
`

	document, output := parseAndRender(t, source)
	if output != `<!-- go-yaml load error in parser (while parsing a block mapping) at L4.C5: did not find expected key -->
<h1>Hello goldmark-meta</h1>
` {
		t.Error("invalid error output")
		println(output)
	}

	v, ok := document.Metadata()["Title"]
	if ok {
		t.Error("Title should not be found in meta data when there are errors")
	}
	if v != nil {
		t.Error("data should be nil when there are errors")
	}
}

func TestMetaTableWithBlankline(t *testing.T) {
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document

# comments
Tags:
    - markdown
    - goldmark
---

# Hello goldmark-meta
`

	_, output := parseAndRender(t, source, NewHTMLRenderer(WithTable()))
	if output != `<table>
<thead>
<tr>
<th>Title</th>
<th>Summary</th>
<th>Tags</th>
</tr>
</thead>
<tbody>
<tr>
<td>goldmark-meta</td>
<td>Add YAML metadata to the document</td>
<td><ul><li>markdown</li><li>goldmark</li></ul></td>
</tr>
</tbody>
</table>
<h1>Hello goldmark-meta</h1>
` {
		t.Error("invalid table output")
	}
}

func TestMetaStoreInDocument(t *testing.T) {
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---
`

	sourceBytes := []byte(source)
	document := parser.New(
		parser.WithExtensions(
			Parser,
		),
	).Parse(sourceBytes).OwnerDocument()
	metaData := document.Metadata()
	title := metaData["Title"]
	s, ok := title.(string)
	if !ok {
		t.Error("Title not found in meta data or is not a string")
	}
	if s != "goldmark-meta" {
		t.Errorf("Title must be %s, but got %v", "goldmark-meta", s)
	}
	tags, ok := metaData["Tags"].([]any)
	if !ok {
		t.Error("Tags not found in meta data or is not a slice")
	}
	if len(tags) != 2 {
		t.Error("Tags must be a slice that has 2 elements")
	}
	if tags[0] != "markdown" {
		t.Errorf("Tag#1 must be 'markdown', but got %s", tags[0])
	}
	if tags[1] != "goldmark" {
		t.Errorf("Tag#2 must be 'goldmark', but got %s", tags[1])
	}
}

func TestTableLayout(t *testing.T) {
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---

# Hello goldmark-meta
`

	_, output := parseAndRender(t, source, NewHTMLRenderer(WithTable(WithLayout(TableLayoutColumns))))
	if output != `<table>
<thead>
<tr>
<th>Title</th>
<th>Summary</th>
<th>Tags</th>
</tr>
</thead>
<tbody>
<tr>
<td>goldmark-meta</td>
<td>Add YAML metadata to the document</td>
<td><ul><li>markdown</li><li>goldmark</li></ul></td>
</tr>
</tbody>
</table>
<h1>Hello goldmark-meta</h1>
` {
		t.Error("invalid table output")
	}

	_, output = parseAndRender(t, source, NewHTMLRenderer(WithTable(WithLayout(TableLayoutRows))))
	if output != `<table>
<tbody>
<tr>
<td>Title</td>
<td>goldmark-meta</td>
</tr>
<tr>
<td>Summary</td>
<td>Add YAML metadata to the document</td>
</tr>
<tr>
<td>Tags</td>
<td><ul><li>markdown</li><li>goldmark</li></ul></td>
</tr>
</tbody>
</table>
<h1>Hello goldmark-meta</h1>
` {
		t.Error("invalid table output")
	}

}
