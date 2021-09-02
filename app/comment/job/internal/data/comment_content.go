package data

import (
	"context"
	"encoding/json"
	"fmt"
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
		log = r.log
		redis = r.data.redis.Get()
	)
	defer redis.Close()

	for _, id := range ids {
		var (
			key = fmt.Sprintf(_commentContentCacheKey, id)
			content CommentContent
			buf []byte
		)

		// 如果存在直接延时
		reply, err := redis.Do("expire", key, _commentContentCacheTtl)
		if err != nil {
			log.Error(err)
		} else {
			if res, ok := reply.(int64); ok && res == 1 {
				return nil
			}
		}

		// database
		result := r.data.db.
			WithContext(ctx).
			Where("comment_id = ?", id).
			First(&content)
		if result.Error != nil {
			return result.Error
		}

		// 填入redis
		buf, err = json.Marshal(content)
		if err != nil {
			return err
		}
		_, err = redis.Do("setex", key, _commentContentCacheTtl, buf)
		if err != nil {
			return err
		}
	}

	return nil
}
