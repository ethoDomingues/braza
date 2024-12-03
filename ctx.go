package braza

import (
	"context"
	"net/http"
	"slices"

	"github.com/ethoDomingues/c3po"
	"github.com/golang-jwt/jwt/v5"
)

type abortCode int

// Returns a new *braza.Ctx
func NewCtx(app *App, wr http.ResponseWriter, rq *http.Request) *Ctx {
	ctx := &Ctx{
		App:    app,
		Global: map[string]any{},
		Session: &Session{
			del:    []string{},
			claims: jwt.MapClaims{},
		},
		MatchInfo: &MatchInfo{},
	}

	ctx.Request = NewRequest(rq, ctx)
	ctx.Response = NewResponse(wr, ctx)

	c := context.Background()
	ctx.backCtx = context.WithValue(c, abortCode(1), nil)

	return ctx
}

type Ctx struct {
	App     *App           // Clone Current App
	Global  map[string]any // global variables of current request
	Session *Session       // Current Cookie Session

	*Response          // Current Response
	Request   *Request // Current Request

	// New Schema valid from route schema
	Schema        Schema
	SchemaFielder *c3po.Fielder

	// Contains information about the current request, route, etc...
	MatchInfo *MatchInfo

	mids       []Func
	midCounter int

	backCtx context.Context
}

func (ctx *Ctx) parseMids() {
	ctx.mids = slices.Concat(
		ctx.MatchInfo.Router.Middlewares,
		ctx.MatchInfo.Route.Middlewares,
	)
	ctx.mids = append(ctx.mids, ctx.MatchInfo.Func)
}

// executes the next middleware or main function of the request
func (ctx *Ctx) Next() {

	if ctx.midCounter < len(ctx.mids) {
		n := ctx.mids[ctx.midCounter]
		ctx.midCounter += 1
		n(ctx)
		if ctx.midCounter < len(ctx.mids) {
			ctx.Next()
		}
	}
}

func (ctx *Ctx) UrlFor(name string, external bool, args ...string) string {
	return ctx.App.UrlFor(name, external, args...)
}
