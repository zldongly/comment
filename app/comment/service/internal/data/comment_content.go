package data

import "time"

type CommentContent struct {
	CommentId int64 `gorm:"primarykey"` // 同 CommentIndex.Id

	Ip          string
	Platform    string
	Device      string

	AtMemberIds string // @的人
	Message string
	Meta    string

	CreateAt time.Time
	UpdateAt time.Time
}

func (*CommentContent) TableName() string {
	return "comment_content"
}
