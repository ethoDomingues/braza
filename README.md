# Ctrl

## [See the Documentation](https://github.com/ethoDomingues/ctrl/blob/main/docs/doc.md)

## Features
    - File Watcher (in development mode)
    - Error management
    - Router
    - Schema Validator (converts the form into a Struct - c3po package)
    - Rendering built-in (template/html)
    - Endpoints
    - Implements net/http
    
    - Supports
        Jwt 
        Cors 
        Sessions
        Websocket
        Middleware & Next
        URL Route builder

## Simple Example

### [With a correctly configured Go toolchain:](https://go.dev/doc/install)

```sh
go get github.com/ethoDomingues/ctrl
```

 _main.go_

```go
package main

import "github.com/ethoDomingues/ctrl"

func main() {
 app := ctrl.NewApp()
 app.GET("/hello", helloWorld)
 app.GET("/hello/{name}", helloUser) // 'name' is any string
 app.GET("/hello/{userID:int}", userByID) // 'userID' is only int

 fmt.Println(app.Listen())
}

func helloWorld(ctx *ctrl.Ctx) {
 hello := map[string]any{"Hello": "World"}
 ctx.JSON(hello, 200)
}

func helloUser(ctx *ctrl.Ctx) {
 rq := ctx.Request   // current Request
 name := rq.Args["name"]
 ctx.HTML("<h1>Hello "+name+"</h1>", 200)
}

func userByID(ctx *ctrl.Ctx) {
 rq := ctx.Request   // current Request
 id := rq.Args["userID"]
 user := AnyQuery(id)
 ctx.JSON(user, 200)
}
```
