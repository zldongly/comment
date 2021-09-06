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

// Comment
// 只做redis缓存不存如mysql
type Comment struct {
	Id             int64
	ObjId          int64
	ObjType        int32
	MemberId       int64
	Root           int64 // 根评论ID
	Parent         int64 // 父评论
	ParentMemberId int64 // 回复的人
	Floor          int32 // 楼层
	Count          int32 // 回复数量
	Like           int32 // 点赞
	Hate           int32 // 点睬
	State          int8  // 状态
	Attrs          int32 // 属性，bit 置顶等
	Ip             int64
	Platform       int8
	Device         string
	AtMemberIds    string // @的人
	Message        string
	Meta           string
	CreateAt       time.Time
	Replies        []*Comment
}

func (c *Comment) merge(index *CommentIndex, content *CommentContent) *Comment {
	c.Id = index.Id
	c.ObjId = index.ObjId
	c.ObjType = index.ObjType
	c.MemberId = index.MemberId
	c.Root = index.Root
	c.Parent = index.Parent
	c.ParentMemberId = index.ParentMemberId
	c.Floor = index.Floor
	c.Count = index.Count
	c.Like = index.Like
	c.Hate = index.Hate
	c.State = index.State
	c.Attrs = index.Attrs
	c.Ip = content.Ip
	c.Platform = content.Platform
	c.Device = content.Device
	c.AtMemberIds = content.AtMemberIds
	c.Message = content.Message
	c.Meta = content.Meta
	c.CreateAt = index.CreateAt
	c.Replies = make([]*Comment, 0, len(index.Replies))
	return c
}

func (r *commentRepo) CacheContent(ctx context.Context, ids []int64) error {
	var (
		log      = r.log
		redis    = r.data.redis.Get()
		moreId   = make([]int64, 0, len(ids))
		contents = make([]*CommentContent, 0, len(ids))
		indexs   = make([]*CommentIndex, 0, len(ids))
		comments = make([]*Comment, 0, len(ids))
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
		// 没有延时成
		moreId = append(moreId, id)
	}

	if len(moreId) == 0 {
		return nil
	}

	result := r.data.db.
		WithContext(ctx).
		Where("id IN (?)", moreId).
		First(&contents)
	if result.Error != nil {
		return result.Error
	}
	commentIds := make([]int64, 0, len(indexs))          // 一级评论的ID
	mIndex := make(map[int64]*CommentIndex, len(indexs)) // m[index.id]
	for _, index := range indexs {
		if index.Root == 0 {
			commentIds = append(commentIds, index.Id)
			mIndex[index.Id] = index
		}
	}
	// 查一级评论下的回复
	if len(commentIds) > 0 {
		var replies []*CommentIndex

		result = r.data.db.
			WithContext(ctx).
			Where("root IN (?)", commentIds).
			Where("floor <= 3").
			Find(&replies)
		if result.Error != nil {
			return result.Error
		}
		for _, reply := range replies {
			moreId = append(moreId, reply.Id)

			// 回复填入对应的一级评论
			if index := mIndex[reply.Root]; index != nil {
				index.Replies = append(index.Replies, reply)
			}
		}
	}

	// 查所有评论及回复的content
	result = r.data.db.
		WithContext(ctx).
		Where("comment_id IN (?)", moreId).
		First(&contents)
	if result.Error != nil {
		return result.Error
	}

	mContent := make(map[int64]*CommentContent, len(contents)) //m[comment.id]
	for _, content := range contents {
		mContent[content.CommentId] = content
	}

	// 合并index和content
	for _, index := range indexs {
		content := mContent[index.Id]
		if content == nil {
			continue
		}
		var comment Comment
		comment.merge(index, content)

		for _, replyIndex := range index.Replies {
			content := mContent[index.Id]
			if content == nil {
				continue
			}
			var reply Comment
			reply.merge(replyIndex, content)
			comment.Replies = append(comment.Replies, &reply)
		}

		comments = append(comments, &comment)
	}

	// 写入redis
	for _, comment := range comments {
		key := fmt.Sprintf(_commentContentCacheKey, comment.Id)
		if buf, err := json.Marshal(comment); err == nil {
			_, err = redis.Do("setex", key, _commentContentCacheTtl, buf)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
