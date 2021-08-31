package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zldongly/comment/app/comment/job/internal/biz"
)

var _ biz.CommentRepo = (*commentRepo)(nil)

type commentRepo struct {
	data *Data
	log  *log.Helper
}

func NewCommentRepo(data *Data, logger log.Logger) biz.CommentRepo {
	return &commentRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/comment")),
	}
}

func (r *commentRepo) Create(ctx context.Context, c *biz.Comment) error {
	return nil
}

func (r *commentRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func (r *commentRepo) CacheIndex(ctx context.Context, objId int64, objType int32, pageNo, pageSize int32) error {
	return nil
}

func (r *commentRepo) CacheContent(ctx context.Context, ids []int64) error {
	return nil
}

func (r *commentRepo) CacheReply(ctx context.Context, rootId int64, pageNo, pageSize int32) error {
	return nil
}
