package data

import (
	"time"
)

const (
	_commentContentIndexKey = `comment_index_cache:%d:%d`
)

type CommentIndex struct {
	Id       int64
	ObjId    int64
	ObjType  int32
	MemberId int64

	Root   int64 // 根评论ID
	Patent int64 // 父评论
	Floor  int32 // 楼层
	Count  int32 // 回复数量

	Like int32 // 点赞
	Hate int32 // 点睬

	State int8  // 状态
	Attrs int32 // 属性，bit 置顶等

	CreateAt time.Time
	UpdateAt time.Time
}

func (*CommentIndex) TableName() string {
	return "comment_index"
}
