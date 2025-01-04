package braza

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethoDomingues/c3po"
)

var (
	htmlTemplates sync.Map
)

func NewResponse(wr http.ResponseWriter, ctx *Ctx) *Response {
	return &Response{
		Buffer:     bytes.NewBufferString(""),
		raw:        wr,
		ctx:        ctx,
		Headers:    Header{},
		StatusCode: 200,
	}
}

type Response struct {
	*bytes.Buffer
	StatusCode int
	Headers    Header
	ctx        *Ctx
	raw        http.ResponseWriter
}

func (r Response) Header() http.Header            { return http.Header(r.Headers) }
func (r Response) Write(b []byte) (int, error)    { return r.Buffer.Write(b) }
func (r Response) WriteHeader(statusCode int)     { r.StatusCode = statusCode }
func (r *Response) SetCookie(cookie *http.Cookie) { r.Headers.SetCookie(cookie) }
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.raw.(http.Hijacker).Hijack()
}

func (r *Response) _write(v any) {
	if reader, ok := v.(io.Reader); ok {
		io.Copy(r, reader)
	} else {
		r.WriteString(fmt.Sprint(v))
	}
}

func (r *Response) parseHeaders() {
	ctx := r.ctx
	method := ctx.Request.Method
	routerCors := ctx.MatchInfo.Router.Cors
	if routerCors != nil {
		routerCors.parse(r.Headers, ctx.Request)
	}
	routeCors := ctx.MatchInfo.Route.Cors

	h := r.Headers
	if routeCors != nil {
		routeCors.parse(r.Headers, ctx.Request)
	}
	if routerCors != nil || routeCors != nil {
		if _, ok := h["Access-Control-Request-Method"]; !ok {
			h.Set("Access-Control-Request-Method", method)
		}
	}
	if method == "OPTIONS" {
		if _, ok := h["Access-Control-Allow-Origin"]; !ok {
			h.Set("Access-Control-Allow-Origin", ctx.App.Servername)
		}
		if _, ok := h["Access-Control-Allow-Headers"]; !ok {
			h.Set("Access-Control-Allow-Headers", "content-type")
		}
		if _, ok := h["Access-Control-Expose-Headers"]; !ok {
			h.Set("Access-Control-Expose-Headers", "")
		}
		if _, ok := h["Access-Control-Allow-Credentials"]; !ok {
			h.Set("Access-Control-Allow-Credentials", "false")
		}
	} else if ctx.Request.Method == "HEAD" {
		r.Reset()
	}
}

func (r *Response) textCode(code int) {
	statusText := http.StatusText(code)
	if statusText == "" {
		panic(fmt.Errorf("unknown status code:'%d'", code))
	}
	r.ctx.backCtx = context.WithValue(r.ctx.backCtx, abortCode(1), code)
	panic(ErrHttpAbort)
}

// Redirect to Following URL
func (r *Response) Redirect(url string) {
	r.Reset()
	r.Headers.Set("Location", url)
	r.StatusCode = 307

	r.Headers.Set("Content-Type", "text/html; charset=utf-8")
	r.WriteString("<a href=\"" + c3po.HtmlEscape(url) + "\"> Manual Redirect </a>.\n")
	panic(ErrHttpAbort)
}

func (r *Response) JSON(body any, code int) {
	r.Reset()
	r.StatusCode = code
	r.Headers.Set("Content-Type", "application/json")

	switch body := body.(type) {
	case string:
		r.WriteString(body)
		panic(ErrHttpAbort)
	case error:
		r.WriteString(body.Error())
		panic(ErrHttpAbort)
	}

	if b, ok := body.(fmt.Stringer); ok {
		r.WriteString(b.String())
		panic(ErrHttpAbort)
	}

	j, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	r.Write(j)
	panic(ErrHttpAbort)
}

func (r *Response) TEXT(body any, code int) {
	r.Reset()
	r.StatusCode = code
	r.Headers.Set("Content-Type", "text/plain")
	r._write(body)
	panic(ErrHttpAbort)
}

func (r *Response) HTML(body any, code int) {
	r.Reset()
	r.StatusCode = code
	r.Headers.Set("Content-Type", "text/html")
	r._write(body)
	panic(ErrHttpAbort)
}

func (r *Response) Abort(code int)       { r.textCode(code) }
func (r *Response) Close()               { panic(ErrHttpAbort) }
func (r *Response) Ok()                  { r.textCode(200) }
func (r *Response) Created()             { r.textCode(201) }
func (r *Response) NoContent()           { r.textCode(204) }
func (r *Response) BadRequest()          { r.textCode(400) }
func (r *Response) Unauthorized()        { r.textCode(401) }
func (r *Response) Forbidden()           { r.textCode(403) }
func (r *Response) NotFound()            { r.textCode(404) }
func (r *Response) MethodNotAllowed()    { r.textCode(405) }
func (r *Response) ImATaerpot()          { r.textCode(418) }
func (r *Response) InternalServerError() { r.textCode(500) }

func (r *Response) RenderTemplate(tmpl string, data ...any) {
	var (
		t       *template.Template
		_t      any
		ok      bool
		value   any
		app     = r.ctx.App
		lenFile int
	)

	if _t, ok = htmlTemplates.Load(app.Name + ":" + tmpl); !ok || (app.Env == "development" && !app.DisableTemplateReloader) {
		pa := filepath.Join(app.TemplateFolder, tmpl)
		_, err := os.Stat(pa)
		if err != nil {
			if app.Env == "development" {
				r.TEXT(err, 404)
			}
			r.NotFound()
		}
		f, err := os.ReadFile(pa)
		r.CheckErr(err)

		t, err = template.New(tmpl).
			Funcs(r.ctx.App.TemplateFuncs).
			Parse(string(f))
		r.CheckErr(err)
		lenFile = len(f)
		htmlTemplates.Store(r.ctx.App.Name+":"+tmpl, t)
	} else {
		t = _t.(*template.Template)
	}
	if len(data) == 1 {
		value = data[0]
	}
	t.Execute(r, value)
	if r.Buffer.Len() == 0 && lenFile > 0 {
		r.TEXT("ouve um erro durante o parse do html, por favor, verificar o arquivo", 500)
	}
	r.Close()
}

// if err != nil, return a 500 Intenal Server Error
func (r *Response) CheckErr(err error) {
	if err != nil {
		l.err.Println(err)
		if r.ctx.App.Env == "development" {
			r.TEXT(err, 500)
		}
		r.InternalServerError()
	}
}

// Serve File
func (r *Response) ServeFile(pathToFile string) {
	ctx := r.ctx
	f, err := os.Open(pathToFile)
	if err == nil {
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
	}
	if ctx.App.Env == "development" {
		ctx.TEXT(err, 404)
	}
	ctx.Response.NotFound()
}

func req500(ctx *Ctx) {
	defer l.LogRequest(ctx)
	if err := recover(); err != nil {
		statusText := "500 Internal Server Error"
		l.Error(err)
		ctx.raw.WriteHeader(500)
		fmt.Fprint(ctx.raw, statusText)
	}
}

func reqOK(ctx *Ctx) {
	mi := ctx.MatchInfo
	rsp := ctx.Response
	if mi.Match {
		if ctx.Session.changed {
			rsp.SetCookie(ctx.Session.save(ctx))
		}
		rsp.parseHeaders()
		rsp.Headers.Save(rsp.raw)
	}
	rsp.raw.WriteHeader(rsp.StatusCode)
	fmt.Fprint(rsp.raw, rsp.String())
}
