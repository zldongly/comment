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

func (r *commentRepo) Create(ctx context.Context, comment *biz.Comment) error {
	// 创建comment database
	// cache comment
	// 修改parent.Count
	// 缓存parent
	return nil
}

func (r *commentRepo) Delete(ctx context.Context, id int64) error {
	// delete database comment
	// delete cache
	// update database parent.Count
	// update cache parent
	return nil
}
