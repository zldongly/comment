package data

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/go-kratos/kratos/v2/log"
	job "github.com/zldongly/comment/api/comment/job/v1"
	"github.com/zldongly/comment/app/comment/service/internal/biz"
	"google.golang.org/protobuf/proto"
	"strconv"
)

const (
	_localCacheExpire = 5
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

func (r *commentRepo) CreateComment(ctx context.Context, comment *biz.Comment) error {
	k := &job.CreateCommentReq{
		ObjId:       comment.ObjId,
		ObjType:     comment.ObjType,
		MemberId:    comment.MemberId,
		Root:        comment.Root,
		Parent:      comment.Parent,
		AtMemberIds: comment.AtMemberIds,
		Message:     comment.Message,
		Meta:        comment.Meta,
		Ip:          comment.Ip,
		Platform:    int32(comment.Platform),
		Device:      comment.Device,
	}
	if buf, err := proto.Marshal(k); err == nil {
		msg := &sarama.ProducerMessage{
			Topic: job.TopicCreateComment,
			Key:   sarama.StringEncoder(fmt.Sprintf("%d+%d", k.ObjId, k.ObjType)),
			Value: sarama.ByteEncoder(buf),
		}
		if _, _, err = r.data.kafka.SendMessage(msg); err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func (r *commentRepo) DeleteComment(ctx context.Context, id int64) error {
	var (
		log = r.log
	)

	k := &job.DeleteCommentReq{
		CommentId: id,
	}
	if buf, err := proto.Marshal(k); err == nil {
		msg := &sarama.ProducerMessage{
			Topic: job.TopicDeleteComment,
			Key:   sarama.StringEncoder(strconv.FormatInt(id, 10)),
			Value: sarama.ByteEncoder(buf),
		}
		if _, _, err = r.data.kafka.SendMessage(msg); err != nil {
			log.Error(err)
		}
	} else {
		log.Error(err)
	}

	return nil
}
