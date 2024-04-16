# Braza: Simple example of an API application

```sh
$ git clone git@github.com:ethoDomingues/braza/examples/api.git
$ cd api
$ go mod init && go mod tidy
$ go run .
```
access 'localhost:5000' with the browser and see the application in operation

## Explanation
```go
// in main.go

type Post struct {
	ID   string
	Text string
} // data structure example

var db = map[string]*Post{} // simple example of database connection

func main() {
	cfg := &braza.Config{
		SecretKey:      "shiiii!!... it's is secret", // used to subscribe to sessions
		Servername:     "localhost:5000",             // reference for using subdomains
		TemplateFolder: "./",                         // reference ctx.RenderTemplate func
	}

	app := braza.NewApp(cfg) // create a new app and set the config
	app.BeforeRequest = beforeReq // exec before each request

	api := getApiRouter() // get a Router
	app.Mount(api)        // mount the router into the app

	app.GET("", func(ctx *braza.Ctx) {
		ctx.RenderTemplate("index.html")
	})            // add a new router to serve the index file
	app.Listen() // run the server
}

func beforeReq(ctx *braza.Ctx) {
	ctx.Global["db"] = db // add db in global context 
}
```

# to be continue....