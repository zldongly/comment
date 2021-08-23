package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
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

	Ip       int64
	Platform int8
	Device   string

	CreateAt time.Time

	Replies []*Comment
}

type CommentIndex struct {
	Id             int64
	ObjId          int64
	ObjType        int32
	MemberId       int64
	Root           int64
	Parent         int64
	ParentMemberId int64
	Floor          int32
	Count          int32
	Like           int32
	Hate           int32
	State          int8
	Attrs          int32
	CreateAt       time.Time

	Replies []*CommentIndex
}

type CommentContent struct {
	Id          int64
	Ip          int64
	Platform    string
	Device      string
	AtMemberIds []int64
	Message     string
	Meta        string
}

type CommentRepo interface {
	CreateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, id int64) error
	ListCommentIndex(ctx context.Context, objType int32, objId int64, pageNo int32, pageSize int32) ([]*CommentIndex, error)
	ListReplyIndex(ctx context.Context, id int64, pageNo int32, pageSize int32) ([]*CommentIndex, error)
	ListCommentContent(ctx context.Context, ids []int64) ([]*CommentContent, error)
}

type CommentUseCase struct {
	commentRepo CommentRepo
	subjectRepo SubjectRepo
	log         *log.Helper
}

func NewCommentUseCase(comment CommentRepo, subject SubjectRepo, logger log.Logger) *CommentUseCase {
	return &CommentUseCase{
		commentRepo: comment,
		subjectRepo: subject,
		log: log.NewHelper(log.With(logger, "module", "usecase/comment")),
	}
}
