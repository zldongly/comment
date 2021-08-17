package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coocood/freecache"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zldongly/comment/app/comment/service/internal/biz"
	"strconv"
	"strings"
	"time"
)

const (
	_commentContentCacheKey = `comment_content_cache:%d`
)

type indexRepo struct {
	data *Data
	log  *log.Helper
}

type CommentContent struct {
	CommentId int64 `gorm:"primarykey"` // 同 CommentIndex.Id

	Ip       string
	Platform string
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

func (c *CommentContent) ToBiz() *biz.CommentContent {
	b := &biz.CommentContent{
		CommentId   : c.CommentId,
		Ip          : c.Ip,
		Platform   : c.Platform,
		Device     : c.Device,
		Message  : c.Message,
		Meta     : c.Meta,
		UpdateAt: c.UpdateAt,
	}
	if len(c.AtMemberIds) == 0 {
		return b
	}
	ids := strings.Split(c.AtMemberIds, ",")
	for _, mid := range ids {
		id , err := strconv.ParseInt(mid, 10, 64)
		if err != nil {
			continue
		}
		b.AtMemberIds = append(b.AtMemberIds, id)
	}
	return b
}

func (r *indexRepo) ListCommentContent(ctx context.Context, ids []int64) ([]*biz.CommentContent, error) {
	var (
		log     = r.log
		err     error
		list    = make([]*CommentContent, 0, len(ids))
		lessIds = make([]int64, 0, len(ids))
		result = make([]*biz.CommentContent, 0, len(ids))
	)

	// 本地缓存
	for _, id := range ids {
		var (
			key = fmt.Sprintf(_commentContentCacheKey, id)
			val []byte
		)

		val, err = r.data.cache.Get([]byte(key))
		if err != nil {
			if !errors.Is(err, freecache.ErrNotFound) {
				log.Error(err)
			}
			lessIds = append(lessIds, id)
			continue
		}

		var content CommentContent
		if err = json.Unmarshal(val, &content); err != nil {
			log.Error(err)
			lessIds = append(lessIds, id)
			continue
		}

		list = append(list, &content)
	}

	ids = lessIds
	lessIds = make([]int64, 0, len(lessIds))
	keys := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		key := fmt.Sprintf(_commentContentCacheKey, id)
		keys = append(keys, key)
	}

	// redis
	if len(ids) > 0 {
		redis := r.data.redis.Get()
		if reply, err := redis.Do("mget", keys...); err == nil {
			if replies, ok := reply.([]interface{}); ok {
				for idx, rep := range replies {
					if rep == nil {
						lessIds = append(lessIds, ids[idx])
						continue
					}

					if bs, ok := rep.([]byte); ok {
						var content CommentContent
						if err = json.Unmarshal(bs, &content); err != nil {
							lessIds = append(lessIds, ids[idx])
							log.Error(err)
						} else {
							list = append(list, &content)
						}
					} else {
						lessIds = append(lessIds, ids[idx])
					}
				}
			} else {
				lessIds = ids
			}
		} else {
			log.Error(err)
			lessIds = ids
		}
		redis.Close()
	}

	ids = lessIds
	// database
	if len(ids) > 0 {
		var contents []*CommentContent
		result := r.data.db.
			Where("comment_id IN (?)", ids).
			Find(&contents)
		if result.Error != nil {
			return nil, result.Error
		}

		list = append(list, contents...)
	}

	for _, content := range list {
		result = append(result, content.ToBiz())
	}

	return result, nil
}
