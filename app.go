package braza

import (
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

var (
	allowEnv = map[string]string{
		"":            "development",
		"d":           "development",
		"dev":         "development",
		"development": "development",

		"t":       "test",
		"test":    "test",
		"testing": "test",

		"p":          "production",
		"prod":       "production",
		"production": "production",
	}
	l             = newLogger("")
	listenInAll   bool
	localAddress  = getOutboundIP()
	htmlTemplates map[string]*template.Template
)

/*
Create a new app with a default settings
*/
func NewApp(cfg *Config) *App {
	router := NewRouter("")
	router.is_main = true
	c := &Config{}
	if cfg != nil {
		*c = *cfg // se estiver clonando o app, evita alguns erros
	}
	return &App{
		Config:       c,
		Router:       router,
		routers:      []*Router{router},
		routerByName: map[string]*Router{"": router},
	}
}

type App struct {
	*Router
	*Config

	BasicAuth     func(*Ctx) (user string, pass string, ok bool) // custom func to parse Authorization Header
	AfterRequest, // exec after each request (if the application dont crash)
	BeforeRequest, // exec before each request
	TearDownRequest Func // exec after each request, after send to cleint ( this dont has effect in response)

	routers      []*Router
	routerByName map[string]*Router

	Srv *http.Server

	built bool
}

func (app *App) logStarterListener() {
	addr, port, err := net.SplitHostPort(app.Srv.Addr)
	if err != nil {
		l.err.Panic(err)
	}
	envDev := app.Env == "development"
	if listenInAll {
		app.Srv.Addr = localAddress
		if envDev {
			l.Defaultf("Server is listening on all address in %sdevelopment mode%s", _RED, _RESET)
		} else {
			l.Default("Server is listening on all address")
		}
		l.info.Printf("          listening on: http://%s:%s", getOutboundIP(), port)
		l.info.Printf("          listening on: http://0.0.0.0:%s", port)
	} else {
		if envDev {
			l.Defaultf("Server is listening in %sdevelopment mode%s", _RED, _RESET)
		} else {
			l.Default("Server is linsten")
		}
		if addr == "" {
			addr = "localhost"
		}
		l.info.Printf("          listening on: http://%s:%s", addr, port)
	}
}

func (app *App) startListener(c chan error) {
	c <- app.Srv.ListenAndServe()
}

func (app *App) startListenerTLS(cert, key string, c chan error) {
	c <- app.Srv.ListenAndServeTLS(cert, key)
}

func runSrv(app *App, cert, key string, host ...string) (err error) {
	app.Build(host...)
	var reboot = make(chan bool)
	var srvErr = make(chan error)

	if !app.DisableFileWatcher {
		if app.Env == "development" {
			go fileWatcher(reboot)
		}
	}

	if !app.Silent {
		app.logStarterListener()
	}

	if cert != "" || key != "" {
		go app.startListenerTLS(cert, key, srvErr)
	} else {
		go app.startListener(srvErr)
	}

	for {
		select {
		case <-reboot:
			app.Srv.Close()
			RestarSelf()
		case err = <-srvErr:
			if !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
	}
}

/*
Build the App, but not start serve

example:

	func index(ctx braza.Ctx){}

	// it's work
	func main() {
		app := braza.NewApp()
		app.GET("/",index)
		app.Build(":5000")
		app.UrlFor("index",true)
	}
	// it's don't work
	func main() {
		app := braza.NewApp()
		app.GET("/",index)
		app.UrlFor("index",true)
	}
*/
func (app *App) Build(addr ...string) {
	app.parseApp()
	l = newLogger(app.LogFile)

	var address string
	if len(addr) > 0 {
		a_ := addr[0]
		if a_ != "" {
			_, _, err := net.SplitHostPort(a_)
			if err == nil {
				address = a_
			}
		}
	}
	if address == "" {
		address = "127.0.0.1:5000"
	}

	if strings.Contains(address, "0.0.0.0") {
		listenInAll = true
	}

	if app.Srv == nil {
		app.Srv = &http.Server{
			Addr:           address,
			Handler:        app,
			MaxHeaderBytes: 1 << 20,
		}
	} else {
		app.Srv.Handler = app
		app.Srv.MaxHeaderBytes = 1 << 20
	}
}

/*
Create a clone of app.
obs: changes to the application do not affect the clone
*/
func (app *App) Clone() *App {
	clone := NewApp(app.Config)
	*clone.Router = *app.Router

	clone.BasicAuth = app.BasicAuth
	clone.AfterRequest = app.AfterRequest
	clone.BeforeRequest = app.BeforeRequest
	clone.TearDownRequest = app.TearDownRequest

	clone.Srv = app.Srv
	clone.routers = app.routers
	clone.routerByName = app.routerByName
	return clone
}

// Parse the router and your routes
func (app *App) parseApp() {
	app.checkConfig()
	if app.Servername != "" {
		Srv := app.Servername
		Srv = strings.TrimPrefix(Srv, ".")
		Srv = strings.TrimSuffix(Srv, "/")
		app.Servername = Srv
	}

	if env, ok := allowEnv[app.Env]; ok {
		app.Env = env
	} else {
		l.err.Fatalf("environment '%s' is not valid", app.Env)
	}

	if !app.DisableStatic {
		staticUrl := "/assets"
		fp := "/{filepath:path}"
		if app.StaticUrlPath != "" {
			staticUrl = app.StaticUrlPath
		}
		path := filepath.Join(staticUrl, fp)
		app.AddRoute(&Route{
			Url:      path,
			Func:     serveFileHandler,
			Name:     "assets",
			IsStatic: true,
		})
	}

	// se o usuario mudar o router principal, isso evita alguns erro
	if !app.is_main {
		app.is_main = true

		if app.Router.Routes == nil {
			app.Router.Routes = []*Route{}
		}
		if app.Router.routesByName == nil {
			app.Router.routesByName = map[string]*Route{}
		}
		if app.Router.Cors == nil {
			app.Router.Cors = &Cors{}
		}
		if app.Router.Middlewares == nil {
			app.Router.Middlewares = []Func{}
		}

	}
	for _, router := range app.routers {
		router.parse(app.Servername)
		if router != app.Router {
			maps.Copy(app.routesByName, router.routesByName)
		}
	}
	app.routerByName[app.Router.Name] = app.Router

	app.built = true
}

/*
Register the router in app

	func main() {
		api := braza.NewRouter("api")
		api.post("/products")
		api.get("/products/{productID:int}")

		app := braza.NewApp(nil)

		app.Mount(getApiRouter)
		app.Listen()
	}
*/
func (app *App) Mount(routers ...*Router) {
	for _, router := range routers {
		if router.Name == "" {
			panic(fmt.Errorf("the routers must be named"))
		} else if _, ok := app.routerByName[router.Name]; ok {
			panic(fmt.Errorf("router '%s' already regitered", router.Name))
		}
		router.parse(app.Servername)
		app.routerByName[router.Name] = router
		app.routers = append(app.routers, router)
	}
}

func (app *App) ErrorHandler(statusCode int, f Func) {
	if app.errHandlers == nil {
		app.errHandlers = map[int]Func{}
	}
	app.errHandlers[statusCode] = f
}

func (app *App) ShowRoutes() { listRoutes(app) }

func (app *App) Listen(host ...string) (err error) {
	return runSrv(app, "", "", host...)
}

func (app *App) ListenTLS(certFile, certKey string, host ...string) (err error) {
	return runSrv(app, certFile, certKey, host...)
}

/*



 */

func (app *App) match(ctx *Ctx) bool {
	rq := ctx.Request

	if app.Servername != "" {
		rqUrl := rq.URL.Host
		if net.ParseIP(rqUrl) != nil {
			return false
		}
		if !strings.Contains(rqUrl, app.Servername) {
			return false
		}
	}

	for _, router := range app.routers {
		if router.match(ctx) {
			if router.StrictSlash && !strings.HasSuffix(rq.URL.Path, "/") {
				args := []string{}
				for k, v := range rq.Args {
					args = append(args, k, v)
				}
				ctx.Response.Redirect(ctx.UrlFor(ctx.MatchInfo.Route.Name, true, args...))
			}
			return true
		}
	}
	mi := ctx.MatchInfo
	if mi.MethodNotAllowed != nil {
		ctx.MethodNotAllowed()
	} else {
		ctx.NotFound()
	}
	return false
}

// exec route and handle errors of application
func (app *App) execRoute(ctx *Ctx) {
	rq := ctx.Request
	mi := ctx.MatchInfo
	if mi.Func == nil && rq.Method == "OPTIONS" {
		optionsHandler(ctx)
	} else {
		rq.parse()
		ctx.parseMids()
		if app.BeforeRequest != nil {
			app.BeforeRequest(ctx)
		}
		ctx.Next()
	}
}

func (app *App) execHandlerError(ctx *Ctx) {
	err := recover()
	if err != nil {
		var errStr string
		if s, ok := err.(string); ok {
			if s == "" {
				return
			}
			errStr = s
		} else {
			errStr = err.(error).Error()
		}

		if strings.HasPrefix(errStr, "abort:") {
			strCode := strings.TrimPrefix(errStr, "abort:")
			code, err := strconv.Atoi(strCode)
			if err != nil {
				panic(err)
			}
			if h, ok := app.errHandlers[code]; ok {
				ctx.StatusCode = code
				ctx.Reset()
				h(ctx)
			}
		} else {
			if h, ok := app.errHandlers[500]; ok {
				ctx.StatusCode = 500
				ctx.Reset()
				h(ctx)
			} else {
				panic(errStr)
			}
		}
	}
}

func execTeardown(ctx *Ctx) {
	if ctx.App.TearDownRequest != nil {
		go ctx.App.TearDownRequest(ctx)
	}
}

func (app *App) closeConn(ctx *Ctx) {
	rsp := ctx.Response
	err := recover()
	mi := ctx.MatchInfo

	defer execTeardown(ctx)
	defer l.LogRequest(ctx)

	if err == nil {
		if mi.Match {
			if ctx.Session.changed {
				rsp.SetCookie(ctx.Session.save())
			}
			rsp.parseHeaders()
			rsp.Headers.Save(rsp.raw)
		}
		rsp.raw.WriteHeader(rsp.StatusCode)
		fmt.Fprint(rsp.raw, rsp.String())
	} else {
		statusText := ""
		errStr, ok := err.(string)
		if ok && errStr == "ok" {
			if ctx.App.AfterRequest != nil {
				ctx.App.AfterRequest(ctx)
			}
		} else {
			rsp.StatusCode = 500
			statusText = "500 Internal Server Error"
			l.Error(err)
		}
		rsp.raw.WriteHeader(rsp.StatusCode)
		fmt.Fprint(rsp.raw, statusText)
	}
}

// # http.Handler
func (app *App) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	ctx := newCtx(app.Clone(), wr, req)
	defer app.closeConn(ctx)
	defer app.execHandlerError(ctx)
	if app.match(ctx) {
		app.execRoute(ctx)
	}
}

// # Url Builder
//
//	app.GET("/users/{userID:int}", index)
//
//	app.UrlFor("index", false, "userID", "1"}) //  /users/1
//	app.UrlFor("index", true, "userID", "1"}) // http://yourAddress/users/1
func (app *App) UrlFor(name string, external bool, args ...string) string {
	var (
		host   = ""
		route  *Route
		router *Router
	)
	if app.Srv == nil {
		l.err.Fatalf("you are trying to use this function outside of a context")
	}
	if len(args)%2 != 0 {
		l.err.Fatalf("numer of args of build url, is invalid: UrlFor only accept pairs of args ")
	}

	// check route name
	if r, ok := app.routesByName[name]; ok {
		route = r
	} else {
		panic(fmt.Sprintf("Route '%s' is undefined \n", name))
	}
	router = route.router

	params := map[string]string{}
	c := len(args)
	for i := 0; i < c; i++ {
		if i%2 != 0 {
			continue
		}
		params[args[i]] = args[i+1]
	}

	// Build Host
	if external {
		schema := "http://"
		if app.ListeningInTLS || len(app.Srv.TLSConfig.Certificates) > 0 {
			schema = "https://"
		}
		if router.Subdomain != "" {
			host = schema + router.Subdomain + "." + app.Servername
		} else {
			if app.Servername == "" {
				_, p, _ := net.SplitHostPort(app.Srv.Addr)
				h := net.JoinHostPort(localAddress, p)
				host = schema + h
			} else {
				host = schema + app.Servername
			}
		}
	}
	url := route.MountURI(args...)
	return host + url
}
