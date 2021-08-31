package biz

import "context"

type Subject struct {
	Id        int64
	ObjId     int64
	ObjType   int32
	MemberId  int64
	Count     int32 // 历史根评论数量，删除评论时不减少
	RootCount int32 // 根评论数量
	AllCount  int32 // 评论 + 评论的回复
	Status    int8
}

type SubjectRepo interface {
	Create(ctx context.Context, s *Subject) error
	Cache(ctx context.Context, objId int64, objType int32) error
}
