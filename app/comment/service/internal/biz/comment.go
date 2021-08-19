package biz

import (
	"time"
)

type Comment struct {
	Id int64

	MemberId       int64
	Root           int64
	Parent         int64
	ParentMemberId int64

	Floor int32
	Count int32
	Like  int32
	Hate  int32
	State int8
	Attrs int32

	AtMemberIds []int64
	Message     string
	Meta        string

	Ip       string
	Platform int8
	Device   string

	CreateAt time.Time

	Replies []*Comment
}

type CommentRepo interface {
	//ListComment(ctx context.Context, objType int32, objId int64, pageNo int32, pageSize int32) ([]*Comment, error)
}

type CommentUseCase struct {
}
