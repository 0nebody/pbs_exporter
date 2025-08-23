package pbsjobs

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestPbsJobJobId(t *testing.T) {
	pbsJob := &PbsJob{
		job: &Job{},
	}

	tests := []struct {
		hashname string
		want     string
	}{
		{"", ""},
		{"12345", "12345"},
		{"12345.pbs", "12345"},
	}
	for _, test := range tests {
		pbsJob.job.Hashname = test.hashname
		got := pbsJob.JobId()
		if got != test.want {
			t.Errorf("JobId() = %v, want %v", got, test.want)
		}
	}
}

func TestPbsJobIsRunning(t *testing.T) {
	pbsJob := &PbsJob{
		job: &Job{
			Hashname: "12345.pbs",
		},
	}

	tests := []struct {
		isRunning bool
		want      bool
	}{
		{true, true},
		{false, false},
	}
	for _, test := range tests {
		pbsJob.isRunning = test.isRunning
		got := pbsJob.IsRunning()
		if got != test.want {
			t.Errorf("IsRunning() = %v, want %v", got, test.want)
		}
	}
}

func loadTestJobCache() *JobCache {
	now := time.Now().Unix()

	return &JobCache{
		jobs: map[string]*PbsJob{
			// Job 1000 is running and not expired
			"1000": {
				expiration: now + 60,
				isRunning:  true,
				job: &Job{
					Hashname: "1000.pbs",
					Stime:    now,
					ResourceList: ResourceList{
						Walltime: "00:00:01",
					},
				},
			},
			// Job 1001 is not running and has expired
			"1001": {
				expiration: now - 600,
				isRunning:  false,
				job: &Job{
					Hashname: "1001.pbs",
					Stime:    now - 1000,
					ResourceList: ResourceList{
						Walltime: "00:00:01",
					},
				},
			},
			// Job 1002 is not running expiry is equal to now
			"1002": {
				expiration: now,
				isRunning:  false,
				job: &Job{
					Hashname: "1002.pbs",
					Stime:    now - 60,
					ResourceList: ResourceList{
						Walltime: "00:00:01",
					},
				},
			},
		},
		logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
		mu:      &sync.RWMutex{},
		timeout: 60,
	}
}

func TestJobCacheCleanup(t *testing.T) {
	now := time.Now().Unix()
	jobCache := loadTestJobCache()

	t.Run("Normal Cleanup", func(t *testing.T) {
		jobCache.cleanup()
		if len(jobCache.jobs) != 2 {
			t.Fatalf("After cleanup job count = %d, want 2", len(jobCache.jobs))
		}
		if _, exists := jobCache.jobs["1001"]; exists {
			t.Error("Job 1001 should have been removed")
		}
	})

	// PBS removes jobs exceeding walltime periodically, ensure cleanup respects running jobs
	t.Run("Cleanup walltime race", func(t *testing.T) {
		jobCache.jobs["1000"].isRunning = true
		jobCache.jobs["1000"].expiration -= 86400
		if !(jobCache.jobs["1000"].expiration < now) {
			t.Error("Job 1000 should be expired")
		}
		jobCache.cleanup()
		if _, exists := jobCache.jobs["1000"]; !exists {
			t.Error("Job 1000 should still exist")
		}
	})
}

func TestJobCacheList(t *testing.T) {
	jobCache := loadTestJobCache()

	jobs := jobCache.List()
	if len(jobs) != 2 {
		t.Errorf("List() = %d jobs, want 2 jobs", len(jobs))
	}

	keys := make([]string, 0, len(jobs))
	for _, job := range jobs {
		keys = append(keys, job.JobId())
	}
	slices.Sort(keys)
	if keys[0] != "1000" || keys[1] != "1002" {
		t.Errorf("List() keys = %v, want [1000 1002]", keys)
	}
}

func TestJobCacheGet(t *testing.T) {
	jobCache := loadTestJobCache()

	tests := []struct {
		name       string
		jobId      string
		wantJob    *Job
		wantExists bool
	}{
		{
			"Existing Job",
			"1000",
			jobCache.jobs["1000"].job,
			true,
		},
		{
			"Expired Job",
			"1001",
			jobCache.jobs["1001"].job,
			false,
		},
		{
			"Job with Expiration Equal to Now",
			"1002",
			jobCache.jobs["1002"].job,
			true,
		},
		{
			"Non-existing Job",
			"9999",
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, exists := jobCache.Get(tt.jobId)
			if exists != tt.wantExists {
				t.Errorf("Get(%s) = %v, want %v", tt.jobId, exists, tt.wantExists)
			}
			if exists && job.Hashname != tt.wantJob.Hashname {
				t.Errorf("Get(%s) = %v, want %v", tt.jobId, job.Hashname, tt.wantJob.Hashname)
			}
		})
	}
}

func TestJobCacheSet(t *testing.T) {
	now := time.Now().Unix()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jobCache := NewJobCache(logger, 60, 15*time.Second)

	tests := []struct {
		name       string
		jobId      string
		job        *Job
		wantExists bool
	}{
		{
			"Job Running",
			"1000",
			&Job{
				Hashname: "1000.pbs",
				Stime:    now,
				ResourceList: ResourceList{
					Walltime: "00:00:01",
				},
			},
			true,
		},
		{
			"Job Expired",
			"1001",
			&Job{
				Hashname: "1001.pbs",
				Stime:    now - 1001,
				ResourceList: ResourceList{
					Walltime: "00:00:01",
				},
			},
			false,
		},
		{
			"Empty Job ID",
			"",
			&Job{},
			false,
		},
		{
			"Uninitialised Job",
			"1003",
			&Job{},
			false,
		},
		{
			"Nil Job",
			"1004",
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobCache.Set(tt.jobId, tt.job)
			_, exists := jobCache.jobs[tt.jobId]
			if exists != tt.wantExists {
				t.Errorf("Set(%s) = %v, want %v", tt.jobId, exists, tt.wantExists)
			}
		})
	}
}

func TestJobCacheDelete(t *testing.T) {
	jobCache := loadTestJobCache()
	tests := []struct {
		name       string
		jobId      string
		wantExists bool
	}{
		{
			"Delete unexpired job",
			"1000",
			true,
		},
		{
			"Delete expired job",
			"1001",
			false,
		},
		{
			"Delete job with Expiration Equal to Now",
			"1002",
			true,
		},
		{
			"Delete non-existing job",
			"9999",
			false,
		},
		{
			"Delete empty job ID",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobCache.Delete(tt.jobId)
			_, exists := jobCache.jobs[tt.jobId]
			if exists != tt.wantExists {
				t.Errorf("Delete(%s) = %v, want %v", tt.jobId, exists, tt.wantExists)
			}
		})
	}
}

func TestJobCacheRace(t *testing.T) {
	now := time.Now().Unix()
	jobCache := NewJobCache(slog.Default(), 60, 15*time.Second)
	jobCacheSize := 1000

	// generate job cache
	for i := 0; i < jobCacheSize; i++ {
		jobId := fmt.Sprintf("%d", i)
		expiration := now + 600
		if i%2 == 0 {
			expiration = now - 600
		}
		jobCache.jobs[jobId] = &PbsJob{
			expiration: expiration,
			isRunning:  i%3 == 0,
			job: &Job{
				Hashname: jobId + ".pbs",
				Stime:    now,
			},
		}
	}

	// list of random job IDs
	var jobList []string
	for i := 0; i < 60; i++ {
		random := rand.Intn(jobCacheSize)
		jobId := fmt.Sprintf("%d", random)
		jobList = append(jobList, jobId)
	}

	// run concurrent operations on the job cache
	batchSize := 3
	var wg sync.WaitGroup
	for i := 0; i < len(jobList); i += batchSize {
		end := i + batchSize
		if end > len(jobList) {
			end = len(jobList)
		}
		batch := jobList[i:end]
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			for _, jobId := range batch {
				ops := []func(){
					func() { jobCache.Get(jobId) },
					func() { jobCache.Set(jobId, &Job{Hashname: jobId + ".pbs", Stime: now}) },
					func() { jobCache.Delete(jobId) },
					func() { jobCache.List() },
				}
				rand.Shuffle(len(ops), func(i, j int) { ops[i], ops[j] = ops[j], ops[i] })
				for _, op := range ops {
					op()
				}
			}
		}(batch)
	}
	wg.Wait()
}
