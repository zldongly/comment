package biz

import (
	"context"
	"time"
)

type CommentContent struct {
	CommentId   int64
	Ip          string
	Platform    string
	Device      string
	AtMemberIds []int64 // @的人
	Message     string
	Meta        string
	UpdateAt    time.Time
}

type ContentRepo interface {
	ListCommentContent(ctx context.Context, ids []int64) ([]*CommentContent, error)
}
