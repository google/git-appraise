package commands

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"

	"github.com/google/git-appraise/repository"
)

var webFlagSet = flag.NewFlagSet("web", flag.ExitOnError)

var (
	webPort = webFlagSet.Int("port", 12345, "Port to run git-appraise-web.")
)

func openBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func openWeb(args []string) error {
	webFlagSet.Parse(args)

	bin, err := exec.LookPath("git-appraise-web")
	if err != nil {
		return err
	}

	cmd := exec.Command(bin, "-port", strconv.Itoa(*webPort))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		fmt.Printf("starting %s on http://localhost:%d\n", bin, *webPort)
		err = cmd.Run()
		wg.Done()
	}()

	if err = openBrowser(fmt.Sprintf("http://localhost:%d", *webPort)); err != nil {
		return err
	}

	wg.Wait()
	return err
}

// rejectCmd defines the "reject" subcommand.
var webCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s web [<option>...]\n\nOptions:\n", arg0)
		webFlagSet.PrintDefaults()
	},
	RunMethod: func(_ repository.Repo, args []string) error {
		return openWeb(args)
	},
}
