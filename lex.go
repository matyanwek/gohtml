package gohtml

import (
	"bytes"
	"fmt"
)

type tokenKind int

const (
	invalidToken tokenKind = iota
	textToken
	verbatimToken
	tagOpenToken
	tagCloseToken
	tagSelfcloseToken
	commentToken
	declarationToken
	eofToken
)

func (kind tokenKind) String() string {
	switch kind {
	case textToken:
		return "textToken"
	case verbatimToken:
		return "verbatimToken"
	case tagOpenToken:
		return "tagOpenToken"
	case tagCloseToken:
		return "tagCloseToken"
	case tagSelfcloseToken:
		return "tagSelfcloseToken"
	case commentToken:
		return "commentToken"
	case declarationToken:
		return "declarationToken"
	case eofToken:
		return "eofToken"
	default:
		return "invalidToken"
	}
}

type token struct {
	Kind tokenKind
	Loc  Location
	Data []byte
}

func (tok token) String() string {
	return fmt.Sprintf("%s: %s %q", tok.Loc, tok.Kind, tok.Data)
}

func stepUntil(data []byte, prefix []byte, loc Location) (Location, bool) {
	for loc.Pos < len(data) && !bytes.HasPrefix(data[loc.Pos:], prefix) {
		if data[loc.Pos] == '\n' {
			loc.Line++
			loc.Col = 1
		} else {
			loc.Col++
		}
		loc.Pos++
	}

	return loc, loc.Pos < len(data)
}

var (
	commentStart     = []byte("<!--")
	commentEnd       = []byte("-->")
	declarationStart = []byte("<!")
	closeTagStart    = []byte("</")
	tagStart         = []byte("<")
	tagEnd           = []byte(">")
	tagSelfcloseEnd  = []byte("/>")
)

func lexComment(data []byte, loc Location) (token, Location, error) {
	tok := token{Loc: loc}

	loc.Pos += len(commentStart)
	newLoc, ok := stepUntil(data, commentEnd, loc)
	if !ok {
		err := fmt.Errorf("%s: error lexing comment: %w", loc, EofErr)
		return tok, newLoc, err
	}
	tok.Kind = commentToken

	tok.Data = data[loc.Pos:newLoc.Pos]
	newLoc.Pos += len(commentEnd)

	return tok, newLoc, nil
}

func lexDeclaration(data []byte, loc Location) (token, Location, error) {
	tok := token{Loc: loc}

	loc.Pos += len(declarationStart)
	newLoc, ok := stepUntil(data, tagEnd, loc)
	if !ok {
		err := fmt.Errorf("%s: error lexing declaration: %w", loc, EofErr)
		return tok, newLoc, err
	}
	tok.Kind = declarationToken

	tok.Data = data[loc.Pos:newLoc.Pos]
	newLoc.Pos += 1

	return tok, newLoc, nil
}

func lexTagClose(data []byte, loc Location) (token, Location, error) {
	tok := token{Loc: loc}

	loc.Pos += len(closeTagStart)
	newLoc, ok := stepUntil(data, tagEnd, loc)
	if !ok {
		err := fmt.Errorf("%s: error lexing closing tag: %w", loc, EofErr)
		return tok, newLoc, err
	}
	tok.Kind = tagCloseToken

	tok.Data = data[loc.Pos:newLoc.Pos]
	newLoc.Pos += 1

	return tok, newLoc, nil
}

func lexTagOpen(data []byte, loc Location) (token, Location, error) {
	tok := token{Loc: loc}

	loc.Pos += 1
	newLoc, ok := stepUntil(data, tagEnd, loc)
	if !ok {
		err := fmt.Errorf("%s: error lexing opening tag: %w", loc, EofErr)
		return tok, newLoc, err
	}

	if data[newLoc.Pos-1] == '/' {
		tok.Kind = tagSelfcloseToken
		tok.Data = data[loc.Pos : newLoc.Pos-1]
	} else {
		tok.Kind = tagOpenToken
		tok.Data = data[loc.Pos:newLoc.Pos]
	}

	newLoc.Pos += 1

	return tok, newLoc, nil
}

func lexText(data []byte, loc Location) (tok token, newLoc Location, err error, warn error) {
	tok = token{Loc: loc}
	newLoc, ok := stepUntil(data, tagStart, loc)
	if !ok {
		if len(bytes.TrimSpace(data[loc.Pos:newLoc.Pos])) == 0 {
			// NOTE: trailing spaces after the closing </html> tag, likely
			// trailing newlines; warn and ignore
			warn = fmt.Errorf("%s: error lexing text: %w", loc, EofErr)
			return
		} else {
			// data after closing </html>
			// TODO: should this be a warning?
			// it's recoverable by simply ignoring it
			err = fmt.Errorf("%s: error lexing text: %w", loc, EofErr)
			return
		}
	}
	tok.Kind = textToken
	tok.Data = data[loc.Pos:newLoc.Pos]
	return
}

func lexVerbatim(data []byte, loc Location, tagName string) (tok token, newLoc Location, err error, warn error) {
	tok = token{Loc: loc}

	closeTag := []byte("</" + tagName + ">")
	newLoc, ok := stepUntil(data, closeTag, loc)
	if !ok {
		if len(bytes.TrimSpace(data[loc.Pos:newLoc.Pos])) == 0 {
			// NOTE: trailing spaces after the closing </html> tag, likely
			// trailing newlines; warn and ignore
			warn = fmt.Errorf("%s: error lexing text: %w", loc, EofErr)
			return
		} else {
			// data after closing </html>
			// TODO: should this be a warning?
			// it's recoverable by simply ignoring it
			err = fmt.Errorf("%s: error lexing text: %w", loc, EofErr)
			return
		}
	}

	tok.Data = data[loc.Pos:newLoc.Pos]
	tok.Kind = verbatimToken
	return
}

// rune version of isSpace
func isSpaceR(r rune) bool {
	return r == '\t' || r == '\n' || r == '\f' || r == '\r' || r == ' '
}

func extractTagName(tok token) string {
	if tok.Kind != tagOpenToken {
		return ""
	}

	data := tok.Data
	i := bytes.IndexFunc(data, isSpaceR)
	if i >= 0 {
		data = data[:i]
	}

	return string(bytes.ToLower(data))
}

var verbatimTags = map[string]bool{
	"script": true,
	"style":  true,
	"pre":    true,
}

func inVerbatim(tokens []token) bool {
	if len(tokens) == 0 {
		return false
	}
	tok := tokens[len(tokens)-1]
	return tok.Kind == tagOpenToken && verbatimTags[extractTagName(tok)]
}

func lex(data []byte) (tokens []token, err error, warns []error) {
	if len(data) == 0 {
		err = EmptyInputErr
		return
	}

	tokens = make([]token, 0, len(data)/5)
	loc := Location{Line: 1, Col: 1, Pos: 0}

	for loc.Pos < len(data) {
		var tok token
		var warn error

		switch data[loc.Pos] {
		case '<':
			// TODO: warning if finding a '<' in a tag token
			// should be invalid HTML, but it's recoverable by treating the '<'
			// as text
			if bytes.HasPrefix(data[loc.Pos:], commentStart) {
				tok, loc, err = lexComment(data, loc)
			} else if bytes.HasPrefix(data[loc.Pos:], declarationStart) {
				tok, loc, err = lexDeclaration(data, loc)
			} else if bytes.HasPrefix(data[loc.Pos:], closeTagStart) {
				tok, loc, err = lexTagClose(data, loc)
			} else {
				tok, loc, err = lexTagOpen(data, loc)
			}
		default:
			if inVerbatim(tokens) {
				tagName := extractTagName(tokens[len(tokens)-1])
				tok, loc, err, warn = lexVerbatim(data, loc, tagName)
			} else {
				tok, loc, err, warn = lexText(data, loc)
				if warn != nil {
					warns = append(warns, warn)
				}
			}
		}

		if err != nil {
			return
		} else if tok.Kind != invalidToken {
			tokens = append(tokens, tok)
		}
	}

	eofToken := token{Kind: eofToken, Loc: loc}
	tokens = append(tokens, eofToken)
	return
}
