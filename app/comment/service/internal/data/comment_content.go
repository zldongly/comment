package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/coocood/freecache"
	"github.com/go-kratos/kratos/v2/errors"
	job "github.com/zldongly/comment/api/comment/job/v1"
	"github.com/zldongly/comment/app/comment/service/internal/biz"
	"github.com/zldongly/comment/app/comment/service/internal/pkg/convert"
	"google.golang.org/protobuf/proto"
	"time"
)

const (
	_commentContentCacheKey = `comment_content_cache:%d`
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

func (c *CommentContent) ToBiz() *biz.CommentContent {
	return &biz.CommentContent{
		Id:          c.CommentId,
		Ip:          c.Ip,
		Platform:    c.Platform,
		Device:      c.Device,
		AtMemberIds: convert.StringToInt64s(c.AtMemberIds),
		Message:     c.Message,
		Meta:        c.Meta,
	}
}

func (r *commentRepo) ListCommentContent(ctx context.Context, ids []int64) ([]*biz.CommentContent, error) {
	var (
		log             = r.log
		err             error
		list            = make([]*CommentContent, 0, len(ids))
		commentContents = make([]*biz.CommentContent, 0, len(ids))
		lessIds         = make([]int64, 0, len(ids))
		cache           = r.data.cache
	)

	// 本地缓存
	for _, id := range ids {
		var (
			key = fmt.Sprintf(_commentContentCacheKey, id)
			val []byte
		)

		val, err = cache.Get([]byte(key))
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
			WithContext(ctx).
			Where("comment_id IN (?)", ids).
			Find(&contents)
		if result.Error != nil {
			return nil, result.Error
		}

		list = append(list, contents...)

		// 填入本地缓存
		for _, content := range contents {
			buf, err := json.Marshal(content)
			if err != nil {
				log.Error(err)
				continue
			}
			key := fmt.Sprintf(_commentContentCacheKey, content.CommentId)
			err = cache.Set([]byte(key), buf, _localCacheExpire)
			if err != nil {
				log.Error(err)
			}
		}
	}

	for _, content := range list {
		commentContents = append(commentContents, content.ToBiz())
	}

	// kafka
	if len(ids) > 0 {
		k := &job.CacheContentReq{
			CommentIds: ids,
		}
		if buf, err := proto.Marshal(k); err != nil {
			log.Error(err)
		} else {
			msg := &sarama.ProducerMessage{
				Topic: job.TopicCacheContent,
				//Key: sarama.ByteEncoder{}, // obj_id+obj_type
				Value: sarama.ByteEncoder(buf),
			}
			_, _, err = r.data.kafka.SendMessage(msg)
			if err != nil {
				log.Error(err)
			}
		}
	}

	return commentContents, nil
}
