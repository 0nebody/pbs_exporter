package pbsjob

import (
	"log/slog"
	"sync"
	"time"
)

// PBS job with its metadata.
type PbsJob struct {
	expiration int64
	isRunning  bool
	job        *Job
}

func (j *PbsJob) JobId() string {
	return j.job.JobId()
}

func (j *PbsJob) IsRunning() bool {
	return j.isRunning
}

// JobCache is a thread-safe cache for PBS jobs.
type JobCache struct {
	jobs    map[string]*PbsJob
	logger  *slog.Logger
	mu      *sync.RWMutex
	timeout int64
}

func NewJobCache(logger *slog.Logger, timeout int64, cleanInterval time.Duration) *JobCache {
	jobCache := &JobCache{
		jobs:    make(map[string]*PbsJob),
		logger:  logger,
		mu:      &sync.RWMutex{},
		timeout: timeout,
	}
	jobCache.StartCleanup(cleanInterval)

	return jobCache
}

func (c *JobCache) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.cleanup()
		}
	}()
}

// Remove expired jobs from the cache.
// Expects Delete to be called before they are removed.
func (c *JobCache) cleanup() {
	now := time.Now().Unix()

	c.mu.Lock()
	defer c.mu.Unlock()

	for jobId, job := range c.jobs {
		if !job.isRunning && job.expiration < now {
			c.logger.Debug("Cleanup: Deleting job from cache", "jobId", jobId, "expiration", job.expiration, "now", now)
			delete(c.jobs, jobId)
		}
	}
}

func (c *JobCache) List() []Job {
	var activeJobs []Job
	now := time.Now().Unix()

	c.mu.RLock()
	defer c.mu.RUnlock()

	jobCount := len(c.jobs)
	for _, job := range c.jobs {
		if job.expiration >= now {
			activeJobs = append(activeJobs, *job.job)
		}
	}
	c.logger.Debug("List: Jobs in cache", "count", jobCount, "active", len(activeJobs))

	return activeJobs
}

func (c *JobCache) Get(jobId string) (Job, bool) {
	now := time.Now().Unix()

	c.mu.RLock()
	defer c.mu.RUnlock()

	pbsJob, exists := c.jobs[jobId]
	if !exists {
		c.logger.Debug("Get: Job not found in cache", "jobId", jobId)
		return Job{}, false
	}

	if pbsJob.expiration < now {
		c.logger.Debug("Get: Job expired in cache", "jobId", jobId, "expiration", pbsJob.expiration, "now", now)
		return Job{}, false
	}

	c.logger.Debug("Get: Job found in cache", "jobId", jobId, "expiration", pbsJob.expiration, "now", now)

	return *pbsJob.job, exists
}

func (c *JobCache) IsRunning(jobId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	job, exists := c.jobs[jobId]

	return exists && job.isRunning
}

func (c *JobCache) Set(jobId string, job *Job) {
	now := time.Now().Unix()

	if job == nil {
		c.logger.Debug("Set: Job is nil, not setting in cache", "jobId", jobId)
		return
	}

	if jobId == "" {
		c.logger.Debug("Set: JobId is empty, not setting in cache")
		return
	}

	// Expiration should always be updated; qalter can modify requested walltime.
	expiration := job.Stime + job.RequestedWalltime() + c.timeout
	if expiration < now {
		c.logger.Debug("Set: Job expiration is in the past, not setting in cache", "jobId", jobId, "expiration", expiration, "now", now)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("Set: Job set in cache", "jobId", jobId, "expiration", expiration, "isRunning", true)
	c.jobs[jobId] = &PbsJob{
		expiration: expiration,
		isRunning:  true,
		job:        job,
	}
}

func (c *JobCache) Delete(jobId string) {
	now := time.Now().Unix()

	c.mu.Lock()
	defer c.mu.Unlock()

	if job, exists := c.jobs[jobId]; exists {
		if job.isRunning {
			c.logger.Debug("Delete: Job is still running, updating expiration", "jobId", jobId, "expiration", job.expiration, "now", now)
			job.isRunning = false
			job.expiration = now + c.timeout
		} else if job.expiration < now {
			c.logger.Debug("Delete: Deleting job from cache", "jobId", jobId, "expiration", job.expiration, "now", now)
			delete(c.jobs, jobId)
		}
	}
}
