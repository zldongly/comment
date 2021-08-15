package data

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/coocood/freecache"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/zldongly/comment/app/comment/service/internal/biz"
	"gorm.io/gorm"

	"context"
	"time"
)

var _ biz.SubjectRepo = (*subjectRepo)(nil)

const (
	_commentSubjectCacheKey = `comment_subject_cache:%d:%d`
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

func (s *CommentSubject) ToBiz() *biz.Subject {
	return &biz.Subject{
		Id:        s.Id,
		ObjId:     s.ObjId,
		ObjType:   s.ObjType,
		MemberId:  s.MemberId,
		Count:     s.Count,
		RootCount: s.RootCount,
		AllCount:  s.AllCount,
		Status:    s.Status,
	}
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

func (r *subjectRepo) Create(ctx context.Context, b *biz.Subject) error {
	// kafka
	return nil
}

func (r *subjectRepo) Get(ctx context.Context, objId int64, objType int32) (b *biz.Subject, err error) {
	var (
		log = r.log
		s   = CommentSubject{}
		key = fmt.Sprintf(_commentSubjectCacheKey, objId, objType)

		ok  bool
		buf []byte
	)
	_ = key
	//TODO: 2021/8/15
	// single flight

	// 读本地缓存
	buf, err = r.data.cache.Get([]byte(key))
	if err != nil {
		if !errors.Is(err, freecache.ErrNotFound) {
			log.Error(err)
		}
	} else { // 命中本地缓存
		err = json.Unmarshal(buf, &s)
		if err != nil {
			log.Error(err)
		} else {
			return s.ToBiz(), nil
		}
	}

	// 读redis
	var reply interface{}
	reply, err = r.data.redis.Get().Do("Get", key)
	if err != nil {
		log.Error(err)
	}
	if reply != nil { // 命中缓存
		if buf, ok = reply.([]byte); ok {
			err = json.Unmarshal(buf, &s)
			if err != nil {
				log.Error(err)
			} else {
				return s.ToBiz(), nil
			}
		}
	}

	// mysql
	result := r.data.db.
		WithContext(ctx).
		Where("obj_id = ?", objId).
		Where("obj_type = ?", objType).
		First(&s)

	err = result.Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = errors.NotFound("comment.subject", "comment_subject not found")
	}

	if err != nil {
		return b, err
	}


	buf, err = json.Marshal(s)
	if err != nil {
		log.Error(err)
		return s.ToBiz(), nil
	}

	// 此处考虑errgroup并行处理
	// 填入本地缓存
	if err = r.data.cache.Set([]byte(key), buf, 8); err != nil {
		log.Error(err)
	}

	// kafka
	msg := &sarama.ProducerMessage{
		Topic: "subject", // 放到 job.api 中
		Key:   sarama.StringEncoder(fmt.Sprintf("%d+%d", objId, objType)),
		Value: sarama.ByteEncoder(buf),
	}
	_, _, err = r.data.kafka.SendMessage(msg)
	if err != nil {
		log.Error(err)
	}

	return s.ToBiz(), nil
}
