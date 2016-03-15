package zk

import (
	"encoding/json"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	event_types = map[zk.EventType]string{
		zk.EventNodeCreated:         "node-created",
		zk.EventNodeDeleted:         "node-deleted",
		zk.EventNodeDataChanged:     "node-data-changed",
		zk.EventNodeChildrenChanged: "node-children-changed",
		zk.EventSession:             "session",
		zk.EventNotWatching:         "not-watching",
	}
	states = map[zk.State]string{
		zk.StateUnknown:           "state-unknown",
		zk.StateDisconnected:      "state-disconnected",
		zk.StateAuthFailed:        "state-auth-failed",
		zk.StateConnectedReadOnly: "state-connected-readonly",
		zk.StateSaslAuthenticated: "state-sasl-authenticated",
		zk.StateExpired:           "state-expired",
		zk.StateConnected:         "state-connected",
		zk.StateHasSession:        "state-has-session",
	}
)

func (e Event) AsMap() map[string]interface{} {
	if e.Action != "" {
		return map[string]interface{}{
			"type":   e.Action,
			"note":   e.Note,
			"path":   e.Path,
			"error":  e.Err,
			"server": e.Server,
		}
	} else {
		return map[string]interface{}{
			"type":   event_types[e.Type],
			"state":  states[e.State],
			"path":   e.Path,
			"error":  e.Err,
			"server": e.Server,
		}
	}
}

func (e Event) JSON() string {
	buff, _ := json.Marshal(e.AsMap())
	return string(buff)
}
