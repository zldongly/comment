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

func (c *Comment) merge(index *CommentIndex, content *CommentContent) *Comment {
	c.Id = index.Id
	c.ObjId = index.ObjId
	c.ObjType = index.ObjType
	c.MemberId = index.MemberId
	c.Root = index.Root
	c.Parent = index.Parent
	c.ParentMemberId = index.ParentMemberId
	c.Floor = index.Floor
	c.Count = index.Count
	c.Like = index.Like
	c.Hate = index.Hate
	c.State = index.State
	c.Attrs = index.Attrs
	c.AtMemberIds = content.AtMemberIds
	c.Message = content.Message
	c.Meta = content.Meta
	c.Ip = content.Ip
	c.Platform = content.Platform
	c.Device = content.Device
	c.CreateAt = index.CreateAt
	c.Replies = make([]*Comment, 0, len(index.Replies))
	return c
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
		log:         log.NewHelper(log.With(logger, "module", "usecase/comment")),
	}
}

func (uc *CommentUseCase) ListComment(ctx context.Context, objType int32, objId int64, pageNo, pageSize int32) (*Subject, []*Comment, error) {
	var (
		subject  *Subject
		comments = make([]*Comment, 0, pageSize)
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
			indexs     []*CommentIndex
			contents   []*CommentContent
			commentIds = make([]int64, 0, pageSize*3)
		)

		// 查索引
		indexs, err = uc.commentRepo.ListCommentIndex(c, objType, objId, pageNo, pageSize)
		if err != nil {
			return err
		}

		// 取出所有评论和回复的id
		for _, index := range indexs {
			commentIds = append(commentIds, index.Id)
			for _, reply := range index.Replies {
				commentIds = append(commentIds, reply.Id)
			}
		}

		// 查评论和回复的内容
		contents, err = uc.commentRepo.ListCommentContent(ctx, commentIds)
		if err != nil {
			return err
		}
		mContent := make(map[int64]*CommentContent, len(contents))
		for _, content := range contents {
			content := content
			mContent[content.Id] = content
		}

		// 合并 index 和 content
		for _, index := range indexs {
			content := mContent[index.Id]
			if content == nil {
				continue
			}

			comment := new(Comment)
			comment.merge(index, content)
			// 回复
			for _, reply := range index.Replies {
				content = mContent[reply.Id]
				if content == nil {
					continue
				}

				r := new(Comment)
				r.merge(reply, content)
				comment.Replies = append(comment.Replies, r)
			}
			comments = append(comments, comment)
		}

		return nil
	})

	err := g.Wait()

	return subject, comments, err
}

func (uc *CommentUseCase) ListReply(ctx context.Context, rootId int64, pageNo, pageSize int32) ([]*Comment, error) {
	var (
		err      error
		indexs   []*CommentIndex
		contents []*CommentContent
		mContent = make(map[int64]*CommentContent, pageSize)
		replies  = make([]*Comment, 0, pageSize)
	)

	// index
	indexs, err = uc.commentRepo.ListReplyIndex(ctx, rootId, pageNo, pageSize)
	if err != nil {
		return nil, err
	}
	commentIds := make([]int64, 0, len(indexs))
	for _, index := range indexs {
		commentIds = append(commentIds, index.Id)
	}

	// 查content
	contents, err = uc.commentRepo.ListCommentContent(ctx, commentIds)
	if err != nil {
		return nil, err
	}
	for idx, _ := range contents {
		content := contents[idx]
		mContent[content.Id] = content
	}

	// 合并 index和content
	for _, index := range indexs {
		content := mContent[index.Id]
		if content == nil {
			continue
		}
		reply := new(Comment).merge(index, content)
		replies = append(replies, reply)
	}

	return replies, nil
}
