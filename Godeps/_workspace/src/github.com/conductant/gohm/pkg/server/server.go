package server

import (
	"fmt"
	"github.com/golang/glog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func Start(port int, endpoint http.Handler, onShutdown func() error, timeout time.Duration) (chan<- int, <-chan error) {
	shutdownTasks := make(chan func() error, 100)

	// Custom shutdown task
	shutdownTasks <- onShutdown

	glog.Infoln("Starting server")
	engineStop, engineStopped := RunServer(&http.Server{Handler: endpoint, Addr: fmt.Sprintf(":%d", port)})
	shutdownTasks <- func() error {
		glog.Infoln("Stopping engine")
		engineStop <- 1
		err := <-engineStopped
		glog.Infoln("Stopped engine. Err=", err)
		return err
	}

	// Pid file
	if pid, pidErr := savePidFile(fmt.Sprintf("%d", port)); pidErr == nil {
		// Clean up pid file
		shutdownTasks <- func() error {
			os.Remove(pid)
			glog.Infoln("Removed pid file:", pid)
			return nil
		}
	}
	shutdownTasks <- nil // stop on this

	// Triggers to start shutdown sequence
	fromKernel := make(chan os.Signal, 1)

	// kill -9 is SIGKILL and is uncatchable.
	signal.Notify(fromKernel, syscall.SIGHUP)  // 1
	signal.Notify(fromKernel, syscall.SIGINT)  // 2
	signal.Notify(fromKernel, syscall.SIGQUIT) // 3
	signal.Notify(fromKernel, syscall.SIGABRT) // 6
	signal.Notify(fromKernel, syscall.SIGTERM) // 15

	fromUser := make(chan int)
	stopped := make(chan error)
	go func() {
		select {
		case <-fromKernel:
			glog.Infoln("Received kernel signal to start shutdown.")
		case <-fromUser:
			glog.Infoln("Received user signal to start shutdown.")
		}
		for {
			task, ok := <-shutdownTasks
			if !ok || task == nil {
				break
			}
			if err := task(); err != nil {
				glog.Warningln("Error while shutting down:", err)
				stopped <- err
				return
			}
		}
		stopped <- nil
		return
	}()

	return fromUser, stopped
}

// Runs the http server.  This server offers more control than the standard go's default http server
// in that when a 'true' is sent to the stop channel, the listener is closed to force a clean shutdown.
// The return value is a channel that can be used to block on.  An error is received if server shuts
// down in error; or a nil is received on a clean signalled shutdown.
func RunServer(server *http.Server) (chan<- int, <-chan error) {
	protocol := "tcp"
	// e.g. 0.0.0.0:80 or :80 or :8080
	if match, _ := regexp.MatchString("[a-zA-Z0-9\\.]*:[0-9]{2,}", server.Addr); !match {
		protocol = "unix"
	}

	listener, err := net.Listen(protocol, server.Addr)
	if err != nil {
		panic(err)
	}

	stop := make(chan int)
	stopped := make(chan error)

	if protocol == "unix" {
		if _, err = os.Lstat(server.Addr); err == nil {
			// Update socket filename permission
			os.Chmod(server.Addr, 0777)
		}
	}

	userInitiated := new(bool)
	go func() {
		<-stop
		*userInitiated = true
		glog.Infoln("Closing listener")
		listener.Close()
		glog.Infoln("Listener closed")
	}()

	go func() {

		glog.Infoln("Starting", protocol, "listener at", server.Addr)

		// Serve will block until an error (e.g. from shutdown, closed connection) occurs.
		err := server.Serve(listener)

		switch {
		case !*userInitiated && err != nil:
			glog.Infoln("Engine stopped due to error", err)
			panic(err)
		case *userInitiated:
			stopped <- nil
		default:
			stopped <- err
		}
	}()
	return stop, stopped
}

func savePidFile(args ...string) (string, error) {
	cmd := filepath.Base(os.Args[0])
	pidFile, err := os.Create(fmt.Sprintf("%s-%s.pid", cmd, strings.Join(args, "-")))
	if err != nil {
		return "", err
	}
	defer pidFile.Close()
	fmt.Fprintf(pidFile, "%d", os.Getpid())
	return pidFile.Name(), nil
}
