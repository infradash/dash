package template

import (
	"bytes"
	"golang.org/x/net/context"
	"io"
	"os"
	"os/exec"
	"time"
)

type timeoutContextKey int

var TimeoutContextKey timeoutContextKey = 1

func ContextPutTimeout(ctx context.Context, duration time.Duration) context.Context {
	return context.WithValue(ctx, TimeoutContextKey, duration)
}

func ContextGetTimeout(ctx context.Context) time.Duration {
	if v, ok := ctx.Value(TimeoutContextKey).(time.Duration); ok {
		return v
	}
	return 24 * time.Hour
}

// This will bock until the shell executes to completion.  The output to stdout is
// buffered internally and available for reading in the returned io.Reader.
// This function will block until all the data to stdout has been captured.
func ExecuteShell(ctx context.Context) interface{} {
	return func(line string) (io.Reader, error) {
		c := exec.Command("sh", "-")

		copyDone := make(chan error)
		timeout := time.After(ContextGetTimeout(ctx))

		output := new(bytes.Buffer)
		if stdout, err := c.StdoutPipe(); err == nil {
			fanout := io.MultiWriter(os.Stdout, output)
			go func() {
				_, err := io.Copy(fanout, stdout)
				copyDone <- err
			}()
		} else {
			return nil, err
		}

		if stderr, err := c.StderrPipe(); err == nil {
			go func() {
				io.Copy(os.Stderr, stderr)
			}()
		} else {
			return nil, err
		}
		stdin, err := c.StdinPipe()
		if err != nil {
			return nil, err
		}
		if err := c.Start(); err != nil {
			return nil, err
		}
		if _, err := stdin.Write([]byte(line)); err != nil {
			stdin.Close()
			return nil, err
		}
		stdin.Close() // finished
		err = c.Wait()
		if err != nil {
			return nil, err
		}
		// Waits for the stdout and stderr copy goroutines to complete.
		select {
		case err = <-copyDone:
			break
		case <-timeout:
			break
		}
		return output, err
	}
}
