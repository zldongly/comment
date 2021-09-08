package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"gorm.io/gorm"
	"time"
)

const (
	_commentSortCacheKey       = `comment_sort_cache:%d:%d`     // obj_id, obj_type
	_commentIndexCacheKey      = `comment_index_cache:%d`       // id
	_commentReplyIndexCacheKey = `comment_reply_index_cache:%d` // root_id

	_commentSortCacheTtl       = 8 * 60 * 60 // 8h
	_commentIndexCacheTtl      = 8 * 60 * 60 // 8h
	_commentReplyIndexCacheTtl = 8 * 60 * 60 // 8h
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

func (r *commentRepo) CacheIndex(ctx context.Context, objId int64, objType int32, pageNo, pageSize int32) error {
	var (
		key    = fmt.Sprintf(_commentSortCacheKey, objId, objType)
		redis  = r.data.redis.Get()
		log    = r.log
		offset int64                             // db offset
		count  = int64(pageNo) * int64(pageSize) // 总数
		indexs []*CommentIndex
	)
	defer redis.Close()

	// 是否存在key，如果存在直接延时
	reply, err := redis.Do("expire", key, _commentSortCacheTtl)
	if err != nil {
		return err
	}
	if res, ok := reply.(int64); ok && res == 1 { // 存在 key
		reply, err = redis.Do("zcard", key) // 现存index数量
		if err != nil {
			return err
		}
		offset, _ = reply.(int64)
		if offset >= count {
			return nil
		}
	}

	// 查db
	result := r.data.db.
		WithContext(ctx).
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		Where("root = ?", 0).
		Order("create_at DESC").
		Offset(int(offset)).
		Limit(int(count - offset)).
		Find(&indexs)
	if result.Error != nil {
		return result.Error
	}

	// 写入redis
	for _, index := range indexs {
		_, err = redis.Do("zadd", key, index.CreateAt.Unix(), index.Id)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (r *commentRepo) CacheReply(ctx context.Context, rootId int64, pageNo, pageSize int32) error {
	var (
		key    = fmt.Sprintf(_commentReplyIndexCacheKey, rootId)
		redis  = r.data.redis.Get()
		log    = r.log
		offset int64                             // db offset
		count  = int64(pageNo) * int64(pageSize) // 总数
		indexs []*CommentIndex
	)
	defer redis.Close()

	// 是否存在key，如果存在直接延时
	reply, err := redis.Do("expire", key, _commentReplyIndexCacheTtl)
	if err != nil {
		return err
	}
	if res, ok := reply.(int64); ok && res == 1 { // 存在 key
		reply, err = redis.Do("zcard", key) // 现存index数量
		if err != nil {
			return err
		}
		offset, _ = reply.(int64)
		if offset >= count {
			return nil
		}
	}

	// db
	result := r.data.db.
		WithContext(ctx).
		Where("root = ?", rootId).
		Order("floor ASC").
		Offset(int(offset)).
		Limit(int(count - offset)).
		Find(&indexs)
	if result.Error != nil {
		return result.Error
	}

	// 写入redis
	for _, index := range indexs {
		_, err = redis.Do("zadd", key, index.Floor, index.Id)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (r *commentRepo) getCommentIndex(ctx context.Context, id int64) (*CommentIndex, error) {
	var (
		redis = r.data.redis.Get()
		key   = fmt.Sprintf(_commentIndexCacheKey, id)
		log   = r.log
		index CommentIndex
	)
	defer redis.Close()

	if reply, err := redis.Do("get", key); err == nil {
		if buf, ok := reply.([]byte); ok {
			if err = json.Unmarshal(buf, &index); err == nil {
				return &index, nil
			} else {
				log.Error(err)
			}
		}
	} else {
		log.Error(err)
	}

	// db
	result := r.data.db.
		WithContext(ctx).
		Where("id = ?", id).
		First(&index)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("comment_index", "not found")
		}
		return nil, err
	}

	return &index, nil
}

func (r *commentRepo) setCommentIndexCache(ctx context.Context, index *CommentIndex) error {
	var (
		redis = r.data.redis.Get()
		key   = fmt.Sprintf(_commentIndexCacheKey, index.Id)
	)
	defer redis.Close()

	buf, err := json.Marshal(index)
	if err != nil {
		return err
	}
	_, err = redis.Do("setex", key, _commentIndexCacheTtl, buf)
	if err != nil {
		return err
	}

	return nil
}
