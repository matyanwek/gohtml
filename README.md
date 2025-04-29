# GoHtml

A simple HTML parser for Go applications focused on convenience.

## Installation

Vendor the package:

```sh
$ mkdir vendor
$ git clone https://github.com/matyanwek/gohtml vendor/gohtml
```

or use `go get`:

```sh
$ go get github.com/matyanwek/gohtml
```

## Usage

```go
package main

import (
	"fmt"
	"os"

	"main/gohtml"
)

func extractHtml(data []byte) {
	// use the gohtml.Parse function to turn []byte into a node tree
	docNode, err, warnings := gohtml.Parse(data)
	if err != nil {
		// parse error; parsing was terminated upon reaching this error
		// error messages include line and column numbers from the data
		panic(err)
	}

	// parse warnings; invalid HTML that the parser was able to resolve
	// (e.g. invalid entities)
	// these also include line and column numbers
	fmt.Printf("%d parse warnings\n", len(warnings))
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "[WARNING]: %s\n", warning)
	}

	// find a tag by name and use it immediately
	fmt.Println("title: %q\n", docNode.Find("title").Text())

	// gohtml.Node methods never return nil
	invalidNode := docNode.Find("not_a_tag_name").Text()
	fmt.Println("this will successfully print empty: %q\n", invalidNode)

	// store tag names with their attributes in an gohtml.Tag var
	contentTag := gohtml.Tag{
		Name:  "div",
		Attrs: map[string]string{"class": "content"},
	}

	// find a tag by gohtml.Tag
	contentNode := docNode.FindTag(contentTag)
	// test if found by checking node.Kind
	if node.Kind == gohtml.InvalidNode {
		panic("content node not found")
	}

	// find multiple tags, either by name or with an gohtml.Tag
	postTag := gohtml.Tag{
		Name:  "div",
		Attrs: map[string]string{"class": "post"},
	}
	postNodes := contentNode.FindTagAll(postTag, true)

	for _, postNode := range postNodes {
		// tag attributes are map[string]string
		fmt.Printf("post #%s", postNode.Attrs["id"])

		// recursively retrieve all text with gohtml.Node.Text()
		fmt.Println("post content:")
		fmt.Println(postNode.Text())
	}
}
```

## TODOs

- Tests
- Additional methods for `gohtml.Node`:
	- text excluding specified tags
	- formatted text, including dedentation and expanded `<br>` tags.
	- as HTML
- Additional validation during parsing
	- unexpected `<` characters in tags
	- mismatched quotes in tag attributes
- Additional parse warnings
	- text after closing last tag

## Related Projects

- [Golang non-core HTML package](https://pkg.go.dev/golang.org/x/net/html)
