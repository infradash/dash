package zk

import (
	"net/url"
	"path/filepath"
	"strconv"
)

func (this *Node) Id() url.URL {
	copy := this.client.Id()
	copy.Path = this.Path
	return copy
}

func (this *Node) Version() int32 {
	return this.Stats.Version
}

func (this *Node) Basename() string {
	return filepath.Base(this.Path)
}

func (this *Node) ValueString() string {
	return string(this.Value)
}

func (this *Node) Load() error {
	if err := this.client.check(); err != nil {
		return err
	}
	v, s, err := this.client.conn.Get(this.Path)
	if err != nil {
		return err
	}
	this.Value = v
	this.Stats = s
	return nil
}

func (this *Node) WatchOnce(f func(Event)) (chan<- bool, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	value, stat, event_chan, err := this.client.conn.GetW(this.Path)
	if err != nil {
		return nil, err
	}
	this.Value = value
	this.Stats = stat
	return runWatch(this.Path, f, event_chan)
}

func (this *Node) WatchOnceChildren(f func(Event)) (chan<- bool, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	members, stat, event_chan, err := this.client.conn.ChildrenW(this.Path)
	if err != nil {
		return nil, err
	}
	this.Members = members
	this.Stats = stat
	return runWatch(this.Path, f, event_chan)
}

func (this *Node) Set(value []byte) error {
	if err := this.client.check(); err != nil {
		return err
	}
	s, err := this.client.conn.Set(this.Path, value, this.Stats.Version)
	if err != nil {
		return err
	}
	this.Value = value
	this.Stats = s
	this.client.trackEphemeral(this, s.EphemeralOwner > 0)
	return nil
}

func (this *Node) CountChildren() int32 {
	if this.Stats == nil {
		if err := this.Load(); err != nil {
			return -1
		}
	}
	return this.Stats.NumChildren
}

func (this *Node) Children() ([]*Node, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	paths, s, err := this.client.conn.Children(this.Path)
	if err != nil {
		return nil, err
	} else {
		this.Stats = s
		children := make([]*Node, len(paths))
		for i, p := range paths {
			children[i] = &Node{Path: this.Path + "/" + p, client: this.client}
			if err := children[i].Load(); err != nil {
				return nil, err
			}
		}
		return children, nil
	}
}

func (this *Node) SubtreePaths() ([]string, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	list := make([]string, 0)

	children, err := this.Children()
	if err != nil {
		return nil, err
	}
	for _, n := range children {
		l, err := n.SubtreePaths()
		if err != nil {
			return nil, err
		}
		list = append_string_slices(list, l)
		list = append(list, n.Path)
	}
	return list, nil
}

func (this *Node) SubtreeNodes() ([]*Node, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	list := make([]*Node, 0)

	children, err := this.Children()
	if err != nil {
		return nil, err
	}

	this.Leaf = len(children) == 0

	for _, n := range children {
		l, err := n.SubtreeNodes()
		if err != nil {
			return nil, err
		}
		list = append_node_slices(list, l)
		list = append(list, n)
	}
	return list, nil
}

// Recursively go through all the children.  Apply filter for each node. If filter returns
// true for the particular node, this node (though not necessarily all its children) will be
// excluded.  This is useful for searching through all true by name or by whether it's a parent
// node or not.
func (this *Node) FilterSubtreeNodes(filter func(*Node) bool) ([]*Node, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	list := make([]*Node, 0)

	children, err := this.Children()
	if err != nil {
		return nil, err
	}

	this.Leaf = len(children) == 0

	for _, n := range children {
		l, err := n.FilterSubtreeNodes(filter)
		if err != nil {
			return nil, err
		}
		list = append_node_slices(list, l)
		if filter != nil && !filter(n) {
			list = append(list, n)
		}
	}
	return list, nil
}

func (this *Node) VisitSubtreeNodes(accept func(*Node) bool) ([]*Node, error) {
	if err := this.client.check(); err != nil {
		return nil, err
	}
	list := make([]*Node, 0)

	children, err := this.Children()
	if err != nil {
		return nil, err
	}

	this.Leaf = len(children) == 0
	for _, n := range children {
		l, err := n.VisitSubtreeNodes(accept)
		if err != nil {
			return nil, err
		}
		list = append_node_slices(list, l)
		if accept != nil && accept(n) {
			list = append(list, n)
		}
	}
	return list, nil
}

func (this *Node) Delete() error {
	if err := this.client.check(); err != nil {
		return err
	}
	err := this.client.DeleteNode(this.Path)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (this *Node) Increment(increment int) (int, error) {
	if err := this.client.check(); err != nil {
		return -1, err
	}
	count, err := strconv.Atoi(this.ValueString())
	if err != nil {
		count = 0
	}
	count += increment
	err = this.Set([]byte(strconv.Itoa(count)))
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (this *Node) CheckAndIncrement(current, increment int) (int, error) {
	if err := this.client.check(); err != nil {
		return -1, err
	}
	count, err := strconv.Atoi(this.ValueString())
	switch {
	case err != nil:
		return -1, err
	case count != current:
		return -1, ErrConflict
	}
	count += increment
	err = this.Set([]byte(strconv.Itoa(count)))
	if err != nil {
		return -1, err
	}
	return count, nil
}
