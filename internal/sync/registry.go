package sync

import (
	"strconv"

	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
)

type Registry struct {
	providers map[syncstore.SyncType]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[syncstore.SyncType]Provider),
	}
}

func (r *Registry) Register(p Provider) {
	if _, exists := r.providers[p.Type()]; exists {
		panic("sync: duplicate provider registration for type " + strconv.Itoa(int(p.Type())))
	}
	r.providers[p.Type()] = p
}

func (r *Registry) Get(t syncstore.SyncType) (Provider, bool) {
	p, ok := r.providers[t]
	return p, ok
}
