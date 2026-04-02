package ccache

type List[T any] struct {
	Head *Item[T]
	Tail *Item[T]
}

func NewList[T any]() *List[T] {
	return &List[T]{}
}

func (l *List[T]) Remove(item *Item[T]) {
	next := item.next
	prev := item.prev

	if next == nil {
		l.Tail = prev
	} else {
		next.prev = prev
	}

	if prev == nil {
		l.Head = next
	} else {
		prev.next = next
	}
	item.next = nil
	item.prev = nil
	item.inList = false
}

func (l *List[T]) MoveToFront(item *Item[T]) {
	l.Remove(item)
	l.Insert(item)
}

func (l *List[T]) Insert(item *Item[T]) {
	head := l.Head
	l.Head = item
	item.inList = true
	if head == nil {
		l.Tail = item
		return
	}
	item.next = head
	head.prev = item
}
