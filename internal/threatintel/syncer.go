package threatintel

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

// ThreatFeed is an interface for threat intelligence providers.
type ThreatFeed interface {
	Name() string
	Fetch(ctx context.Context) ([]string, error)
}

// IPStore provides IP list persistence for the syncer.
type IPStore interface {
	AddIPEntry(e model.IPEntry) error
	ListIPEntries(listType string) ([]model.IPEntry, error)
}

// Syncer periodically fetches threat intelligence and syncs to the IP blacklist.
type Syncer struct {
	feed      ThreatFeed
	store     IPStore
	onChanged func()
	mu        sync.Mutex
	lastSync  time.Time
	lastCount int
	stopCh    chan struct{}
	running   bool
}

func NewSyncer(feed ThreatFeed, store IPStore, onChanged func()) *Syncer {
	return &Syncer{
		feed:      feed,
		store:     store,
		onChanged: onChanged,
		stopCh:    make(chan struct{}),
	}
}

func (s *Syncer) Start(interval time.Duration) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Sync()
			case <-s.stopCh:
				return
			}
		}
	}()
	log.Printf("threatintel: syncer started with interval %v", interval)
}

func (s *Syncer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// Sync performs a single fetch-and-store cycle.
func (s *Syncer) Sync() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ips, err := s.feed.Fetch(ctx)
	if err != nil {
		log.Printf("threatintel sync error: %v", err)
		return
	}

	for _, ip := range ips {
		entry := model.IPEntry{
			ID:        uuid.New().String(),
			IPAddress: ip,
			ListType:  "blacklist",
			Note:      "威胁情报自动同步 - " + s.feed.Name(),
		}
		if err := s.store.AddIPEntry(entry); err != nil {
			log.Printf("threatintel: failed to add IP %s: %v", ip, err)
		}
	}

	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastCount = len(ips)
	s.mu.Unlock()

	log.Printf("threatintel: synced %d IPs from %s", len(ips), s.feed.Name())

	if s.onChanged != nil {
		s.onChanged()
	}
}

func (s *Syncer) Status() (lastSync time.Time, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSync, s.lastCount
}
