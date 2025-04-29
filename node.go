package gohtml

import (
	"fmt"
	"strings"
)

// Cursor location in a file.
type Location struct {
	Line int // 1-indexed line number
	Col  int // 1-indexed column number
	Pos  int // 0-indexed byte offset
}

// Error message-friendly string representation.
func (loc Location) String() string {
	return fmt.Sprintf("%d:%d", loc.Line, loc.Col)
}

// Represents an HTML tag.
type Tag struct {
	Name  string            // Tag name
	Attrs map[string]string // Tag attributes
}

// Kind of node (e.g. element node, text node, etc.).
type NodeKind int

const (
	InvalidNode     NodeKind = iota // Signifies an erroneous or zero node
	DocumentNode                    // Top-level node for a whole HTML document
	ElementNode                     // Element
	TextNode                        // Content between start and end tags
	CommentNode                     // Comment
	DeclarationNode                 // Declaration (e.g. <!DOCTYPE html>)
)

// Error message-friendly string representation.
func (kind NodeKind) String() string {
	switch kind {
	case DocumentNode:
		return "DocumentNode"
	case TextNode:
		return "TextNode"
	case ElementNode:
		return "ElementNode"
	case CommentNode:
		return "CommentNode"
	case DeclarationNode:
		return "DeclarationNode"
	default:
		return "InvalidNode"
	}
}

// Node in an HTML tree.  Nodes represent parts of an HTML document according
// to their NodeKind.
type Node struct {
	// Kind.
	Kind NodeKind

	// Text for TextNode, CommentNode, and DeclarationNode, and tag name for
	// ElementNode. Empty otherwise.
	Content string

	// Tag attributes. Only applicable to ElementNode.
	Attrs map[string]string

	// Child nodes. Only applicable to ElementNode and DocumentNode.
	Children []*Node

	// Location in the original document where the node began.
	Loc Location
}

// Make a new empty node.
func EmptyNode() (node *Node) {
	return &Node{
		Kind:     InvalidNode,
		Content:  "",
		Attrs:    make(map[string]string),
		Children: make([]*Node, 0),
		Loc:      Location{Line: 0, Col: 0, Pos: -1},
	}
}

// Match an HTML element with a Tag.  Only applicable to ElementNode; returns
// false for any other NodeKind.
//
// Checks whether the element has the same name as the tag and whether the
// element has at least every attribute that the tag has; i.e. returns
// true if tag.Name == node.Content and node.Attrs[attr] == tag.Attrs[attr]
// for every attr in tag.Attrs.
func (node *Node) MatchTag(tag Tag) bool {
	if node.Kind != ElementNode || node.Content != tag.Name {
		return false
	}

	for key, tagVal := range tag.Attrs {
		nodeVal, ok := node.Attrs[key]
		if !ok || nodeVal != tagVal {
			return false
		}
	}

	return true
}

// Find the first descendant node that has the tag name tagName; i.e. returns
// the first ElementNode with node.Content == tagName.  Returns an empty,
// non-nil *Node of InvalidNode kind if no matching descendant was
// found.
func (node *Node) Find(tagName string) *Node {
	stk := make(stack[*Node], 0, 16)
	stk.Push(node)

	for node, ok := stk.Pop(); ok; node, ok = stk.Pop() {
		if node.Kind == ElementNode && node.Content == tagName {
			return node
		}

		// reverse iteration so that first child is pushed last
		for i := len(node.Children) - 1; i >= 0; i-- {
			stk.Push(node.Children[i])
		}
	}

	return EmptyNode()
}

// Find the first descendant node that matches tag; i.e. returns the first
// ElementNode with node.Content == tag.Name and has
// node.Attrs[attr] == tag.Attrs[attr] for every attr in tag.Attrs.  Returns
// an empty, non-nil *Node of InvalidNode kind if no matching descendant was
// found.
func (node *Node) FindTag(tag Tag) *Node {
	stk := make(stack[*Node], 0, 16)
	stk.Push(node)

	for node, ok := stk.Pop(); ok; node, ok = stk.Pop() {
		if node.Kind == ElementNode && node.MatchTag(tag) {
			return node
		}

		// reverse iteration so that first child is pushed last
		for i := len(node.Children) - 1; i >= 0; i-- {
			stk.Push(node.Children[i])
		}
	}

	return EmptyNode()
}

// Find all descendant nodes that have the tag name tagName; i.e. returns all
// ElementNodes with node.Content == tagName.  Returns an empty, non-nil slice
// of *Node if no matching descendants were found.
//
// The prune flag determines whether to prune the search on descendant node
// matches, thus returning a flat slice of nodes where no result is the
// descendant of any other result.
func (node *Node) FindAll(tagName string, prune bool) []*Node {
	matches := make([]*Node, 0, 16)

	stk := make(stack[*Node], 0, 16)
	stk.Push(node)

	for node, ok := stk.Pop(); ok; node, ok = stk.Pop() {
		if node.Kind == ElementNode && node.Content == tagName {
			matches = append(matches, node)
			if prune {
				continue
			}
		}

		// reverse iteration so that first child is pushed last
		for i := len(node.Children) - 1; i >= 0; i-- {
			stk.Push(node.Children[i])
		}
	}

	return matches
}

// Find all descendant nodes that have the tag name tagName; i.e. returns all
// ElementNodes with node.Content == tagName.
//
// Pass prune == true to prune the search if the descendant node matches, thus
// returning a flat slice of nodes.
func (node *Node) FindTagAll(tag Tag, prune bool) []*Node {
	matches := make([]*Node, 0, 16)

	stk := make(stack[*Node], 0, 16)
	stk.Push(node)

	for node, ok := stk.Pop(); ok; node, ok = stk.Pop() {
		if node.Kind == ElementNode && node.MatchTag(tag) {
			matches = append(matches, node)
			if prune {
				continue
			}
		}

		// reverse iteration so that first child is pushed last
		for i := len(node.Children) - 1; i >= 0; i-- {
			stk.Push(node.Children[i])
		}
	}

	return matches
}

// Return the concatenated contents of all descendent TextNodes.
func (node *Node) Text() string {
	contents := make([]string, 0, len(node.Children))

	stk := make(stack[*Node], 0, 16)
	stk.Push(node)

	for node, ok := stk.Pop(); ok; node, ok = stk.Pop() {
		if node.Kind == TextNode {
			contents = append(contents, node.Content)
		}

		// reverse iteration so that first child is pushed last
		for i := len(node.Children) - 1; i >= 0; i-- {
			stk.Push(node.Children[i])
		}
	}

	return strings.Join(contents, "")
}

// TODO: func (node *Node) TextExclude(tags []Tag) string
// text that excludes tags (e.g. <script>)

// TODO: func (node *Node) TextFormatted() string
// expand text elements (e.g. <br>)

// TODO: func (node *Node) Dump() []byte
// recursively dump node contents as HTML
