package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/coocood/freecache"
	"github.com/go-kratos/kratos/v2/errors"
	job "github.com/zldongly/comment/api/comment/job/v1"
	"github.com/zldongly/comment/app/comment/service/internal/pkg/convert"
	"google.golang.org/protobuf/proto"
	"time"
)

const (
	_commentSortCacheKey      = `comment_sort_cache:%d:%d`    // obj_id, obj_type
	_commentReplySortCacheKey = `comment_reply_sort_cache:%d` // root_id
	_commentIndexCacheKey     = `comment_index_cache:%d`      // id

	_commentIndexLocalCacheKey      = `comment_index_cache:%d:%d:%d:%d`    // obj_id, obj_type, pageNo, pageSize
	_commentReplyIndexLocalCacheKey = `comment_reply_index_cache:%d:%d:%d` // root_id, pageNo, pageSize
)

type CommentIndex struct {
	Id       int64
	ObjId    int64
	ObjType  int32
	MemberId int64

	Root           int64 // 根评论ID
	Parent         int64 // 父评论
	ParentMemberId int64 // 回复的人
	Floor          int32 // 楼层
	Count          int32 // 回复数量

	Like int32 // 点赞
	Hate int32 // 点睬

	State int8  // 状态
	Attrs int32 // 属性，bit 置顶等

	CreateAt time.Time
	UpdateAt time.Time

	Replies []*CommentIndex `gorm:"-"`
}

func (*CommentIndex) TableName() string {
	return "comment_index"
}

func (r *commentRepo) ListCommentId(ctx context.Context, objType int32, objId int64, pageNo int32, pageSize int32) ([]int64, error) {
	var (
		key        = fmt.Sprintf(_commentSortCacheKey, objId, objType)
		redis      = r.data.redis.Get()
		log        = r.log
		offset     = convert.Offset(pageNo, pageSize)
		commentIds = make([]int64, 0, pageSize)
	)
	defer redis.Close()

	// 查redis
	if reply, err := redis.Do("zrevrange", key, offset, offset+pageSize-1); err == nil {
		if list, ok := reply.([]interface{}); ok {
			for _, item := range list {
				if id, ok := item.(int64); ok {
					commentIds = append(commentIds, id)
				}
			}

			if len(commentIds) >= int(pageSize) {
				return commentIds, nil
			}
		}
	} else {
		log.Error(err)
	}

	// cache
	cacheKey := fmt.Sprintf(_commentIndexLocalCacheKey, objId, objType, pageNo, pageSize)
	if reply, err := r.data.cache.Get([]byte(cacheKey)); err == nil {
		var ids []int64
		if err = json.Unmarshal(reply, &ids); err == nil {
			if len(ids) >= int(pageSize) {
				return ids, nil
			}
		} else {
			log.Error(err)
		}
	} else if !errors.Is(err, freecache.ErrNotFound) {
		log.Error(err)
	}

	var comments []*CommentIndex
	result := r.data.db.
		WithContext(ctx).
		Select("id").
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		Where("root = 0"). // 一级评论
		Where("state = 0").
		Order("create_at DESC").
		Offset(int(offset) + len(commentIds)).
		Limit(int(pageSize) - len(commentIds)).
		Find(&comments)
	if result.Error != nil {
		return nil, result.Error
	}

	for _, comment := range comments {
		commentIds = append(commentIds, comment.Id)
	}

	// cache
	if buf, err := json.Marshal(commentIds); err == nil {
		if err = r.data.cache.Set([]byte(cacheKey), buf, _localCacheExpire); err != nil {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	// kafka
	k := job.CacheIndexReq{
		ObjId:    objId,
		ObjType:  objType,
		PageNo:   pageNo,
		PageSize: pageSize,
	}
	if buf, err := proto.Marshal(&k); err == nil {
		msg := &sarama.ProducerMessage{
			Topic: job.TopicCacheIndex,
			Key:   sarama.StringEncoder(fmt.Sprintf("%d+%d", objId, objType)),
			Value: sarama.StringEncoder(buf),
		}
		_, _, err = r.data.kafka.SendMessage(msg)
		if err != nil {
			log.Error(err)
		}
	}

	return commentIds, nil
}

func (r *commentRepo) ListReplyId(ctx context.Context, rootId int64, pageNo int32, pageSize int32) ([]int64, error) {
	var (
		key      = fmt.Sprintf(_commentReplySortCacheKey, rootId)
		redis    = r.data.redis.Get()
		log      = r.log
		offset   = convert.Offset(pageNo, pageSize)
		replyIds = make([]int64, 0, pageSize)
	)
	defer redis.Close()

	// 查redis
	if reply, err := redis.Do("zrange", key, offset, offset+pageSize-1); err == nil {
		if list, ok := reply.([]interface{}); ok {
			for _, item := range list {
				if id, ok := item.(int64); ok {
					replyIds = append(replyIds, id)
				}
			}

			if len(replyIds) >= int(pageSize) {
				return replyIds, nil
			}
		}
	} else {
		log.Error(err)
	}

	// cache
	cacheKey := fmt.Sprintf(_commentReplyIndexLocalCacheKey, rootId, pageNo, pageSize)
	if reply, err := r.data.cache.Get([]byte(cacheKey)); err == nil {
		var ids []int64
		if err = json.Unmarshal(reply, &ids); err == nil {
			if len(ids) >= int(pageSize) {
				return ids, nil
			}
		} else {
			log.Error(err)
		}
	} else if !errors.Is(err, freecache.ErrNotFound) {
		log.Error(err)
	}

	var comments []*CommentIndex
	result := r.data.db.
		WithContext(ctx).
		Select("id").
		Where("root = ?", rootId).
		Where("state = 0").
		Order("floor ASC").
		Offset(int(offset) + len(replyIds)).
		Limit(int(pageSize) - len(replyIds)).
		Find(&comments)
	if result.Error != nil {
		return nil, result.Error
	}

	for _, comment := range comments {
		replyIds = append(replyIds, comment.Id)
	}

	// cache
	if buf, err := json.Marshal(replyIds); err == nil {
		if err = r.data.cache.Set([]byte(cacheKey), buf, _localCacheExpire); err != nil {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	// kafka
	k := &job.CacheReplyReq{CommentId: rootId, PageNo: pageNo, PageSize: pageSize}
	if buf, err := proto.Marshal(k); err == nil {
		msg := &sarama.ProducerMessage{
			Topic: job.TopicCacheReply,
			Key:   sarama.StringEncoder(fmt.Sprintf("%d", rootId)),
			Value: sarama.ByteEncoder(buf),
		}
		if _, _, err = r.data.kafka.SendMessage(msg); err != nil {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	return replyIds, nil
}
