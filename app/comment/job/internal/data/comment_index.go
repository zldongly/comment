package data

import "time"

const (
	_commentIndexCacheKey           = `comment_index_cache:%d:%d`          // obj_id, obj_type
	_commentIndexLocalCacheKey      = `comment_index_cache:%d:%d:%d:%d`    // obj_id, obj_type, pageNo, pageSize
	_commentReplyIndexCacheKey      = `comment_reply_index_cache:%d`       // root_id
	_commentReplyIndexLocalCacheKey = `comment_reply_index_cache:%d:%d:%d` // root_id, pageNo, pageSize
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
