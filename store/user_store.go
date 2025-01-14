package store

import "sync"

type User struct {
	ChatId       int64
	IsConnecting bool
	IsConnected  bool
	Partner      int64
}

type UserStore struct {
	Mu    sync.Mutex
	Users map[int64]*User
}

func (u *UserStore) AddUser(user *User) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	u.Users[user.ChatId] = user
}

func (u *UserStore) RemoveUser(chatId int64) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	delete(u.Users, chatId)
}

func (u *UserStore) GetUser(chatId int64) (*User, bool) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	user, exists := u.Users[chatId]
	return user, exists
}

func (u *UserStore) FindMatch(excludeChatId int64) (*User, bool) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	for _, user := range u.Users {
		if user.IsConnecting && user.ChatId != excludeChatId {
			return user, true
		}
	}
	return nil, false
}
