
package service

import(
	"context"
	pb "github.com/zldongly/comment/api/comment/service/v1"

)

type CommentService struct {
	pb.UnimplementedCommentServiceServer
}

func NewCommentServiceService() *CommentService {
	return &CommentService{}
}

func (s *CommentService) CreateSubject(ctx context.Context, req *pb.CreateSubjectReq) (*pb.CreateSubjectReply, error) {
	return &pb.CreateSubjectReply{}, nil
}
func (s *CommentService) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.CreateCommentReply, error) {
	return &pb.CreateCommentReply{}, nil
}
func (s *CommentService) DeleteComment(ctx context.Context, req *pb.DeleteCommentReq) (*pb.DeleteCommentReply, error) {
	return &pb.DeleteCommentReply{}, nil
}
func (s *CommentService) ListComment(ctx context.Context, req *pb.ListCommentReq) (*pb.ListCommentReply, error) {
	return &pb.ListCommentReply{}, nil
}
func (s *CommentService) ListReply(ctx context.Context, req *pb.ListReplyReq) (*pb.ListReplyReply, error) {
	return &pb.ListReplyReply{}, nil
}
