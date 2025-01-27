package test

import (
	"telepushx/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserQueue(t *testing.T) {
	queue := &model.UserQueue{}

	// Test PushFront
	user1 := &model.User{Id: 1, ChatId: "123", Name: "User1"}
	user2 := &model.User{Id: 2, ChatId: "456", Name: "User2"}
	queue.PushFront(user1)
	queue.PushFront(user2)

	// Test ForEach
	var users []*model.User
	queue.ForEach(func(u *model.User) {
		users = append(users, u)
	})
	assert.Equal(t, 2, len(users))
	assert.Equal(t, user2, users[0]) // user2 should be at the front
	assert.Equal(t, user1, users[1]) // user1 should be at the back

	// Test Pop
	poppedUser := queue.Pop()
	assert.Equal(t, user2, poppedUser) // user2 should be popped first
	assert.Equal(t, 1, queue.Len())    // only user1 should remain

	// Test PushBatch
	user3 := &model.User{Id: 3, ChatId: "789", Name: "User3"}
	user4 := &model.User{Id: 4, ChatId: "101", Name: "User4"}
	queue.PushBatch([]*model.User{user3, user4})

	// Test Pop again
	poppedUser = queue.Pop()
	assert.Equal(t, user1, poppedUser) // user1 should be popped next
	assert.Equal(t, 2, queue.Len())    // user3 and user4 should remain

	// Test ForEach again
	users = nil
	queue.ForEach(func(u *model.User) {
		users = append(users, u)
	})
	assert.Equal(t, 2, len(users))
	assert.Equal(t, user3, users[0]) // user3 should be first
	assert.Equal(t, user4, users[1]) // user4 should be second

	// Test Pop until empty
	queue.Pop()                // pop user3
	queue.Pop()                // pop user4
	assert.Nil(t, queue.Pop()) // should return nil since the queue is empty

}
