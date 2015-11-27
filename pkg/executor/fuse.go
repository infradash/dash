package executor

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/pubsub"
	"github.com/qorio/maestro/pkg/registry"
	"golang.org/x/net/context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrMissingMount         = errors.New("err-missing-mount-point")
	ErrMissingResource      = errors.New("err-missing-resource")
	ErrMountPointNotAbs     = errors.New("err-mount-point-not-absolute-path")
	ErrResourceNotSupported = errors.New("err-resource-not-supported")
	ResourceMqtt            = "mqtt://"
	ResourceZk              = "zk://"
	ResourceTypes           = []string{ResourceMqtt, ResourceZk}

	resourceFS = map[string]func(resource string) fs.FS{
		ResourceMqtt: func(resource string) fs.FS {
			return &MqttFs{
				Topic: pubsub.Topic(resource),
			}
		},
		ResourceZk: func(resource string) fs.FS {
			return &ZkFs{
				Path: registry.Path(resource),
			}
		},
	}

	fsOpen     = map[*fuse.Conn]*fuse.Conn{}
	fsRegister = make(chan *fuse.Conn)
	fsClose    = make(chan *fuse.Conn)
	fsErrors   = make(chan error)
)

func init() {
	go func() {
		for {
			select {
			case c := <-fsRegister:
				fsOpen[c] = c
			case c := <-fsClose:
				c.Close()
				delete(fsOpen, c)
			case e := <-fsErrors:
				glog.Warningln("Error from fuse filesystem:", e)
			}
		}
	}()
}

func StartFileMounts(list []Fuse) error {
	for _, f := range list {
		if err := f.IsValid(); err != nil {
			fsErrors <- err
			return err
		}
		if c, err := f.Mount(); err != nil {
			fsErrors <- err
			return err
		} else {
			fsRegister <- c
		}
	}
	return nil
}

func StopFileMounts() {
	stop := []*fuse.Conn{}
	for c, _ := range fsOpen {
		stop = append(stop, c)
	}
	for _, c := range stop {
		fsClose <- c
	}
}

func (this *Fuse) IsValid() error {
	if len(this.MountPoint) == 0 {
		return ErrMissingMount
	}
	if len(this.Resource) == 0 {
		return ErrMissingResource
	}
	if !filepath.IsAbs(this.MountPoint) {
		return ErrMountPointNotAbs
	}

	misses := 0
	for _, p := range ResourceTypes {
		if strings.Index(this.Resource, p) != 0 {
			misses++
		}
	}
	if misses == len(ResourceTypes) {
		return ErrResourceNotSupported
	}
	return nil
}

func (this *Fuse) GetResourceType() string {
	switch {
	case strings.Index(this.Resource, ResourceMqtt) == 0:
		return this.Resource[0:len(ResourceMqtt)]
	case strings.Index(this.Resource, ResourceZk) == 0:
		return this.Resource[0:len(ResourceZk)]
	}
	return ""
}

func (this *Fuse) Mount() (*fuse.Conn, error) {
	if err := this.IsValid(); err != nil {
		return nil, err
	}

	switch {
	case strings.Index(this.MountPoint, ".") == 0:
		this.MountPoint = strings.Replace(this.MountPoint, ".", os.Getenv("PWD"), 1)
	case strings.Index(this.MountPoint, "~") == 0:
		this.MountPoint = strings.Replace(this.MountPoint, "~", os.Getenv("HOME"), 1)
	}

	var perm os.FileMode = 0777
	fmt.Sscanf(this.Perm, "%v", &perm)

	if err := os.MkdirAll(this.MountPoint, perm); err != nil {
		return nil, err
	}

	filesys := resourceFS[this.GetResourceType()](this.Resource)
	c, err := fuse.Mount(this.MountPoint)
	if err != nil {
		return nil, err
	}

	//	go func() {
	if err := fs.Serve(c, filesys); err != nil {
		fsClose <- c
		fsErrors <- err
	}
	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		fsErrors <- err
	}
	glog.Infoln("FS", filesys, "done serving")
	//	}()

	return c, nil
}

type MqttFs struct {
	Topic pubsub.Topic
}

type ZkFs struct {
	Path registry.Path
}

func (this *MqttFs) Root() (fs.Node, error) {
	return &Dir{}, nil
}

type Dir struct {
}

var _ = fs.Node(&Dir{})
var _ = fs.HandleReadDirAller(&Dir{})

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	res := []fuse.Dirent{
		fuse.Dirent{
			Name: "test",
			Type: fuse.DT_File,
		},
	}
	return res, nil
}

type file struct {
}

func (this *Dir) Attr(c context.Context, attr *fuse.Attr) error {
	attr.Mode = 0755
	attr.Size = 0
	attr.Mtime = time.Now()
	attr.Ctime = time.Now()
	attr.Crtime = time.Now()
	return nil
}

func (this *ZkFs) Root() (fs.Node, error) {
	return nil, nil
}
