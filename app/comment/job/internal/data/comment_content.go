package data

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"time"
)

const (
	_commentContentCacheKey = `comment_content_cache:%d` // comment_id
	_commentContentCacheTtl = 12 * 60 * 60               // 12h
)

type CommentContent struct {
	CommentId int64 `gorm:"primarykey"` // 同 CommentIndex.Id

	Ip       int64
	Platform int8
	Device   string

	AtMemberIds string // @的人
	Message     string
	Meta        string

	CreateAt time.Time
	UpdateAt time.Time
}

func (*CommentContent) TableName() string {
	return "comment_content"
}

func (r *commentRepo) CacheContent(ctx context.Context, ids []int64) error {
	var (
		log  = r.log
		g, c = errgroup.WithContext(ctx)
	)

	// cache content
	g.Go(func() (err error) {
		var (
			redis    = r.data.redis.Get()
			moreId   = make([]int64, 0, len(ids))
			contents = make([]*CommentContent, 0, len(ids))
		)
		defer redis.Close()

		for _, id := range ids {
			key := fmt.Sprintf(_commentContentCacheKey, id)
			// 如果存在直接延时
			reply, err := redis.Do("expire", key, _commentContentCacheTtl)
			if err != nil {
				log.Error(err)
			} else {
				if res, ok := reply.(int64); ok && res == 1 { // 延时成功，原来已经存在
					continue
				}
			}
			// 没有延时成功
			moreId = append(moreId, id)
		}
		if len(moreId) == 0 {
			return nil
		}

		result := r.data.db.
			WithContext(c).
			Where("comment_id IN (?)", moreId).
			Find(&contents)
		if result.Error != nil {
			return result.Error
		}

		return r.setCommentContentCache(c, contents...)
	})

	// cache index
	g.Go(func() (err error) {
		var (
			redis  = r.data.redis.Get()
			moreId = make([]int64, 0, len(ids))
			indexs = make([]*CommentIndex, 0, len(ids))
		)
		defer redis.Close()

		for _, id := range ids {
			key := fmt.Sprintf(_commentIndexCacheKey, id)
			// 如果存在直接延时
			reply, err := redis.Do("expire", key, _commentIndexCacheTtl)
			if err != nil {
				log.Error(err)
			} else if res, ok := reply.(int64); ok && res == 1 { // 延时成功，原来已经存在
				continue
			}

			// 没有延时成功
			moreId = append(moreId, id)
		}
		if len(moreId) == 0 {
			return nil
		}

		result := r.data.db.
			WithContext(c).
			Where("id IN (?)", moreId).
			Find(&indexs)
		if result.Error != nil {
			return result.Error
		}

		return r.setCommentIndexCache(c, indexs...)
	})

	err := g.Wait()
	if err != nil {
		return err
	}

	return nil
}
