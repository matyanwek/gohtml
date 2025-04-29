package gohtml

type stack[T any] []T

func (stk stack[T]) Len() int {
	return len(stk)
}

func (stk *stack[T]) Push(val T) {
	*stk = append(*stk, val)
}

func (stk *stack[T]) Pop() (T, bool) {
	if len(*stk) == 0 {
		var val T
		return val, false
	}

	val := (*stk)[len(*stk)-1]
	*stk = (*stk)[:len(*stk)-1]
	return val, true
}

func (stk stack[T]) Peek() (T, bool) {
	if len(stk) == 0 {
		var val T
		return val, false
	} else {
		return stk[len(stk)-1], true
	}
}
