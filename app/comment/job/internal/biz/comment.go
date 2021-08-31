package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type Comment struct {
	Id int64

	ObjId   int64
	ObjType int32

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

type CommentRepo interface {
	Create(ctx context.Context, c *Comment) error
	Delete(ctx context.Context, id int64) error
	CacheIndex(ctx context.Context, objId int64, objType int32, pageNo, pageSize int32) error
	CacheContent(ctx context.Context, ids []int64) error
	CacheReply(ctx context.Context, rootId int64, pageNo, pageSize int32) error
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
		log:         log.NewHelper(log.With(logger, "module", "usecase/comment")),
	}
}
