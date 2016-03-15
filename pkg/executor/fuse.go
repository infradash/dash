package executor

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/zk"
	"os"
	"strings"
)

type FileSystem interface {
	Mount(dir string, perm os.FileMode) error
	Close() error
}

var (
	ErrMissingMount         = errors.New("err-missing-mount-point")
	ErrMissingResource      = errors.New("err-missing-resource")
	ErrMountPointNotAbs     = errors.New("err-mount-point-not-absolute-path")
	ErrResourceNotSupported = errors.New("err-resource-not-supported")
	ResourceMqtt            = "mqtt://"
	ResourceZk              = "zk://"
	ResourceTypes           = []string{ResourceMqtt, ResourceZk}

	resourceFS = map[string]func(resource string, zc zk.ZK) FileSystem{
		ResourceZk: func(resource string, zc zk.ZK) FileSystem {
			return NewZkFS(zc, registry.Path(resource))
		},
	}

	fsOpen     = map[FileSystem]FileSystem{}
	fsRegister = make(chan FileSystem)
	fsClose    = make(chan FileSystem)
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

func StartFileMounts(list []*Fuse, zc zk.ZK) error {
	for _, f := range list {
		if err := f.IsValid(); err != nil {
			fsErrors <- err
			return err
		}
		f.zc = zc
		if fs, err := f.Mount(); err != nil {
			glog.Infoln("Error mounting filesystem:", f, "err=", err)
			fsErrors <- err
			return err
		} else {
			fsRegister <- fs
		}
	}
	return nil
}

func StopFileMounts() {
	stop := []FileSystem{}
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

func strip_resource_type(p string) string {
	switch {
	case strings.Index(p, ResourceMqtt) == 0:
		return p[len(ResourceMqtt):]
	case strings.Index(p, ResourceZk) == 0:
		return p[len(ResourceZk):]
	default:
		return p
	}
}

func (this *Fuse) Mount() (FileSystem, error) {
	if err := this.IsValid(); err != nil {
		return nil, err
	}

	switch {
	case strings.Index(this.MountPoint, ".") == 0:
		this.MountPoint = strings.Replace(this.MountPoint, ".", os.Getenv("PWD"), 1)
	case strings.Index(this.MountPoint, "~") == 0:
		this.MountPoint = strings.Replace(this.MountPoint, "~", os.Getenv("HOME"), 1)
	}

	var perm os.FileMode = 0644
	fmt.Sscanf(this.Perm, "%v", &perm)

	if err := os.MkdirAll(this.MountPoint, perm); err != nil {
		return nil, err
	}

	filesys := resourceFS[this.GetResourceType()](strip_resource_type(this.Resource), this.zc)
	go func() {
		glog.Infoln("Mounting filesystem:", this, "type=", this.GetResourceType(), "filesys=", filesys)
		glog.Infoln("Start serving", filesys, "on", this.MountPoint, "backed", this.Resource)
		filesys.Mount(this.MountPoint, perm)
	}()

	return filesys, nil
}

func NewZkFS(zc zk.ZK, path registry.Path) *zkFs {
	return &zkFs{
		impl: zk.NewFS(zc, path),
	}
}

type zkFs struct {
	impl *zk.FS
}

func (this *zkFs) Close() error {
	return this.impl.Shutdown()
}

func (this *zkFs) Mount(dir string, perm os.FileMode) error {
	return this.impl.Mount(dir, perm)
}
