// Package gohtml parses HTML data into a tree of nodes.
package gohtml

// Parse HTML.  Returns the node representing the entire document, a fatal
// parse error (if encountered), and a slice of warnings.  The returned node is
// never nil, regardless of the value of err.
func Parse(data []byte) (node *Node, err error, warns []error) {
	tokens, err, warns := lex(data)
	if err != nil {
		return node, err, warns
	}

	node, err, parseWarns := parse(tokens)
	warns = append(warns, parseWarns...)
	if err != nil {
		return node, err, warns
	}

	return node, nil, warns
}
