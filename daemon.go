package braza

import (
	"errors"
)

func runAppDaemon(app *App, err chan map[string]error) {
	err <- map[string]error{app.uuid: app.Listen()}
}

func Daemon(apps ...*App) error {
	if len(apps) < 2 {
		panic(errors.New("Daemon precisa de pelo menos 2 apps"))
	}
	cErrs := make(chan map[string]error)
	countErrors := map[string]int{}
	if mapStackApps == nil {
		mapStackApps = map[string]*App{}
	}

	for c, app := range apps {
		if app.Name == "" {
			l.warn.Println("When using 'Daemon', a good practice is to name all 'apps'")
		}
		countErrors[app.uuid] = 0
		mapStackApps[app.uuid] = app
		if c > 0 {
			app.DisableFileWatcher = true
		}
		app.Build()
		go runAppDaemon(app, cErrs)
	}

	for {
		<-cErrs
		for _, a := range mapStackApps {
			a.Srv.Close()
		}
	}

}
