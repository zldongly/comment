package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zldongly/comment/app/comment/job/internal/biz"
	"github.com/zldongly/comment/app/comment/job/internal/pkg/convert"
	"gorm.io/gorm"
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

func (r *commentRepo) Create(ctx context.Context, comment *biz.Comment) (err error) {
	var (
		log     = r.log
		subject *CommentSubject
		root    *CommentIndex
		parent  *CommentIndex
		db      *gorm.DB
	)

	// get subject
	subject, err = r.getSubject(ctx, comment.ObjId, comment.ObjType)
	if err != nil {
		return err
	}

	index := CommentIndex{
		ObjId:          comment.ObjId,
		ObjType:        comment.ObjType,
		MemberId:       comment.MemberId,
		Root:           comment.Root,
		Parent:         comment.Parent,
		ParentMemberId: 0,
		Floor:          0,
	}
	if index.Root == 0 { // 一级评论
		subject.Count++
		subject.RootCount++
		index.Floor = subject.RootCount
	} else { // 这个评论是回复
		root, err = r.getCommentIndex(ctx, index.Root)
		if err != nil {
			return err
		}
		root.Count++
		root.ReplyCount++
		index.Floor = root.ReplyCount

		if index.Parent > 0 {
			parent, err = r.getCommentIndex(ctx, index.Parent)
			if err != nil {
				return err
			}
			index.ParentMemberId = parent.MemberId
		}
	}
	subject.AllCount++
	result := db.WithContext(ctx).Create(&index)
	if err = result.Error; err != nil {
		return err
	}

	content := CommentContent{
		CommentId:   index.Id,
		Ip:          comment.Ip,
		Platform:    comment.Platform,
		Device:      comment.Device,
		AtMemberIds: convert.Int64sToString(comment.AtMemberIds),
		Message:     comment.Message,
		Meta:        comment.Meta,
	}
	result = db.WithContext(ctx).Create(&content)
	if err = result.Error; err != nil {
		return err
	}

	// 更新 subject.count
	result = db.
		WithContext(ctx).
		Model(subject).
		Where("id = ?", subject.Id).
		Updates(
			map[string]interface{}{
				"count":      subject.Count,
				"root_count": subject.RootCount,
				"all_count":  subject.AllCount,
			})
	if err = result.Error; err != nil {
		return err
	}

	// 更新 root.count
	if root != nil {
		result = db.
			WithContext(ctx).
			Model(root).
			Where("id = ?", root.Id).
			Updates(
				map[string]interface{}{
					"count":       root.Count,
					"reply_count": root.ReplyCount,
				})
		if err = result.Error; err != nil {
			return err
		}
	}

	// TODO 2021/9/9 14:58 mysql开启binlog,写入redis cache保证最终一致
	// 以下错误都可忽略
	// cache subject
	if err = r.setSubjectCache(ctx, subject); err != nil {
		log.Error(err)
	}
	// cache comment
	if err = r.setCommentIndexCache(ctx, &index); err != nil {
		log.Error(err)
	}
	if err = r.setCommentContentCache(ctx, &content); err != nil {
		log.Error(err)
	}
	// cache root
	if root != nil && root.ReplyCount == 1 {
		root.Replies = append(root.Replies, &index)
		if err = r.setCommentIndexCache(ctx, root); err != nil {
			log.Error(err)
		}
	}
	// comment sort
	redis := r.data.redis.Get()
	defer redis.Close()
	key := fmt.Sprintf(_commentSortCacheKey, index.ObjId, index.ObjType)
	if reply, err := redis.Do("expire", key, _commentSortCacheTtl); err == nil {
		if i, ok := reply.(int64); ok && i == 1 {
			_, err = redis.Do("zadd", key, index.CreateAt.Unix(), index.Id)
			if err != nil {
				log.Error(err)
			}
		}
	} else {
		log.Error(err)
	}

	return nil
}

func (r *commentRepo) Delete(ctx context.Context, id int64) (err error) {
	// delete database comment
	// delete cache
	// update database parent.Count
	// update cache parent

	var (
		log   = r.log
		db    = r.data.db // todo TX
		redis = r.data.redis.Get()
		subject *CommentSubject
		root *CommentIndex
	)
	defer redis.Close()

	var index CommentIndex
	result := db.
		WithContext(ctx).
		Where("id = ?", id).
		First(&index)
	if err = result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if index.State == StateDelete {
		return nil
	}

	result = db.
		WithContext(ctx).
		Where("id = ?", id).
		Where("state != ?", StateDelete).
		Update("state", StateDelete)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}

	// subject
	result = db.
		WithContext(ctx).
		Where("obj_id = ?", index.ObjId).
		Where("obj_type = ?", index.ObjType).
		Update("count", gorm.Expr("count-1"))

	// root.count
	if index.Root > 0 {
		result = db.
			WithContext(ctx).
			Where("id = ?", index.Root).
			Update("count", gorm.Expr("count-1"))
		if result.Error != nil {
			return result.Error
		}
	}

	// del cache
	indexKey := fmt.Sprintf(_commentIndexCacheKey, id)
	if _, err := redis.Do("del", indexKey); err != nil {
		log.Error(err)
	}
	contentKey := fmt.Sprintf(_commentContentCacheKey, id)
	if _, err := redis.Do("del", contentKey); err != nil {
		log.Error(err)
	}

	// del cache sort
	sortKey := fmt.Sprintf(_commentSortCacheKey, index.ObjId, index.ObjType)
	if _, err := redis.Do("zrem", sortKey, index.Id); err != nil {
		log.Error(err)
	}

	subjectKey := fmt.Sprintf(_commentSubjectCacheKey, index.ObjId, index.ObjType)
	if reply, err := redis.Do("get", subjectKey); err == nil && reply != nil {
		if buf, ok := reply.([]byte); ok {
			if err = json.Unmarshal(buf, subject); err == nil {
				subject.Count--
				buf , err = json.Marshal(subject)
				if err == nil {
					_, err = redis.Do("setex", subjectKey, _commentSubjectCacheTtl, buf)
				} else {
					log.Error(err)
				}
			} else {
				log.Error(err)
			}
		}
	} else if err != nil {
		log.Error(err)
	}

	if index.Root > 0{
		rootKey := fmt.Sprintf(_commentIndexCacheKey, index.Root)
		if reply, err := redis.Do("get", rootKey); err == nil && reply != nil {
			if buf, ok := reply.([]byte); ok {
				if err = json.Unmarshal(buf, root); err == nil {
					root.Count--
					buf , err = json.Marshal(root)
					if err == nil {
						_, err = redis.Do("setex", rootKey, _commentIndexCacheTtl, buf)
					} else {
						log.Error(err)
					}
				} else {
					log.Error(err)
				}
			}
		} else if err != nil {
			log.Error(err)
		}

	}

	return nil
}
