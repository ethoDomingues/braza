package braza

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func serveFileHandler(ctx *Ctx) {
	rq := ctx.Request

	urlFilePath := rq.Args["filepath"]
	pathToFile := filepath.Join(ctx.App.StaticFolder, urlFilePath)
	if f, err := os.Open(pathToFile); err == nil {
		_, file := filepath.Split(pathToFile)
		defer f.Close()
		if fStat, err := f.Stat(); err != nil || fStat.IsDir() {
			ctx.NotFound()
		}
		ctx.ReadFrom(f)
		ctype := mime.TypeByExtension(filepath.Ext(file))
		if ctype == "application/octet-stream" {
			ctype = http.DetectContentType(ctx.Bytes())
		}
		ctx.Headers.Set("Content-Type", ctype)
		ctx.Close()
	} else {
		if ctx.App.Env == "development" {
			ctx.TEXT(err, 404)
		}
		ctx.Response.NotFound()
	}
}

func optionsHandler(ctx *Ctx) {
	rsp := ctx.Response
	mi := ctx.MatchInfo
	rsp.StatusCode = 200
	strMeths := strings.Join(mi.Route.Cors.AllowMethods, ", ")
	if rsp.Headers.Get("Access-Control-Allow-Methods") == "" {
		rsp.Headers.Set("Access-Control-Allow-Methods", strMeths)
	}
	rsp.parseHeaders()
	rsp.Headers.Save(rsp.raw)
}
