package braza

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func getFileStats() map[string]time.Time {
	mapFiles := map[string]time.Time{}
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		f, fErr := d.Info()
		if fErr != nil {
			return nil
		}

		if strings.HasSuffix(f.Name(), ".go") {
			mapFiles[path] = f.ModTime()
		}
		return nil
	})
	return mapFiles
}

func fileWatcher(c chan bool) {
	mapF := getFileStats()
	for {
		time.Sleep(time.Second)
		for fn, lastMode := range mapF {
			f, err := os.Open(fn)
			if err != nil {
				c <- true
			}
			fs, _ := f.Stat()
			f.Close()
			if fs.ModTime().After(lastMode) {
				mapF[fn] = fs.ModTime()
				c <- true
			}
		}
	}
}

func RestarSelf() {
	fmt.Print("\nChanges detected, restarting the server...\n\n")
	_, self, _, ok := runtime.Caller(3)
	if !ok {
		panic("impossible recover the file")
	}
	dir := filepath.Join(self, "../")

	nArgs := append([]string{"run", dir}, os.Args[1:]...)

	errBuf := bytes.NewBufferString("")
	cmd := exec.Command("go", nArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = errBuf
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	err := cmd.Run()
	if ers := err.Error(); ers == "exit status 1" {
		fmt.Println(errBuf.String())
	}
}
