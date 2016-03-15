package zk

import (
	"errors"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	ErrNotConnected            = errors.New("zk-not-initialized")
	ErrConflict                = errors.New("error-conflict")
	ErrNotExist                = zk.ErrNoNode
	ErrConnectionClosed        = zk.ErrConnectionClosed
	ErrUnknown                 = zk.ErrUnknown
	ErrAPIError                = zk.ErrAPIError
	ErrNoAuth                  = zk.ErrNoAuth
	ErrBadVersion              = zk.ErrBadVersion
	ErrNoChildrenForEphemerals = zk.ErrNoChildrenForEphemerals
	ErrNodeExists              = zk.ErrNodeExists
	ErrNotEmpty                = zk.ErrNotEmpty
	ErrSessionExpired          = zk.ErrSessionExpired
	ErrInvalidACL              = zk.ErrInvalidACL
	ErrAuthFailed              = zk.ErrAuthFailed
	ErrClosing                 = zk.ErrClosing
	ErrNothing                 = zk.ErrNothing
	ErrSessionMoved            = zk.ErrSessionMoved
)
