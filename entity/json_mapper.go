package entity

import "encoding/json"

type JsonMessage interface {
	Json() ([]byte, error)
}

func (m *Message) Json() ([]byte, error) {
	return json.Marshal(*m)
}


func (m *Contact) Json() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *Identity) Json() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *ChatInfo) Json() ([]byte, error) {
	return json.Marshal(*m)
}



type ChatSlice []ChatInfo
func (m ChatSlice) Json() ([]byte, error) {
	return json.Marshal(m)
}

type MessageSlice []Message
func (m MessageSlice) Json() ([]byte, error) {
	return json.Marshal(m)
}

type ContactSlice []Contact
func (m ContactSlice) Json() ([]byte, error) {
	return json.Marshal(m)
}

