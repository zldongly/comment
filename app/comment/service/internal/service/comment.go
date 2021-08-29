package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	pb "github.com/zldongly/comment/api/comment/service/v1"
	"github.com/zldongly/comment/app/comment/service/internal/biz"
)

var ProviderSet = wire.NewSet(NewCommentServiceService)

type CommentService struct {
	pb.UnimplementedCommentServiceServer

	cc  *biz.CommentUseCase
	log *log.Helper
}

func NewCommentServiceService(cc *biz.CommentUseCase, logger log.Logger) *CommentService {
	return &CommentService{
		cc:  cc,
		log: log.NewHelper(log.With(logger, "module", "comment-service/service")),
	}
}

func (s *CommentService) CreateSubject(ctx context.Context, req *pb.CreateSubjectReq) (*pb.CreateSubjectReply, error) {
	subject := &biz.Subject{
		ObjId:    req.ObjId,
		ObjType:  req.ObjType,
		MemberId: req.MemberId,
	}
	err := s.cc.CreateSubject(ctx, subject)
	return &pb.CreateSubjectReply{}, err
}

func (s *CommentService) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.CreateCommentReply, error) {
	comment := &biz.Comment{
		ObjId:       req.ObjId,
		ObjType:     req.ObjType,
		MemberId:    req.MemberId,
		Root:        req.Root,
		Parent:      req.Parent,
		AtMemberIds: req.AtMemberIds,
		Message:     req.Message,
		Meta:        req.Meta,
		Ip:          req.Ip,
		Platform:    int8(req.Platform),
		Device:      req.Device,
	}
	err := s.cc.CreateComment(ctx, comment)
	return &pb.CreateCommentReply{}, err
}

func (s *CommentService) DeleteComment(ctx context.Context, req *pb.DeleteCommentReq) (*pb.DeleteCommentReply, error) {
	err := s.cc.DeleteComment(ctx, req.CommentId)
	return &pb.DeleteCommentReply{}, err
}

func (s *CommentService) ListComment(ctx context.Context, req *pb.ListCommentReq) (*pb.ListCommentReply, error) {
	subject, comments, err := s.cc.ListComment(ctx, req.ObjType, req.ObjId, req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}

	result := &pb.ListCommentReply{
		Total: subject.Count,
		List:  make([]*pb.ListCommentReply_Comment, 0, len(comments)),
	}
	for _, c := range comments {
		comment := &pb.ListCommentReply_Comment{
			CommentId:   c.Id,
			MemberId:    c.MemberId,
			Floor:       c.Floor,
			Like:        c.Like,
			Hate:        c.Hate,
			AtMemberIds: c.AtMemberIds,
			Message:     c.Message,
			Meta:        c.Meta,
			CreateTime:  c.CreateAt.Unix(),
			Count:       c.Count,
			Replies:     make([]*pb.Reply, 0, len(c.Replies)),
		}
		for _, r := range c.Replies {
			comment.Replies = append(comment.Replies, replyBizToApi(r))
		}

		result.List = append(result.List, comment)
	}

	return result, nil
}

func (s *CommentService) ListReply(ctx context.Context, req *pb.ListReplyReq) (*pb.ListReplyReply, error) {
	replies, err := s.cc.ListReply(ctx, req.CommentId, req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}

	result := &pb.ListReplyReply{
		Replies: make([]*pb.Reply, 0, len(replies)),
	}
	for _, r := range replies {
		result.Replies = append(result.Replies, replyBizToApi(r))
	}

	return result, nil
}

func replyBizToApi(r *biz.Comment) *pb.Reply {
	return &pb.Reply{
		CommentId:      r.Id,
		MemberId:       r.MemberId,
		ParentId:       r.Parent,
		ParentMemberId: r.ParentMemberId,
		Floor:          r.Floor,
		Like:           r.Like,
		Hate:           r.Hate,
		AtMemberIds:    r.AtMemberIds,
		Message:        r.Message,
		CreateTime:     r.CreateAt.Unix(),
	}
}
