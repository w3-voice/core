package entity

import (
	"time"
)

type Status int

const (
    Pending Status = iota
    Sent
    Seen
)

type Message struct {
	ID            string
	CreatedAt     time.Time
	Text          string
	Status        Status
	Author        Contact
}

type Contact struct {
	ID string
	Name string
}

type ChatInfo struct {
	ID string
	Name string
	Members []Contact
}

type Chat struct {
	Info ChatInfo
	Messages []Message
}

func (c Chat) ID() string {
	return c.Info.ID
}



