package gateway

import (
	"sync"

	"github.com/google/uuid"
)

type auditJob struct {
	AuditID     uuid.UUID
	SessionID   uuid.UUID
	FileName    string
	ContentType string
	Redacted    []byte
	releaseOnce sync.Once
	release     func()
}

func (j *auditJob) Release() {
	if j == nil || j.release == nil {
		return
	}
	j.releaseOnce.Do(j.release)
}

type jobStore struct {
	mu   sync.Mutex
	jobs map[uuid.UUID]*auditJob
}

func newJobStore() *jobStore {
	return &jobStore{jobs: make(map[uuid.UUID]*auditJob)}
}

func (s *jobStore) put(job *auditJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.AuditID] = job
}

func (s *jobStore) take(id uuid.UUID) (*auditJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if ok {
		delete(s.jobs, id)
	}
	return job, ok
}

func (s *jobStore) drop(id uuid.UUID) *auditJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	delete(s.jobs, id)
	return job
}
