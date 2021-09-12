package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/errgroup"
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

/*
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
	Platform    int8
	Device      string
	AtMemberIds []int64
	Message     string
	Meta        string
}
*/
type CommentRepo interface {
	CreateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, id int64) error
	ListCommentId(ctx context.Context, objType int32, objId int64, pageNo int32, pageSize int32) ([]int64, error)
	ListReplyId(ctx context.Context, id int64, pageNo int32, pageSize int32) ([]int64, error)
	ListComment(ctx context.Context, ids []int64) ([]*Comment, error)
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

func (uc *CommentUseCase) ListComment(ctx context.Context, objType int32, objId int64, pageNo, pageSize int32) (*Subject, []*Comment, error) {
	var (
		subject  *Subject
		comments []*Comment
	)

	g, c := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		subject, err = uc.subjectRepo.Get(c, objId, objType)
		return err
	})

	g.Go(func() error {
		var (
			err        error
			commentIds []int64
		)

		// 查索引
		commentIds, err = uc.commentRepo.ListCommentId(c, objType, objId, pageNo, pageSize)
		if err != nil {
			return err
		}

		if len(commentIds) == 0 {
			return nil
		}

		// 查评论和回复的内容
		comments, err = uc.commentRepo.ListComment(ctx, commentIds)
		if err != nil {
			return err
		}

		return nil
	})

	err := g.Wait()

	return subject, comments, err
}

func (uc *CommentUseCase) ListReply(ctx context.Context, rootId int64, pageNo, pageSize int32) ([]*Comment, error) {
	var (
		err      error
		replyIds []int64
		replies  []*Comment
	)

	// index
	replyIds, err = uc.commentRepo.ListReplyId(ctx, rootId, pageNo, pageSize)
	if err != nil {
		return nil, err
	}

	// 查content
	replies, err = uc.commentRepo.ListComment(ctx, replyIds)
	if err != nil {
		return nil, err
	}

	return replies, nil
}

func (uc *CommentUseCase) CreateSubject(ctx context.Context, subject *Subject) error {
	return uc.subjectRepo.Create(ctx, subject)
}

func (uc *CommentUseCase) CreateComment(ctx context.Context, comment *Comment) error {
	return uc.commentRepo.CreateComment(ctx, comment)
}

func (uc *CommentUseCase) DeleteComment(ctx context.Context, id int64) error {
	return uc.commentRepo.DeleteComment(ctx, id)
}
