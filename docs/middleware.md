# Middleware - Full Usage Example

```go
package main

import (
 "fmt"

 "github.com/ethoDomingues/ctrl"
)

func main() {
 app := ctrl.NewApp(nil)
 app.Middlewares = ctrl.Middlewares{middle1, middle2}

 app.AddRoute(&ctrl.Route{
  Url:         "/",
  Func:        home,
  Middlewares: ctrl.Middlewares{middle3, middle4},
 })

 app.AddRoute(&ctrl.Route{
  Url:  "/echo",
  Name: "echo",
  Func: echo,
 })
 app.Listen()
}

func home(ctx *ctrl.Ctx) {
 ctx.response.HTML("<h1>Hello</h1>",200)
}


func middle1(ctx *ctrl.Ctx) {
 fmt.Println("middle 1")
 ctx.Next()
}

func middle2(ctx *ctrl.Ctx) {
 fmt.Println("middle 2")
 ctx.Next()
}

func middle3(ctx *ctrl.Ctx) {
 fmt.Println("middle 3")
 ctx.Next()
}
func middle4(ctx *ctrl.Ctx) {
 fmt.Println("middle 4")
 ctx.Next()
}
```
