package run

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Command ...
func Command(name string, args ...string) error {
	full := fmt.Sprintf("%s %s", name, strings.Join(args, " "))

	log.Printf("Running %q\n", full)
	cmd := exec.Command(name, args...)

	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	wg.Add(2)
	go func() {
		io.Copy(os.Stdout, stdout)
		wg.Done()
	}()
	go func() {
		io.Copy(os.Stderr, stderr)
		wg.Done()
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error while runnning command %q: %v", full, err)
	}

	wg.Wait()
	return nil
}
