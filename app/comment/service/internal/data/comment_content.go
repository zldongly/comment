package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
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

func toComment(index *CommentIndex, content *CommentContent) *biz.Comment {
	return &biz.Comment{
		Id:             index.Id,
		ObjId:          index.ObjId,
		ObjType:        index.ObjType,
		MemberId:       index.MemberId,
		Root:           index.Root,
		Parent:         index.Parent,
		ParentMemberId: index.ParentMemberId,
		Floor:          index.Floor,
		Count:          index.Count,
		Like:           index.Like,
		Hate:           index.Hate,
		State:          index.State,
		Attrs:          index.Attrs,
		AtMemberIds:    convert.StringToInt64s(content.AtMemberIds),
		Message:        content.Message,
		Meta:           content.Meta,
		Ip:             content.Ip,
		Platform:       content.Platform,
		Device:         content.Device,
		CreateAt:       index.CreateAt,
		Replies:        make([]*biz.Comment, 0, len(index.Replies)),
	}
}

func (r *commentRepo) ListComment(ctx context.Context, ids []int64) ([]*biz.Comment, error) {
	var (
		err      error
		comments []*biz.Comment
		indexs   []*CommentIndex
		contents []*CommentContent
		mIndex = make(map[int64]*CommentIndex, len(ids))
		mContent = make(map[int64]*CommentContent, len(ids))
	)

	indexs, err = r.listCommentIndex(ctx, ids)
	if err != nil {
		return nil, err
	}

	contentIds := make([]int64, 0, len(ids)*4)
	for _, index := range indexs {
		mIndex[index.Id] = index

		contentIds = append(contentIds, index.Id)
		for _, reply := range index.Replies {
			contentIds = append(contentIds, reply.Id)
		}
	}

	contents, err = r.listCommentContent(ctx, contentIds)
	if err != nil {
		return nil, err
	}
	for _, content := range contents {
		mContent[content.CommentId] = content
	}

	// 按id顺序合并index&content
	for _, id := range ids {
		index := mIndex[id]
		if index == nil {
			continue
		}
		content := mContent[id]
		if content == nil {
			continue
		}

		comment := toComment(index, content)
		for _, reply := range index.Replies {
			content = mContent[reply.Id]
			if content == nil {
				continue
			}

			comment.Replies = append(comment.Replies, toComment(reply, content))
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (r *commentRepo) listCommentIndex(ctx context.Context, ids []int64) ([]*CommentIndex, error) {
	var (
		list    []*CommentIndex
		redis   = r.data.redis.Get()
		log     = r.log
		keys    = make([]interface{}, 0, len(ids))
		lessIds = make([]int64, 0, len(ids))
	)
	defer redis.Close()

	// redis
	for _, id := range ids {
		key := fmt.Sprintf(_commentIndexCacheKey, id)
		keys = append(keys, key)
	}
	if reply, err := redis.Do("mget", keys...); err == nil {
		if replies, ok := reply.([]interface{}); ok {
			for idx, rep := range replies {
				if rep == nil {
					lessIds = append(lessIds, ids[idx])
					continue
				}

				if buf, ok := rep.([]byte); ok {
					var index CommentIndex
					if err = json.Unmarshal(buf, &index); err != nil {
						lessIds = append(lessIds, ids[idx])
						log.Error(err)
					} else {
						list = append(list, &index)
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

	if len(lessIds) == 0 {
		return list, nil
	}

	// local cache
	ids = lessIds
	lessIds = make([]int64, 0, len(lessIds))
	for _, id := range ids {
		key := fmt.Sprintf(_commentIndexCacheKey, id)
		if buf, err := r.data.cache.Get([]byte(key)); err == nil {
			var index CommentIndex
			if err = json.Unmarshal(buf, &index); err == nil {
				list = append(list, &index)
			} else {
				lessIds = append(lessIds, id)
				log.Error(err)
			}
		} else {
			lessIds = append(lessIds, id)
			log.Error(err)
		}
	}

	if len(lessIds) == 0 {
		return list, nil
	}

	// database
	ids = lessIds
	var indexs []*CommentIndex
	result := r.data.db.
		WithContext(ctx).
		Where("id IN (?)", ids).
		Where("state = 0").
		Find(&indexs)
	if err := result.Error; err != nil {
		return nil, err
	}
	commentIds := make([]int64, 0, len(indexs)*3)
	for _, index := range indexs {
		if index.Root == 0 {
			commentIds = append(commentIds, index.Id)
		}
	}
	// 回复
	if len(commentIds) > 0 {
		var replies []*CommentIndex
		result = r.data.db.
			WithContext(ctx).
			Where("root IN (?)", commentIds).
			Where("floor <= 3").
			Where("state = 0").
			Order("floor ASC").
			Find(&replies)
		if err := result.Error; err != nil {
			return nil, err
		}
		mReply := make(map[int64][]*CommentIndex) // m[root]
		for _, reply := range replies {
			list := mReply[reply.Root]
			mReply[reply.Root] = append(list, reply)
		}

		for _, index := range indexs {
			index.Replies = mReply[index.Id]
		}
	}

	list = append(list, indexs...)

	// local cache
	for _, index := range indexs {
		key := fmt.Sprintf(_commentIndexCacheKey, index.Id)
		if buf, err := json.Marshal(index); err == nil {
			if err = r.data.cache.Set([]byte(key), buf, _localCacheExpire); err != nil {
				log.Error(err)
			}
		} else {
			log.Error(err)
		}
	}

	// kafka
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

	return list, nil
}

func (r *commentRepo) listCommentContent(ctx context.Context, ids []int64) ([]*CommentContent, error) {
	var (
		list    []*CommentContent
		redis   = r.data.redis.Get()
		log     = r.log
		keys    = make([]interface{}, 0, len(ids))
		lessIds = make([]int64, 0, len(ids))
	)
	defer redis.Close()

	// redis
	for _, id := range ids {
		key := fmt.Sprintf(_commentContentCacheKey, id)
		keys = append(keys, key)
	}
	if reply, err := redis.Do("mget", keys...); err == nil {
		if replies, ok := reply.([]interface{}); ok {
			for idx, rep := range replies {
				if rep == nil {
					lessIds = append(lessIds, ids[idx])
					continue
				}

				if buf, ok := rep.([]byte); ok {
					var content CommentContent
					if err = json.Unmarshal(buf, &content); err != nil {
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

	if len(lessIds) == 0 {
		return list, nil
	}

	// local cache
	ids = lessIds
	lessIds = make([]int64, 0, len(lessIds))
	for _, id := range ids {
		key := fmt.Sprintf(_commentContentCacheKey, id)
		if buf, err := r.data.cache.Get([]byte(key)); err == nil {
			var content CommentContent
			if err = json.Unmarshal(buf, &content); err == nil {
				list = append(list, &content)
			} else {
				lessIds = append(lessIds, id)
				log.Error(err)
			}
		} else {
			lessIds = append(lessIds, id)
			log.Error(err)
		}
	}

	if len(lessIds) == 0 {
		return list, nil
	}

	// database
	ids = lessIds
	var contnets []*CommentContent
	result := r.data.db.
		WithContext(ctx).
		Where("id IN (?)", ids).
		Where("state = 0").
		Find(&contnets)
	if err := result.Error; err != nil {
		return nil, err
	}

	list = append(list, contnets...)

	// local cache
	for _, content := range contnets {
		key := fmt.Sprintf(_commentContentCacheKey, content.CommentId)
		if buf, err := json.Marshal(content); err == nil {
			if err = r.data.cache.Set([]byte(key), buf, _localCacheExpire); err != nil {
				log.Error(err)
			}
		} else {
			log.Error(err)
		}
	}

	// kafka
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

	return list, nil
}
