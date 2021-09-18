package main

import "C"
import (
	"errors"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/icccm"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	tsExecutable = "./TeamSpeak"
	tsWindowName = "TeamSpeak"
	tsInstance   = "TeamSpeak"
	tsClass      = "TeamSpeak"
)

func main() {
	file, err := os.OpenFile("tswm.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	fatal(err)
	defer file.Close()

	writer := io.MultiWriter(os.Stdout, file)
	log.SetOutput(writer)
	log.Println("configured logger")

	runTeamspeak()

	log.Printf("searching windows...")
	windowsUpdated := 0
	attempts := 0
	for ok := true; ok; ok = windowsUpdated < 2 && attempts < 5 {
		time.Sleep(1 * time.Second)
		windowsUpdated += setTeamspeakWmClass()
		attempts++
	}

	log.Printf("updated %d windows in %d attempt(s)", windowsUpdated, attempts)

	if windowsUpdated == 0 {
		fatal(errors.New("could not update teamspeak window class in time"))
	}
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func runTeamspeak() {
	log.Println("starting teamspeak...")
	cmd := exec.Command(tsExecutable)
	fatal(cmd.Start())
}

func setTeamspeakWmClass() int {
	x, err := xgb.NewConn()
	fatal(err)

	defer x.Close()

	screen := xproto.Setup(x).DefaultScreen(x)
	teamspeakWindows := findWindowsByName(x, screen.Root, tsWindowName)

	xu, err := xgbutil.NewConn()
	fatal(err)

	defer x.Close()

	for _, window := range teamspeakWindows {
		log.Printf("setting WM_CLASS on window %d \n", window)

		icccm.WmClassSet(xu, window, &icccm.WmClass{
			Instance: tsInstance,
			Class:    tsClass,
		})

	}

	return len(teamspeakWindows)
}

// Recursively search the window tree and return a slice of windows that exactly match the name parameter
func findWindowsByName(X *xgb.Conn, parent xproto.Window, name string) []xproto.Window {
	query := xproto.QueryTree(X, parent)
	reply, _ := query.Reply()

	log.Printf("found %d children on window %d", reply.ChildrenLen, parent)

	var windows []xproto.Window
	for _, child := range reply.Children {
		r, _ := xproto.GetProperty(X, false, child, xproto.AtomWmName, xproto.AtomAny,
			0, (1<<32)-1).Reply()

		if r.ValueLen == 0 {
			continue
		}

		wmName := string(r.Value)

		if wmName == name {
			windows = append(windows, child)
		}

		childWindows := findWindowsByName(X, child, name)
		windows = append(windows, childWindows...)
	}

	return windows
}
