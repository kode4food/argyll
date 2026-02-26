package util

type (
	// PathTree indexes values by hierarchical string paths
	PathTree[T any] struct {
		root *pathTreeNode[T]
	}

	pathTreeNode[T any] struct {
		value    T
		hasValue bool
		children map[string]*pathTreeNode[T]
	}
)

// NewPathTree creates a new hierarchical path index
func NewPathTree[T any]() *PathTree[T] {
	return &PathTree[T]{
		root: &pathTreeNode[T]{children: map[string]*pathTreeNode[T]{}},
	}
}

// Insert stores a value at the exact path
func (t *PathTree[T]) Insert(path []string, v T) {
	cur := t.root
	if len(path) == 0 {
		cur.value = v
		cur.hasValue = true
		return
	}
	for _, p := range path {
		next, ok := cur.children[p]
		if !ok {
			next = &pathTreeNode[T]{children: map[string]*pathTreeNode[T]{}}
			cur.children[p] = next
		}
		cur = next
	}
	cur.value = v
	cur.hasValue = true
}

// Remove clears the value at the exact path
func (t *PathTree[T]) Remove(path []string) {
	if len(path) == 0 {
		t.root.hasValue = false
		var zero T
		t.root.value = zero
		return
	}
	t.root.remove(path, 0)
}

// Detach removes a prefix subtree
func (t *PathTree[T]) Detach(prefix []string) {
	t.DetachWith(prefix, nil)
}

// DetachWith removes a prefix subtree and calls fn for each detached value
func (t *PathTree[T]) DetachWith(prefix []string, fn func(T)) {
	if len(prefix) == 0 {
		root := t.root
		t.root = &pathTreeNode[T]{children: map[string]*pathTreeNode[T]{}}
		if fn != nil {
			root.walkValues(fn)
		}
		return
	}
	if n := t.root.detach(prefix); n != nil && fn != nil {
		n.walkValues(fn)
	}
}

func (n *pathTreeNode[T]) remove(path []string, idx int) bool {
	if idx == len(path) {
		n.hasValue = false
		var zero T
		n.value = zero
		return len(n.children) == 0
	}
	next, ok := n.children[path[idx]]
	if !ok {
		return false
	}
	if next.remove(path, idx+1) {
		delete(n.children, path[idx])
	}
	return !n.hasValue && len(n.children) == 0
}

func (n *pathTreeNode[T]) detach(prefix []string) *pathTreeNode[T] {
	cur := n
	for _, p := range prefix[:len(prefix)-1] {
		next, ok := cur.children[p]
		if !ok {
			return nil
		}
		cur = next
	}
	last := prefix[len(prefix)-1]
	next, ok := cur.children[last]
	if !ok {
		return nil
	}
	delete(cur.children, last)
	return next
}

func (n *pathTreeNode[T]) walkValues(fn func(T)) {
	if n.hasValue {
		fn(n.value)
	}
	for _, child := range n.children {
		child.walkValues(fn)
	}
}
