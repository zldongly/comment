package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zldongly/comment/app/comment/service/internal/biz"
)

const (
	_localCacheExpire = 5
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

func (r *commentRepo) CreateComment(ctx context.Context, comment *biz.Comment) error {
	return nil
}

func (r *commentRepo)DeleteComment(ctx context.Context, id int64) error {
	return nil
}
