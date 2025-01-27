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
func (q *UserQueue) ForEachWithRetry(f func(*User)) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Create a slice to keep track of already processed users
	processed := make(map[*User]struct{})
	var retryQueue []*User

	for _, user := range q.users {
		if _, exists := processed[user]; !exists {
			// Execute the function and check if it needs to retry
			if f(user) {
				// If the function returns true, add the user to the retry queue
				retryQueue = append(retryQueue, user)
			}
			processed[user] = struct{}{}
		}
	}

	// Retry processing the users that need to be retried
	for _, user := range retryQueue {
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
