package main

import "github.com/ethoDomingues/braza"

type Post struct {
	ID    string
	Text  string
	Likes int
}

var db = map[string]*Post{}

func main() {
	cfg := &braza.Config{
		SecretKey:      "shiiii!!... it's is secret",
		Servername:     "localhost:5000",
		TemplateFolder: "./",
	}
	app := braza.NewApp(cfg)
	app.BeforeRequest = beforeReq // exec befor each request

	api := getApiRouter()
	app.Mount(api)

	app.GET("", func(ctx *braza.Ctx) {
		ctx.RenderTemplate("index.html")
	})
	app.Listen()
}

func beforeReq(ctx *braza.Ctx) {
	ctx.Global["db"] = db
}
