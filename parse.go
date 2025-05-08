package gohtml

import (
	"bytes"
	"fmt"
	"strconv"
)

var unpairedTags = map[string]bool{
	"area":     true,
	"base":     true,
	"br":       true,
	"colgroup": true,
	"col":      true,
	"embed":    true,
	"hr":       true,
	"image":    true,
	"img":      true,
	"input":    true,
	"link":     true,
	"meta":     true,
	"param":    true,
	"source":   true,
	"track":    true,
	"wbr":      true,
}

var (
	equals = []byte("=")
	amp    = []byte("&")
)

func parseEntity(data []byte) (exp string, entityLen int, err error) {
	const longestEntity = 33      // &CounterClockwiseContourIntegral;
	const shortestEntity = 4      // &gt;
	const longestNoscEntity = 7   // &aacute
	const shortestNoscEntity = 33 // &gt

	// empty or insufficient data; i.e. data == "" || data == "&"
	if len(data) < 2 {
		err = fmt.Errorf("%w: no data", EntityErr)
		return string(data), len(data), err
	}

	entityLen = bytes.IndexByte(data, ';') + 1

	if entityLen == 2 {
		// empty entity, data[:2] == "&;"
		err = fmt.Errorf("%w: empty entity", EntityErr)

	} else if entityLen <= 0 || entityLen > longestEntity {
		// no semicolon, data == "&...
		// NOTE: entities without semicolon terminators are invalid, but are
		// explicitly named in the HTML spec and browsers tend to support them.
		// see: <https://html.spec.whatwg.org/multipage/named-characters.html#named-character-references>
		for entityLen = shortestNoscEntity; entityLen <= longestNoscEntity; entityLen++ {
			var ok bool
			exp, ok = entityMap[string(data[:entityLen])]
			if ok {
				err = fmt.Errorf("%w: no terminating semicolon", EntityErr)
				return
			}
		}
		err = fmt.Errorf("%w: no matching entity", EntityErr)
		entityLen = 1

	} else if data[1] != '#' {
		// ordinary entity, e.g. data == "&gt;"
		var ok bool
		exp, ok = entityMap[string(data[:entityLen])]
		if !ok {
			err = fmt.Errorf("%w: no matching entity", EntityErr)
		}

		// } else if len(data) <= 2 {
		// 	err = fmt.Errorf("%w: insufficient data for number entity", EntityErr)
	} else if data[2] != 'x' {
		// number entity, e.g. data == "&#35;"
		var codepoint int
		codepoint, err = strconv.Atoi(string(data[2 : entityLen-1]))
		if err == nil {
			exp = string(rune(codepoint))
		} else {
			err = fmt.Errorf("%w: %w", EntityErr, err)
		}

	} else if len(data) <= 3 {
		err = fmt.Errorf("%w: insufficient data for hex number entity", EntityErr)
	} else if data[3] == 'x' {
		// hex number entity, e.g. data == "&#x23;"
		var codepoint int64
		codepoint, err = strconv.ParseInt(string(data[3:entityLen-1]), 16, 0)
		if err == nil {
			exp = string(rune(codepoint))
		} else {
			err = fmt.Errorf("%w: %w", EntityErr, err)
		}
	}

	if exp == "" {
		exp = string(data[:entityLen])
	}
	return
}

// Expand entities in a data slice and return the expanded data and any entity
// parse errors as warnings. 'loc' is needed to report warning locations.
func expandEntitys(data []byte, loc Location) ([]byte, []error) {
	var warns []error

	buf := bytes.Buffer{}
	buf.Grow(len(data))

	for {
		loc.Pos = 0
		loc, ok := stepUntil(data, amp, loc)
		buf.Write(data[:loc.Pos])
		if !ok {
			break
		}

		data = data[loc.Pos:]
		exp, entityLen, warn := parseEntity(data)
		if warn != nil {
			warn = fmt.Errorf("%s: %w", loc, warn)
			warns = append(warns, warn)
		}
		buf.WriteString(exp)
		data = data[entityLen:]
	}

	return buf.Bytes(), warns
}

func parseComment(tok token) (*Node, error) {
	node := &Node{Kind: CommentNode, Loc: tok.Loc, Content: string(tok.Data)}
	return node, nil
}

func parseDeclaration(tok token) (*Node, error) {
	node := &Node{Kind: DeclarationNode, Loc: tok.Loc}

	data := bytes.TrimSpace(tok.Data)
	if len(data) == 0 {
		err := fmt.Errorf("%s: error parsing declaration: %w", tok.Loc, EmptyContentErr)
		return node, err
	}

	node.Content = string(data)
	return node, nil
}

func parseText(tok token) (node *Node, err error, warns []error) {
	node = &Node{Kind: TextNode, Loc: tok.Loc}

	// NOTE: don't discard leading and trailing spaces that may have been
	// intentionally added to the content
	// data := bytes.TrimSpace(tok.Data)
	if len(tok.Data) == 0 {
		err = fmt.Errorf("%s: error parsing text: %w", tok.Loc, EmptyContentErr)
		return node, err, warns
	}

	content, warns := expandEntitys(tok.Data, tok.Loc)
	node.Content = string(content)
	return node, nil, warns
}

func parseCloseTag(tok token) (*Node, error) {
	node := &Node{Kind: InvalidNode, Loc: tok.Loc}

	data := bytes.TrimSpace(tok.Data)
	if len(data) == 0 {
		err := fmt.Errorf("%s: error parsing closing tag: %w", tok.Loc, EmptyContentErr)
		return node, err
	}

	node.Content = string(data)
	return node, nil
}

func isSpace(c byte) bool {
	return c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}

// Find the end of a tag field, which is either a space or a quote.
func findAttrEnd(data []byte) int {
	i := 0
	for {
		i++
		if i >= len(data) || isSpace(data[i]) {
			return i
		} else if data[i] == '=' {
			break
		}
	}

	i += 1
	if i >= len(data) || isSpace(data[i]) {
		return i
	}

	// TODO: validate matching quotes; may need to return an error
	if data[i] == '"' || data[i] == '\'' {
		quote := data[i]
		i++
		for i < len(data) && data[i] != quote {
			i++
		}
		i++
	} else {
		for i < len(data) && !isSpace(data[i]) {
			i++
		}
	}

	return i
}

// Split tags by "fields", such that all fields are separated by spaces that
// are not escaped or quoted (e.g. <tag attr1=val attr2="val with space">
// => ['tag', 'attr1=val', 'attr2="val with space"']).
// Fields are enclosed in text tokens to report warning locations more easily.
func splitAttrs(data []byte, loc Location) []token {
	fields := make([]token, 0, 16)

	loc.Pos = 0
	for {
		for loc.Pos < len(data) && isSpace(data[loc.Pos]) {
			if data[loc.Pos] == '\n' {
				loc.Line++
				loc.Col = 0
			}
			loc.Col++
			loc.Pos++
		}
		if loc.Pos >= len(data) {
			break
		}

		attrLen := findAttrEnd(data[loc.Pos:])
		field := token{Kind: textToken, Loc: loc, Data: data[loc.Pos : loc.Pos+attrLen]}
		fields = append(fields, field)
		loc.Pos += attrLen
		loc.Col += attrLen
	}

	return fields
}

func parseAttr(field token) (key string, val string, warns []error) {
	keyData, valData, found := bytes.Cut(field.Data, equals)
	if found {
		// attribute with a value (e.g. key="val")

		// advance field location to the value
		field.Loc.Pos = len(keyData) + len(equals)
		field.Loc.Col += len(keyData) + len(equals)

		valData = bytes.Trim(valData, "\"'")
		// TODO: should entities not be expanded in href attributes?
		valData, warns = expandEntitys(valData, field.Loc)
	}
	return string(keyData), string(valData), warns
}

func parseOpenTag(tok token) (node *Node, err error, warns []error) {
	node = &Node{Kind: ElementNode, Loc: tok.Loc}

	fields := splitAttrs(tok.Data, tok.Loc)
	if len(fields) == 0 {
		err = fmt.Errorf("%s: error parsing opening tag: %w", tok.Loc, EmptyContentErr)
		return
	}

	node.Children = make([]*Node, 0, 16)

	node.Content = string(fields[0].Data)

	node.Attrs = make(map[string]string, len(fields)-1)
	for _, field := range fields[1:] {
		key, val, fieldWarns := parseAttr(field)
		node.Attrs[key] = val
		warns = append(warns, fieldWarns...)
	}

	return
}

func parse(tokens []token) (docNode *Node, err error, warns []error) {
	docNode = &Node{Kind: DocumentNode, Children: make([]*Node, 0, 4)}
	tags := make(stack[*Node], 0, 16)
	tags.Push(docNode)

	i := 0

loop:
	for ; i < len(tokens); i++ {
		tok := tokens[i]

		parent, ok := tags.Peek()
		if !ok {
			err = fmt.Errorf("%s: error parsing document: %w", tok.Loc, EmptyTagStackErr)
			return
		}

		var node *Node
		var tokWarns []error

		switch tok.Kind {
		case eofToken:
			break loop
		case commentToken:
			node, err = parseComment(tok)
		case declarationToken:
			node, err = parseDeclaration(tok)
		case verbatimToken:
			node = &Node{Kind: TextNode, Loc: tok.Loc, Content: string(tok.Data)}
		case textToken:
			node, err, tokWarns = parseText(tok)
		case tagSelfcloseToken:
			node, err, tokWarns = parseOpenTag(tok)
		case tagOpenToken:
			node, err, tokWarns = parseOpenTag(tok)
		case tagCloseToken:
			node, err = parseCloseTag(tok)
			if err != nil {
				break
			} else if node.Content == parent.Content {
				tags.Pop()
				continue loop
			} else {
				warn := fmt.Errorf("%s: error parsing closing tag: %w: expected %q but got %q", node.Loc, TagMismatchErr, parent.Content, node.Content)
				tokWarns = append(tokWarns, warn)
			}
		default:
			err = fmt.Errorf("%s: error parsing document: %w: %s", tok.Loc, TokenErr, tok.Kind)
		}

		if err != nil {
			return
		}

		parent.Children = append(parent.Children, node)

		if node.Kind == ElementNode && !unpairedTags[node.Content] {
			tags.Push(node)
		}

		warns = append(warns, tokWarns...)
	}

	if len(tags) > 1 {
		node, _ := tags.Peek()
		warn := fmt.Errorf("%s: error parsing document: %w: %q", node.Loc, UnclosedTagErr, node.Content)
		warns = append(warns, warn)
	} else if len(tags) < 1 {
		warn := fmt.Errorf("%s: error parsing document: %w", tokens[len(tokens)-1].Loc, EmptyTagStackErr)
		warns = append(warns, warn)
	}

	return
}
