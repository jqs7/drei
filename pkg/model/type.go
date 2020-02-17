package model

import "time"

type KV struct {
	K string
	V string
}

type MsgToDelete struct {
	ChatID int64
	MsgID  int
}

type CountdownMsg struct {
	ChatID int64
	UserID int
}

type MsgToRepeat struct {
	Text   string
	SendTo int
	Times  int
}

type Idiom struct {
	ID   int64
	Word string
}

type Blacklist struct {
	ChatID      int64
	UserID      int
	MsgID       int
	Index       int
	ExpireAt    time.Time
	UserLink    string
	MsgTemplate string
}

type Answer struct {
	Number int
	String string
}
