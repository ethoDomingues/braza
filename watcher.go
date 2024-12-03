package braza

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func getFileStats() map[string]time.Time {
	mapFiles := map[string]time.Time{}
	rootpath, _ := os.Getwd()
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		f, fErr := d.Info()
		if fErr != nil {
			return nil
		}
		if strings.HasSuffix(f.Name(), ".go") {
			mapFiles[filepath.Join(rootpath, path)] = f.ModTime()
		}
		return nil
	})
	return mapFiles
}

func fileWatcher(reboot chan bool) {
	mapF := getFileStats()
	for {
		time.Sleep(time.Second)
		for fn, lastMode := range mapF {
			f, err := os.Open(fn)
			if err != nil {
				reboot <- true
			}
			fs, _ := f.Stat()
			f.Close()
			if fs.ModTime().After(lastMode) {
				mapF[fn] = fs.ModTime()
				reboot <- true
			}
		}
	}
}

func selfReboot() {
	fmt.Println()
	l.warn.Print("Changes detected, restarting the server...\n\n")
	self, _ := os.Getwd()
	for _, app := range mapStackApps {
		app.Srv.Close()
	}
	nArgs := append([]string{"run", self}, os.Args[1:]...)

	errBuf := bytes.NewBufferString("")
	cmd := exec.Command("go", nArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = errBuf
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	err := cmd.Run()

	if err != nil && err.Error() == "exit status 1" {
		l.err.Println(errBuf.String())
	}
}
