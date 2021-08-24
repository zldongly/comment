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
	_commentIndexCacheKey           = `comment_index_cache:%d:%d`          // obj_id, obj_type
	_commentIndexLocalCacheKey      = `comment_index_cache:%d:%d:%d:%d`    // obj_id, obj_type, pageNo, pageSize
	_commentReplyIndexCacheKey      = `comment_reply_index_cache:%d`       // root_id
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

func (c *CommentIndex) ToBiz() *biz.CommentIndex {
	index := &biz.CommentIndex{
		Id:             c.Id,
		ObjId:          c.ObjId,
		ObjType:        c.ObjType,
		MemberId:       c.MemberId,
		Root:           c.Root,
		Parent:         c.Parent,
		ParentMemberId: c.ParentMemberId,
		Floor:          c.Floor,
		Count:          c.Count,
		Like:           c.Like,
		Hate:           c.Hate,
		State:          c.State,
		Attrs:          c.Attrs,
		CreateAt:       c.CreateAt,
		Replies:        make([]*biz.CommentIndex, 0, len(c.Replies)),
	}
	for _, reply := range c.Replies {
		index.Replies = append(index.Replies, reply.ToBiz())
	}
	return index
}

func (r *commentRepo) ListCommentIndex(ctx context.Context, objType int32, objId int64, pageNo int32, pageSize int32) ([]*biz.CommentIndex, error) {
	var (
		list     = make([]*biz.CommentIndex, 0, pageSize)
		comments = make([]*CommentIndex, 0, pageSize)
		offset   = convert.Offset(pageNo, pageSize)
		key      = fmt.Sprintf(_commentIndexCacheKey, objId, objType)
		redis    = r.data.redis.Get()
		log      = r.log
	)
	defer redis.Close()

	// 先查redis
	if reply, err := redis.Do("zrevrange", key, offset, offset+pageSize); err == nil {
		if replies, ok := reply.([]interface{}); ok {
			for _, rep := range replies {
				if bs, ok := rep.([]byte); ok {
					var comment CommentIndex
					if err = json.Unmarshal(bs, &comment); err == nil {
						comments = append(comments, &comment)
					} else {
						log.Error(err)
					}
				}
			}
		}
	} else {
		log.Error(err)
	}
	if len(comments) >= int(pageSize) {
		for _, comment := range comments {
			list = append(list, comment.ToBiz())
		}
		return list, nil
	}

	// 查本地缓存
	key = fmt.Sprintf(_commentIndexLocalCacheKey, objId, objType, pageNo, pageSize)
	if buf, err := r.data.cache.Get([]byte(key)); err == nil {
		if err = json.Unmarshal(buf, &comments); err == nil {
			for _, comment := range comments {
				list = append(list, comment.ToBiz())
			}
			return list, nil
		} else {
			log.Error(err)
		}
	} else if !errors.Is(err, freecache.ErrNotFound) {
		log.Error(err)
	}

	// database
	db := r.data.db
	result := db.WithContext(ctx).
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		Where("root = 0").
		Order("create_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&comments)

	if result.Error != nil {
		return list, result.Error
	}
	// 查回复reply
	var replies []*CommentIndex
	commentIds := make([]int64, 0, len(comments))
	for _, comment := range comments {
		commentIds = append(commentIds, comment.Id)
	}
	if len(commentIds) == 0 {
		return list, nil
	}
	result = db.WithContext(ctx).
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		Where("root IN (?)", commentIds).
		Where("floor <= ?", 3).
		Find(&replies)
	if result.Error != nil {
		return list, result.Error
	}
	mReplies := make(map[int64][]*CommentIndex, len(replies))
	for idx, _ := range replies {
		reply := replies[idx]
		l := mReplies[reply.Root]
		mReplies[reply.Root] = append(l, reply)
	}
	for idx, _ := range comments {
		comment := comments[idx]
		comment.Replies = mReplies[comment.Id]
	}

	// 填入本地缓存
	if buf, err := json.Marshal(comments); err == nil {
		if err = r.data.cache.Set([]byte(key), buf, _localCacheExpire); err != nil {
			log.Error(err)
		}
	}

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

	for _, comment := range comments {
		list = append(list, comment.ToBiz())
	}
	return list, nil
}

func (r *commentRepo) ListReplyIndex(ctx context.Context, rootId int64, pageNo int32, pageSize int32) ([]*biz.CommentIndex, error) {
	var (
		log    = r.log
		redis  = r.data.redis.Get()
		key    = fmt.Sprintf(_commentReplyIndexCacheKey, rootId)
		offset = convert.Offset(pageNo, pageSize)
		list   = make([]*biz.CommentIndex, 0, pageSize)
		indexs = make([]*CommentIndex, 0, pageSize)
	)
	defer redis.Close()

	// redis
	if reply, err := redis.Do("zrange", key, offset, pageSize); err == nil {
		if replies, ok := reply.([]interface{}); ok {
			for _, rep := range replies {
				if bs, ok := rep.([]byte); ok {
					var comment CommentIndex
					if err = json.Unmarshal(bs, &comment); err == nil {
						indexs = append(indexs, &comment)
					} else {
						log.Error(err)
					}
				}
			}
		}
	} else {
		log.Error(err)
	}
	if len(indexs) >= int(pageSize) {
		for _, index := range indexs {
			list = append(list, index.ToBiz())
		}

		return list, nil
	}

	// cache本地缓存
	key = fmt.Sprintf(_commentReplyIndexLocalCacheKey, rootId, pageNo, pageSize)
	if buf, err := r.data.cache.Get([]byte(key)); err == nil {
		if err = json.Unmarshal(buf, &indexs); err == nil {
			for _, index := range indexs {
				list = append(list, index.ToBiz())
				return list, nil
			}
		} else if !errors.Is(err, freecache.ErrNotFound) {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	// database
	result := r.data.db.
		WithContext(ctx).
		Where("root = ?", rootId).
		Offset(int(offset)).
		Limit(int(pageSize)).
		Order("floor ASC").
		Find(&indexs)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(indexs) == 0 {
		return list, nil
	}

	// cache
	if buf, err := json.Marshal(indexs); err == nil {
		if err = r.data.cache.Set([]byte(key), buf, _localCacheExpire); err != nil {
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
			Key:   sarama.StringEncoder(fmt.Sprintf("%d+%d", indexs[0].ObjId, indexs[0].ObjType)),
			Value: sarama.ByteEncoder(buf),
		}
		if _, _, err = r.data.kafka.SendMessage(msg); err != nil {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	for _, index := range indexs {
		list = append(list, index.ToBiz())
	}

	return list, nil
}
