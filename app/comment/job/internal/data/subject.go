package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zldongly/comment/app/comment/job/internal/biz"
	"gorm.io/gorm"
	"time"
)

var _ biz.SubjectRepo = (*subjectRepo)(nil)

const (
	_commentSubjectCacheKey = `comment_subject_cache:%d:%d` // obj_id, obj_type
	_commentSubjectCacheTtl = 8 * 60 * 60                   // 8h
)

type CommentSubject struct {
	Id        int64
	ObjId     int64
	ObjType   int32
	MemberId  int64
	Count     int32 // 历史根评论数量，删除评论时不减少
	RootCount int32 // 根评论数量
	AllCount  int32 // 评论 + 评论的回复
	Status    int8

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (*CommentSubject) TableName() string {
	// TODO: 2021/8/15 分表
	return "comment_subject"
}

type subjectRepo struct {
	data *Data
	log  *log.Helper
}

func NewSubjectRepo(data *Data, logger log.Logger) biz.SubjectRepo {
	return &subjectRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/comment_subject")),
	}
}

func (r *subjectRepo) Create(ctx context.Context, s *biz.Subject) error {
	var (
		log = r.log
	)

	subject := &CommentSubject{
		ObjId:    s.ObjId,
		ObjType:  s.ObjType,
		MemberId: s.MemberId,
	}

	// database
	result := r.data.db.
		WithContext(ctx).
		Create(subject)
	if result.Error != nil {
		return result.Error
	}

	redis := r.data.redis.Get()
	defer redis.Close()

	// redis cache
	key := fmt.Sprintf(_commentSubjectCacheKey, s.ObjId, s.ObjType)
	if buf, err := json.Marshal(subject); err == nil {
		if _, err = redis.Do("setex", key, _commentSubjectCacheTtl, buf); err != nil {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	return nil
}

func (r *subjectRepo) Cache(ctx context.Context, objId int64, objType int32) error {
	var (
		s *CommentSubject
	)

	result := r.data.db.
		WithContext(ctx).
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		First(s)
	if result.Error != nil {
		return result.Error
	}

	redis := r.data.redis.Get()
	defer redis.Close()

	key := fmt.Sprintf(_commentSubjectCacheKey, objId, objType)
	if buf, err := json.Marshal(s); err == nil {
		if _, err = redis.Do("setex", key, _commentSubjectCacheTtl, buf); err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func (r *commentRepo) getSubject(ctx context.Context, objId int64, objType int32) (*CommentSubject, error) {
	var (
		log     = r.log
		redis   = r.data.redis.Get()
		key     = fmt.Sprintf(_commentSubjectCacheKey, objId, objType)
		subject CommentSubject
	)
	defer redis.Close()

	// redis
	if reply, err := redis.Do("get", key); err == nil {
		if buf, ok := reply.([]byte); ok {
			if err = json.Unmarshal(buf, &subject); err == nil {
				return &subject, nil
			} else {
				return nil, err
			}
		}
	} else {
		log.Error(err)
	}

	// database
	result := r.data.db.
		WithContext(ctx).
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		First(&subject)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) { // 不存在
			return nil, errors.NotFound("comment_subject", "not found")
		}
		return nil, result.Error
	}

	return &subject, nil
}

func (r *commentRepo) setSubjectCache(ctx context.Context, subject *CommentSubject) error {
	var (
		redis = r.data.redis.Get()
		key   = fmt.Sprintf(_commentSubjectCacheKey, subject.ObjId, subject.ObjType)
	)
	defer redis.Close()

	buf, err := json.Marshal(subject)
	if err != nil {
		return err
	}
	_, err = redis.Do("setex", key, _commentSubjectCacheTtl, buf)
	if err != nil {
		return err
	}

	return nil
}
