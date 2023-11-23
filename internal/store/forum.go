package store

import "github.com/leighmacdonald/steamid/v3/steamid"

type ForumCategory struct {
	CategoryID  int    `json:"category_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordering    int    `json:"ordering"`
	TimeStamped
}

type Forum struct {
	ForumID       int    `json:"forum_id"`
	CategoryID    int    `json:"category_id"`
	LastThreadID  int64  `json:"last_thread_id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Ordering      int    `json:"ordering"`
	CountThreads  int64  `json:"count_threads"`
	CountMessages int64  `json:"count_messages"`
	TimeStamped
}

type ForumThread struct {
	ThreadID int64         `json:"thread_id"`
	ForumID  int           `json:"forum_id"`
	SourceID steamid.SID64 `json:"source_id"`
	Title    string        `json:"title"`
	Sticky   bool          `json:"sticky"`
	Locked   bool          `json:"locked"`
	Views    int64         `json:"views"`
	TimeStamped
}

type ForumMessage struct {
	MessageID int64         `json:"message_id"`
	ThreadID  int64         `json:"thread_id"`
	SourceID  steamid.SID64 `json:"source_id"`
	BodyMD    string        `json:"body_md"`
	TimeStamped
}

type ForumMessageVote struct {
	MessageID int64         `json:"message_id"`
	SourceID  steamid.SID64 `json:"source_id"`
	Vote      int           `json:"vote"` // -1/+1
	TimeStamped
}
