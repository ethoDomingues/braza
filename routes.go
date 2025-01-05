package braza

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/ethoDomingues/c3po"
)

type Func func(ctx *Ctx)

func (f Func) String() string {
	return "func(ctx *braza.Ctx)"
}

/*
example:

	type User struct{
		Name string `braza:"name=name"`
		Email string `braza:"name=username,in=auth"`    // name need be 'username'
		Password string `braza:"name=password,in=auth"` // name need be 'password'
		ProfilePhoto *braza.File `braza:"name=img,in=files"`
		KeepConnected bool `braza:"name=keep,inquery"`
	}

	func AnyHandler(ctx *braza.Ctx){
		u, ok := ctx.Schema.(*User)
		...
	}
*/
type Schema any

type Meth struct {
	Func
	Schema
	SchemaFielder *c3po.Fielder
}

type MapCtrl map[string]*Meth

type Route struct {
	Url  string
	Name string

	Func        Func
	Cors        *Cors
	Schema      Schema
	MapCtrl     MapCtrl
	Methods     []string
	Middlewares []Func

	parsed      bool
	router      *Router
	urlRegex    []*regexp.Regexp
	hasSufix    bool
	isStatic    bool
	simpleUrl   string
	simpleName  string
	isUrlPrefix bool
}

func (r *Route) compileUrl() {
	uri := r.Url
	isPrefix := false
	r.hasSufix = strings.HasSuffix(r.simpleUrl, "/")
	if uri != "" && uri != "/" {
		uri = strings.TrimPrefix(strings.TrimSuffix(uri, "/"), "/")
		strs := strings.Split(uri, "/")

		for _, str := range strs {
			if str == "" {
				continue
			}
			if isPrefix {
				log.Panicf("Url Variable Invalid: '%s'", str)
			}
			if re.all.MatchString(str) {
				isPrefix = true
			}
			if re.dot2.MatchString(str) {
				str = re.dot2.ReplaceAllString(str, "/") // -> /../../home = /////home
			}
			if re.slash2.MatchString(str) {
				str = re.slash2.ReplaceAllString(str, "/") // -> /////home = /home
			}

			if re.isVar.MatchString(str) {
				str = re.str.ReplaceAllString(str, `(([\x00-\x7F]+)([^\\\/\s]+)|\d+)`) // This expression will search for non-ASCII values:
				str = re.digit.ReplaceAllString(str, `(\d+)`)
			}
			if !isPrefix {
				r.urlRegex = append(r.urlRegex, regexp.MustCompile(fmt.Sprintf("^%s$", str)))
			}
		}
	}
	if r.hasSufix {
		if r.Url != "/" {
			r.Url = r.Url + "/"
		}
	}
	r.isUrlPrefix = isPrefix
}

func (r *Route) compileMethods() {
	ctrl := MapCtrl{"OPTIONS": &Meth{}}

	// allow Route{URL:"/",Name:"index",Func:func()} with method default "GET"
	if len(r.MapCtrl) == 0 && len(r.Methods) == 0 {
		r.Methods = []string{"GET"}
	}

	for verb, m := range r.MapCtrl {
		v := strings.ToUpper(verb)
		if !reMethods.MatchString(v) {
			l.err.Fatalf("route '%s' has invalid Request Method: '%s'", r.Name, verb)
		}
		if m.Schema != nil {
			sch := c3po.ParseSchemaWithTag("braza", m.Schema)
			m.SchemaFielder = sch
		}
		ctrl[v] = m
	}

	r.MapCtrl = ctrl

	for _, verb := range r.Methods {
		v := strings.ToUpper(verb)
		if !reMethods.MatchString(v) {
			l.err.Fatalf("route '%s' has invalid Request Method: '%s'", r.Name, verb)
		}

		if _, ok := r.MapCtrl[v]; !ok {
			r.MapCtrl[v] = &Meth{
				Func: r.Func,
			}

			if r.Schema != nil {
				sch := c3po.ParseSchemaWithTag("braza", r.Schema)
				r.MapCtrl[v].SchemaFielder = sch
				r.MapCtrl[v].Schema = r.Schema
			}
		}
	}

	r.Methods = []string{}
	for verb := range r.MapCtrl {
		r.Methods = append(r.Methods, verb)
		if verb == "GET" {
			if _, ok := r.MapCtrl["HEAD"]; !ok {
				r.Methods = append(r.Methods, "HEAD")
			}
		}
	}

	if len(r.Methods) <= 1 {
		r.Methods = []string{"GET", "HEAD"}
		r.MapCtrl["GET"] = &Meth{
			Func: r.Func,
		}
	}
}

func (r *Route) parse() {
	if r.Func == nil && r.MapCtrl == nil {
		l.err.Fatalf("Route '%s' need a Func or MapCtrl\n", r.Name)
	}

	r.compileUrl()
	r.compileMethods()
	if r.Cors != nil {
		r.Cors.AllowMethods = r.Methods
	} else {
		r.Cors = &Cors{AllowMethods: r.Methods}
	}
}

func (r *Route) matchURL(ctx *Ctx, url string) bool {
	if url == r.Url {
		return true
	}
	// if url == /  and route.Url == ""
	if url == "/" && r.Url == "" {
		if !ctx.App.StrictSlash {
			return true
		} else {
			return false // poderia fazer um redirect, nas acho q ia dar b.o
		}
	}

	// if url == ""  and route.Url == /
	if url == "" && r.Url == "/" {
		if !ctx.App.StrictSlash {
			return true
		} else {
			return false // poderia fazer um redirect, nas acho q ia dar b.o²
		}
	}

	nurl := strings.TrimPrefix(url, "/")
	nurl = strings.TrimSuffix(nurl, "/")
	urlSplit := strings.Split(nurl, "/")

	lSplit := len(urlSplit)
	lRegex := len(r.urlRegex)

	if lSplit != lRegex {
		if !ctx.App.DisableStatic && r.isStatic {
			if strings.HasPrefix(ctx.Request.URL.Path, ctx.App.StaticUrlPath) {
				return true
			}
		}
		if r.isUrlPrefix {
			if lRegex < lSplit {
				for i, uRe := range r.urlRegex {
					str := urlSplit[i]
					if !uRe.MatchString(str) {
						return false
					}
				}
				return true
			}
		}
		return false
	}

	for i, uRe := range r.urlRegex {
		str := urlSplit[i]
		if !uRe.MatchString(str) {
			return false
		}
	}

	if ctx.App.StrictSlash {
		last := string(url[len(url)-1])
		if r.hasSufix && last == "/" {
			return true
		} else if !r.hasSufix && last != "/" {
			return true
		}
		return false
	}
	return true
}

func (r *Route) match(ctx *Ctx) bool {
	mi := ctx.MatchInfo
	rq := ctx.Request
	m := rq.Method
	url := rq.URL.Path

	if !r.matchURL(ctx, url) {
		return false
	}
	if m == "HEAD" {
		m = "GET"
	}

	if meth, ok := r.MapCtrl[m]; ok {
		mi.MethodNotAllowed = nil
		if meth.Func != nil {
			mi.Func = meth.Func
		}
		mi.Match = true
		mi.Route = r
		ctx.SchemaFielder = meth.SchemaFielder
		return true
	}

	mi.Route = nil
	mi.MethodNotAllowed = ErrorMethodMismatch
	return false
}

func (r *Route) mountURI(args ...string) string {
	var (
		params = map[string]string{}
	)

	c := len(args)
	for i := 0; i < c; i++ {
		if i%2 != 0 {
			continue
		}
		params[args[i]] = args[i+1]
	}

	// Pre Build
	var sUrl = strings.Split(r.Url, "/")
	var urlBuf strings.Builder

	// Build path
	for _, str := range sUrl {
		if re.isVar.MatchString(str) {
			fname := re.getVarName(str)
			value, ok := params[fname]
			if !ok {
				if !re.isVarOpt.MatchString(str) {
					panic(fmt.Errorf("Route '%s' needs parameter '%s' but not passed", r.Name, str))
				}
			} else {
				urlBuf.WriteString("/" + value)
				delete(params, fname)
			}
		} else {
			urlBuf.WriteString("/" + str)
		}
	}
	// Build Query
	var query strings.Builder
	if len(params) > 0 {
		urlBuf.WriteString("?")
		for k, v := range params {
			query.WriteString(k + "=" + v + "&")
		}
	}
	url := urlBuf.String()
	url = re.slash2.ReplaceAllString(url, "/")
	url = re.dot2.ReplaceAllString(url, ".")
	if len(params) > 0 {
		return url + strings.TrimSuffix(query.String(), "&")
	}
	return url
}

func (r *Route) GetRouter() *Router {
	if r.router == nil {
		panic(fmt.Errorf("unregistered route: '%s'", r.Name))
	}
	return r.router
}

/*

 */

func GET(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"GET"},
	}
}

func HEAD(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"HEAD"},
	}
}

func POST(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"POST"},
	}
}

func PUT(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"PUT"},
	}
}

func DELETE(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"DELETE"},
	}
}

func CONNECT(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"CONNECT"},
	}
}

func OPTIONS(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"OPTIONS"},
	}
}

func TRACE(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"TRACE"},
	}
}

func PATCH(url string, f Func) *Route {
	return &Route{
		Url:     url,
		Func:    f,
		Name:    getFunctionName(f),
		Methods: []string{"PATCH"},
	}
}

func Handler(h http.Handler) Func {
	if h == nil {
		panic("handler is nil")
	}
	return func(ctx *Ctx) { h.ServeHTTP(ctx, ctx.Request.raw) }
}
