package main

import (
	"fmt"

	"github.com/ethoDomingues/braza"
)

var apiRoutes = []*braza.Route{
	// posts => /posts
	{
		Name: "posts",
		Url:  "/posts",
		MapCtrl: braza.MapCtrl{
			"GET": &braza.Meth{
				Func: getManyPost,
			},
			"POST": &braza.Meth{
				Func:   postPost,
				Schema: &PostPostSchema{},
			},
		},
	},
	// post => /posts/{postID}
	{
		Name: "post",
		Url:  "/posts/{postID:int}",
		MapCtrl: braza.MapCtrl{
			"GET": &braza.Meth{
				Func:   getPost,
				Schema: &GetPostSchema{},
			},
			"PUT": &braza.Meth{
				Func:   putPost,
				Schema: PutPostSchema{},
			},
			"DELETE": &braza.Meth{
				Func:   deletePost,
				Schema: &DeletePostSchema{},
			},
		},
	},
}

func getApiRouter() *braza.Router {
	api := &braza.Router{
		Name:      "api",
		Prefix:    "/v1",
		Subdomain: "api",
		Cors: &braza.Cors{
			AllowOrigins: []string{"*"},
		},
	}

	api.AddRoute(apiRoutes...)

	return api
}

type GetPostSchema struct {
	PostID string `braza:"in=args,name=postID"`
}

func getPost(ctx *braza.Ctx) {
	sch := ctx.Schema.(*GetPostSchema)
	db := ctx.Global["db"].(map[string]*Post)
	if post, ok := db[sch.PostID]; ok {
		ctx.JSON(post, 200)
	}
	ctx.NotFound()
}

func getManyPost(ctx *braza.Ctx) {
	db := ctx.Global["db"].(map[string]*Post)
	data := []any{}
	for _, p := range db {
		data = append(data, p)
	}
	ctx.JSON(data, 200)
}

type PostPostSchema struct {
	Text string
}

func postPost(ctx *braza.Ctx) {
	db := ctx.Global["db"].(map[string]*Post)
	sch := ctx.Schema.(*PostPostSchema)
	id := fmt.Sprint(len(db))
	post := &Post{
		ID:   id,
		Text: sch.Text,
	}
	db[id] = post
	ctx.JSON(post, 201)
}

type PutPostSchema struct {
	Text   string
	PostID string `braza:"in=args,name=postID"`
}

func putPost(ctx *braza.Ctx) {
	db := ctx.Global["db"].(map[string]*Post)
	sch := ctx.Schema.(PutPostSchema)
	id := sch.PostID
	if _, ok := db[id]; ok {
		post := &Post{
			ID:   id,
			Text: sch.Text,
		}
		db[id] = post
		ctx.JSON(post, 200)
	}
	ctx.NotFound()
}

type DeletePostSchema struct {
	PostID string `braza:"in=args,name=postID"`
}

func deletePost(ctx *braza.Ctx) {
	sch := ctx.Schema.(*DeletePostSchema)
	db := ctx.Global["db"].(map[string]*Post)
	if _, ok := db[sch.PostID]; !ok {
		delete(db, sch.PostID)
		ctx.NoContent()
	}
	ctx.NotFound()
}
