package kontrol

import (
	"errors"
	"strings"

	"github.com/coreos/etcd/store"
	"github.com/coreos/go-etcd/etcd"
)

// Storage is an interface to a kite storage.
type Storage interface {
	Get(key string) (*Node, error)
	Set(key, value string) error
	Delete(key string) error
	Watch(key string, index uint64) (*Watcher, error)
}

type Etcd struct {
	client *etcd.Client
}

type Watcher struct {
	recv chan *etcd.Response
	stop chan bool
}

func (w *Watcher) Stop() {
	w.stop <- true
}

func NewEtcd(machines []string) (*Etcd, error) {
	client := etcd.NewClient(machines)
	ok := client.SetCluster(machines)
	if !ok {
		return nil, errors.New("cannot connect to etcd cluster: " + strings.Join(machines, ","))
	}

	return &Etcd{
		client: client,
	}, nil
}

func (e *Etcd) Delete(key string) error {
	_, err := e.client.Delete(key, true)
	return err
}

func (e *Etcd) Set(key, value string) error {
	_, err := e.client.Set(key, value, uint64(HeartbeatDelay))
	return err
}

func (e *Etcd) Update(key, value string) error {
	_, err := e.client.Update(key, value, uint64(HeartbeatDelay))
	return err
}

func (e *Etcd) Watch(key string, index uint64) (*Watcher, error) {
	// TODO: make the buffer configurable
	responses := make(chan *etcd.Response, 1000)
	stopChan := make(chan bool, 1)

	// TODO: pass a stop channel for further customization
	_, err := e.client.Watch(key, index, true, responses, nil)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		recv: responses,
		stop: stopChan,
	}, nil
}

func (e *Etcd) Get(key string) (*Node, error) {
	resp, err := e.client.Get(key, false, true)
	if err != nil {
		return nil, err
	}

	nodeExtern := convertNodeToNodeExtern(resp.Node)

	return NewNode(nodeExtern), nil
}

func convertNodeToNodeExtern(node *etcd.Node) *store.NodeExtern {
	s := &store.NodeExtern{
		Key:           node.Key,
		Value:         &node.Value,
		Dir:           node.Dir,
		Expiration:    node.Expiration,
		TTL:           node.TTL,
		ModifiedIndex: node.ModifiedIndex,
		CreatedIndex:  node.CreatedIndex,
		Nodes:         make(store.NodeExterns, len(node.Nodes)),
	}

	for i, n := range node.Nodes {
		s.Nodes[i] = convertNodeToNodeExtern(n)
	}

	return s
}
