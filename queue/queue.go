package queue

import (
	"fmt"
	"sync"
)

type Node struct {
	ChatId int64
	Next   *Node
}

type Queue struct {
	Head *Node
	Tail *Node
	mu   sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		Head: nil,
		Tail: nil,
	}
}

func (q *Queue) Enqueue(chatId int64) {
	newNode := &Node{ChatId: chatId}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.Head == nil {
		q.Head = newNode
		q.Tail = newNode
	} else {
		q.Tail.Next = newNode
		q.Tail = newNode
	}
}

func (q *Queue) Dequeue() (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.Head == nil {
		return 0, fmt.Errorf("queue is empty")
	}

	dequeuedChatId := q.Head.ChatId
	q.Head = q.Head.Next
	if q.Head == nil {
		q.Tail = nil
	}
	return dequeuedChatId, nil
}

func (q *Queue) RemoveNode(chatId int64) error {

	if q.Head == nil {
		return fmt.Errorf("queue is empty")
	}

	if q.Head.ChatId == chatId {
		q.Head = q.Head.Next
		if q.Head == nil {
			q.Tail = nil
		}
		return nil
	}

	current := q.Head
	for current.Next != nil {
		if current.Next.ChatId == chatId {
			current.Next = current.Next.Next
			if current.Next == nil {
				q.Tail = current
			}
		}
		current = current.Next
	}

	// If we reach here, the node wasn't found
	return fmt.Errorf("chatId %d not found in the queue", chatId)
}
