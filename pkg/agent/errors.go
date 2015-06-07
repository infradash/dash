package agent

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"runtime"
)

var (
	ErrBadSchedulerConfig             = errors.New("bad-scheduler-config")
	ErrCannotDetermineContainerImage  = errors.New("cannot-determine-image")
	ErrUnknownService                 = errors.New("unknown-service")
	ErrHeapEmpty                      = errors.New("heap-empty")
	ErrNoImage                        = errors.New("no-image-at-path")
	ErrNoImageRegistryAuth            = errors.New("no-image-registry-auth")
	ErrNoHost                         = errors.New("no-host")
	ErrNoName                         = errors.New("no-name")
	ErrNotConnectedToRegistry         = errors.New("not-connected-to-registry")
	ErrNoContainerInformation         = errors.New("no-container-information")
	ErrNoDomain                       = errors.New("no-domain")
	ErrNoConfig                       = errors.New("no-config")
	ErrNoDockerName                   = errors.New("docker-name-env-not-set")
	ErrMoreThanOneAgent               = errors.New("more-than-one-agent")
	ErrNoDockerTlsCert                = errors.New("no-docker-tls-cert")
	ErrNoDockerTlsKey                 = errors.New("no-docker-tls-key")
	ErrBadDockerTlsCert               = errors.New("cannot-add-docker-tls-cert")
	ErrWatchReleaseMissingRegistryKey = errors.New("watch-release-missing-registry-key")
	ErrNoSchedulerReleasePath         = errors.New("no-scheduler-release-path")
	ErrMaxAttemptsExceeded            = errors.New("max-attempts-exceeded")
	ErrBadSchedulerSpec               = errors.New("bad-scheduler-spec")
	ErrBadVacuumConfig                = errors.New("bad-vacuum-config")
	ErrDebug                          = errors.New("REMOVE_ME")
)

func ExceptionEvent(err error, context interface{}, a ...interface{}) {
	source := ""
	_, file, line, ok := runtime.Caller(1)
	if ok {
		source = fmt.Sprintf("%s:%d", file, line)
	}

	glog.Warningln("!!!! Err=", err, "Source=", source, "Context=", context, fmt.Sprintln(a...))
}
