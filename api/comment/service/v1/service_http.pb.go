// Code generated by protoc-gen-go-http. DO NOT EDIT.

package v1

import (
	context "context"
	http1 "github.com/go-kratos/kratos/v2/transport/http"
	binding "github.com/go-kratos/kratos/v2/transport/http/binding"
	mux "github.com/gorilla/mux"
	http "net/http"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(http.Request)
var _ = new(context.Context)
var _ = binding.MapProto
var _ = mux.NewRouter

const _ = http1.SupportPackageIsVersion1

type CommentServiceHandler interface {
	CreateComment(context.Context, *CreateCommentReq) (*CreateCommentReply, error)

	CreateSubject(context.Context, *CreateSubjectReq) (*CreateSubjectReply, error)

	DeleteComment(context.Context, *DeleteCommentReq) (*DeleteCommentReply, error)

	ListComment(context.Context, *ListCommentReq) (*ListCommentReply, error)

	ListReply(context.Context, *ListReplyReq) (*ListReplyReply, error)
}

func NewCommentServiceHandler(srv CommentServiceHandler, opts ...http1.HandleOption) http.Handler {
	h := http1.DefaultHandleOptions()
	for _, o := range opts {
		o(&h)
	}
	r := mux.NewRouter()

	r.HandleFunc("/comment.service.v1.CommentService/CreateSubject", func(w http.ResponseWriter, r *http.Request) {
		var in CreateSubjectReq
		if err := h.Decode(r, &in); err != nil {
			h.Error(w, r, err)
			return
		}

		next := func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.CreateSubject(ctx, req.(*CreateSubjectReq))
		}
		if h.Middleware != nil {
			next = h.Middleware(next)
		}
		out, err := next(r.Context(), &in)
		if err != nil {
			h.Error(w, r, err)
			return
		}
		reply := out.(*CreateSubjectReply)
		if err := h.Encode(w, r, reply); err != nil {
			h.Error(w, r, err)
		}
	}).Methods("POST")

	r.HandleFunc("/comment.service.v1.CommentService/CreateComment", func(w http.ResponseWriter, r *http.Request) {
		var in CreateCommentReq
		if err := h.Decode(r, &in); err != nil {
			h.Error(w, r, err)
			return
		}

		next := func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.CreateComment(ctx, req.(*CreateCommentReq))
		}
		if h.Middleware != nil {
			next = h.Middleware(next)
		}
		out, err := next(r.Context(), &in)
		if err != nil {
			h.Error(w, r, err)
			return
		}
		reply := out.(*CreateCommentReply)
		if err := h.Encode(w, r, reply); err != nil {
			h.Error(w, r, err)
		}
	}).Methods("POST")

	r.HandleFunc("/comment.service.v1.CommentService/DeleteComment", func(w http.ResponseWriter, r *http.Request) {
		var in DeleteCommentReq
		if err := h.Decode(r, &in); err != nil {
			h.Error(w, r, err)
			return
		}

		next := func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteComment(ctx, req.(*DeleteCommentReq))
		}
		if h.Middleware != nil {
			next = h.Middleware(next)
		}
		out, err := next(r.Context(), &in)
		if err != nil {
			h.Error(w, r, err)
			return
		}
		reply := out.(*DeleteCommentReply)
		if err := h.Encode(w, r, reply); err != nil {
			h.Error(w, r, err)
		}
	}).Methods("POST")

	r.HandleFunc("/comment.service.v1.CommentService/ListComment", func(w http.ResponseWriter, r *http.Request) {
		var in ListCommentReq
		if err := h.Decode(r, &in); err != nil {
			h.Error(w, r, err)
			return
		}

		next := func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.ListComment(ctx, req.(*ListCommentReq))
		}
		if h.Middleware != nil {
			next = h.Middleware(next)
		}
		out, err := next(r.Context(), &in)
		if err != nil {
			h.Error(w, r, err)
			return
		}
		reply := out.(*ListCommentReply)
		if err := h.Encode(w, r, reply); err != nil {
			h.Error(w, r, err)
		}
	}).Methods("POST")

	r.HandleFunc("/comment.service.v1.CommentService/ListReply", func(w http.ResponseWriter, r *http.Request) {
		var in ListReplyReq
		if err := h.Decode(r, &in); err != nil {
			h.Error(w, r, err)
			return
		}

		next := func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.ListReply(ctx, req.(*ListReplyReq))
		}
		if h.Middleware != nil {
			next = h.Middleware(next)
		}
		out, err := next(r.Context(), &in)
		if err != nil {
			h.Error(w, r, err)
			return
		}
		reply := out.(*ListReplyReply)
		if err := h.Encode(w, r, reply); err != nil {
			h.Error(w, r, err)
		}
	}).Methods("POST")

	return r
}

type CommentServiceHTTPClient interface {
	CreateComment(ctx context.Context, req *CreateCommentReq, opts ...http1.CallOption) (rsp *CreateCommentReply, err error)

	CreateSubject(ctx context.Context, req *CreateSubjectReq, opts ...http1.CallOption) (rsp *CreateSubjectReply, err error)

	DeleteComment(ctx context.Context, req *DeleteCommentReq, opts ...http1.CallOption) (rsp *DeleteCommentReply, err error)

	ListComment(ctx context.Context, req *ListCommentReq, opts ...http1.CallOption) (rsp *ListCommentReply, err error)

	ListReply(ctx context.Context, req *ListReplyReq, opts ...http1.CallOption) (rsp *ListReplyReply, err error)
}

type CommentServiceHTTPClientImpl struct {
	cc *http1.Client
}

func NewCommentServiceHTTPClient(client *http1.Client) CommentServiceHTTPClient {
	return &CommentServiceHTTPClientImpl{client}
}

func (c *CommentServiceHTTPClientImpl) CreateComment(ctx context.Context, in *CreateCommentReq, opts ...http1.CallOption) (out *CreateCommentReply, err error) {
	path := binding.EncodePath("POST", "/comment.service.v1.CommentService/CreateComment", in)
	out = &CreateCommentReply{}

	err = c.cc.Invoke(ctx, path, nil, &out, http1.Method("POST"), http1.PathPattern("/comment.service.v1.CommentService/CreateComment"))

	return
}

func (c *CommentServiceHTTPClientImpl) CreateSubject(ctx context.Context, in *CreateSubjectReq, opts ...http1.CallOption) (out *CreateSubjectReply, err error) {
	path := binding.EncodePath("POST", "/comment.service.v1.CommentService/CreateSubject", in)
	out = &CreateSubjectReply{}

	err = c.cc.Invoke(ctx, path, nil, &out, http1.Method("POST"), http1.PathPattern("/comment.service.v1.CommentService/CreateSubject"))

	return
}

func (c *CommentServiceHTTPClientImpl) DeleteComment(ctx context.Context, in *DeleteCommentReq, opts ...http1.CallOption) (out *DeleteCommentReply, err error) {
	path := binding.EncodePath("POST", "/comment.service.v1.CommentService/DeleteComment", in)
	out = &DeleteCommentReply{}

	err = c.cc.Invoke(ctx, path, nil, &out, http1.Method("POST"), http1.PathPattern("/comment.service.v1.CommentService/DeleteComment"))

	return
}

func (c *CommentServiceHTTPClientImpl) ListComment(ctx context.Context, in *ListCommentReq, opts ...http1.CallOption) (out *ListCommentReply, err error) {
	path := binding.EncodePath("POST", "/comment.service.v1.CommentService/ListComment", in)
	out = &ListCommentReply{}

	err = c.cc.Invoke(ctx, path, nil, &out, http1.Method("POST"), http1.PathPattern("/comment.service.v1.CommentService/ListComment"))

	return
}

func (c *CommentServiceHTTPClientImpl) ListReply(ctx context.Context, in *ListReplyReq, opts ...http1.CallOption) (out *ListReplyReply, err error) {
	path := binding.EncodePath("POST", "/comment.service.v1.CommentService/ListReply", in)
	out = &ListReplyReply{}

	err = c.cc.Invoke(ctx, path, nil, &out, http1.Method("POST"), http1.PathPattern("/comment.service.v1.CommentService/ListReply"))

	return
}
