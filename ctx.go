package braza

import (
	"slices"

	"github.com/ethoDomingues/c3po"
)

// Returns a new *braza.Ctx
func newCtx(app *App) *Ctx {
	return &Ctx{
		App:       app,
		Global:    map[string]any{},
		MatchInfo: &MatchInfo{},
		Session:   newSession(app.SecretKey),
	}
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

	mids  Middlewares
	c_mid int
}

// executes the next middleware or main function of the request
func (ctx *Ctx) Next() {
	if ctx.c_mid < len(ctx.mids) {
		n := ctx.mids[ctx.c_mid]
		ctx.c_mid += 1
		n(ctx)
	} else {
		panic(ErrHttpAbort)
	}
}

func (ctx *Ctx) parseMids() {
	ctx.mids = slices.Concat(
		ctx.MatchInfo.Router.Middlewares,
		ctx.MatchInfo.Route.Middlewares,
	)
	ctx.mids = append(ctx.mids, ctx.MatchInfo.Func)
}

func (ctx *Ctx) UrlFor(name string, external bool, args ...string) string {
	return ctx.App.UrlFor(name, external, args...)
}
