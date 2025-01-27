package model

import (
	"sync"
)

type UserQueue struct {
	mu    sync.Mutex
	users []*User
}

func (q *UserQueue) PushFront(u *User) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.users = append([]*User{u}, q.users...)
}

func (q *UserQueue) Pop() *User {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.users) == 0 {
		return nil
	}

	u := q.users[0]
	q.users = q.users[1:]
	return u
}

func (q *UserQueue) PushBatch(users []*User) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 将用户列表追加到队列
	q.users = append(q.users, users...)
}
func (q *UserQueue) ForEachSkipFront(f func(*User), skipFrontCount int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Calculate the number of users to skip from the front
	startIndex := skipFrontCount
	if startIndex >= len(q.users) {
		return // No users to iterate over
	}

	for _, user := range q.users[startIndex:] {
		f(user)
	}
}

// 遍历队列中的所有用户
func (q *UserQueue) ForEach(f func(*User)) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, user := range q.users {
		f(user)
	}
}

// 获取队列长度
func (q *UserQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.users)
}
