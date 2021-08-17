package biz

import "time"

type CommentContent struct {
	CommentId   int64 `gorm:"primarykey"` // 同 CommentIndex.Id
	Ip          string
	Platform    string
	Device      string
	AtMemberIds []int64 // @的人
	Message     string
	Meta        string
	UpdateAt    time.Time
}

type CommentRepo interface {
}
