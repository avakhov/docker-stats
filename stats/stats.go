package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/avakhov/docker-stats/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"sync"
	"time"
)

var EXPIRE_AT = 5 * time.Minute

type jsonStats struct {
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`

	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`

	CpuStats struct {
		CpuUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCpuUsage uint64 `json:"system_cpu_usage"`
		OnlineCpus     uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
}

type Container struct {
	expiredAt time.Time
	ID        string
	Up        int
	MemUsed   uint64
	MemTotal  uint64
	CpuUsed   float64
	Labels    []string
}

type Stats struct {
	tick       int
	mu         sync.Mutex
	grabLabels []string
	containers map[string]Container
}

func NewStats(labels []string) *Stats {
	return &Stats{
		tick:       0,
		grabLabels: labels,
		containers: map[string]Container{},
	}
}

func (s *Stats) Run() {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	for {
		// prepare
		s.tick++
		fmt.Printf("[%s] stats tick: %d\n", time.Now().Format("2006-01-02 15:04:05"), s.tick)

		// step 1
		err := s.grabContainers(dockerClient)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}

		// step 2
		err = s.cleanExpired()
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}

		// sleep
		fmt.Printf("[%s] stats tick: %d - end\n", time.Now().Format("2006-01-02 15:04:05"), s.tick)
		time.Sleep(10 * time.Second)
	}
}

func (s *Stats) grabContainers(dockerClient *client.Client) error {
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return util.WrapError(err)
	}

	// prepare
	s.mu.Lock()
	current := map[string]Container{}
	for _, c := range containers {
		var container Container
		// get
		var ok bool
		container, ok = s.containers[c.ID]
		if !ok {
			container = Container{
				ID:        c.ID,
				expiredAt: time.Now().Add(EXPIRE_AT),
			}
		}
		container.Labels = make([]string, len(s.grabLabels))
		for i, label := range s.grabLabels {
			lab, ok := c.Labels[label]
			if ok {
				container.Labels[i] = lab
			} else {
				container.Labels[i] = "missed"
			}
		}
		if c.State == "running" {
			container.Up = 1
			container.expiredAt = time.Now().Add(EXPIRE_AT)
		} else {
			container.Up = 0
			container.MemUsed = 0
			container.MemTotal = 0
			container.CpuUsed = 0.0
		}
		current[c.ID] = container
	}
	s.mu.Unlock()

	// grab stats
	wg := sync.WaitGroup{}
	for id, _ := range current {
		if current[id].Up == 0 {
			continue
		}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			reader, err := dockerClient.ContainerStats(context.Background(), id, false)
			if err != nil {
				fmt.Printf("ERROR 1: %s\n", err.Error())
				return
			}
			defer reader.Body.Close()
			body, err := io.ReadAll(reader.Body)
			if err != nil {
				fmt.Printf("ERROR 2: %s\n", err.Error())
				return
			}
			stat := jsonStats{}
			err = json.Unmarshal(body, &stat)
			if err != nil {
				fmt.Printf("ERROR 3: %s\n", err.Error())
				return
			}
			c := current[id]
			c.MemUsed = stat.MemoryStats.Usage
			c.MemTotal = stat.MemoryStats.Limit
			cpuDelta := float64(stat.CpuStats.CpuUsage.TotalUsage - stat.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta := float64(stat.CpuStats.SystemCpuUsage - stat.PreCPUStats.SystemCPUUsage)
			if systemDelta > 0.0 && cpuDelta > 0.0 {
				c.CpuUsed = (cpuDelta / systemDelta) * float64(stat.CpuStats.OnlineCpus)
			} else {
				c.CpuUsed = 0.0
			}
			current[id] = c
		}(id)
	}
	wg.Wait()

	// fill data
	s.mu.Lock()
	ids := []string{}
	ids = append(ids, util.Keys(current)...)
	ids = append(ids, util.Keys(s.containers)...)
	ids = util.Uniq(ids)
	for _, id := range ids {
		c, ok := current[id]
		if !ok {
			c := s.containers[id]
			c.Up = 0
			c.MemUsed = 0
			c.MemTotal = 0
			c.CpuUsed = 0.0
			s.containers[id] = c
		} else {
			s.containers[id] = c
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *Stats) cleanExpired() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, c := range s.containers {
		if c.expiredAt.Before(time.Now()) {
			delete(s.containers, id)
		}
	}
	return nil
}

func (s *Stats) GetContainers() []Container {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Container{}
	for _, c := range s.containers {
		out = append(out, c)
	}
	return out
}
