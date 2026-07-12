goldmark-meta
=========================
[![GoDev][godev-image]][godev-url]

[godev-image]: https://pkg.go.dev/badge/github.com/yuin/goldmark-meta/v2
[godev-url]: https://pkg.go.dev/github.com/yuin/goldmark-meta/v2


goldmark-meta is an extension for the [goldmark](http://github.com/yuin/goldmark) 
that allows you to define document metadata in YAML format.

Usage
--------------------
### Compatiblity
goldmark-meta/v2 is compatible with goldmark/v2.

### Installation

```
go get github.com/yuin/goldmark-meta/v2
```

### Markdown syntax

YAML metadata block is a leaf block that can not have any markdown element
as a child.

YAML metadata must start with a **YAML metadata separator**.
This separator must be at first line of the document.

A **YAML metadata separator** is a line that only `---`.

YAML metadata must end with a **YAML metadata separator**.

You can define objects as a 1st level item. At deeper level, you can define 
any kind of YAML element.

Example:

```
---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---

# Heading 1
```


### Access the metadata

```go
import (
    "fmt"
    "github.com/yuin/goldmark/v2/parser"
    "github.com/yuin/goldmark/v2/ast"
    "github.com/yuin/goldmark-meta/v2"
)

func main() {
	source := []byte(`---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---
`)

	document := parser.New(
		parser.WithExtensions(
			meta.Parser,
		),
	).ParseBytes(source)
	metaData := document.(*ast.Document).Metadata()
	title := metaData["Title"]
	fmt.Print(title)
}
```

### Render the metadata as a table

Use `meta.WithTable()` on the HTML renderer extension:

```go
import (
    "bytes"
    "fmt"
    "github.com/yuin/goldmark/v2/parser"
    "github.com/yuin/goldmark/v2/renderer/html"
    "github.com/yuin/goldmark-meta/v2"
)

func main() {
    source := []byte(`---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---

# Hello goldmark-meta
`)
    p := parser.New(
        parser.WithExtensions(
            meta.Parser,
        ),
    )
    r := html.New(
        html.WithExtensions(
            meta.NewHTMLRenderer(meta.WithTable()),
        ),
    )
    document := p.ParseBytes(source)

    var buf bytes.Buffer
    if err := r.Render(&buf, source, document); err != nil {
        panic(err)
    }
    fmt.Print(buf.String())
}
```


License
--------------------
MIT

Author
--------------------
Yusuke Inuzuka
