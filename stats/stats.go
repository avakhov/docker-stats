package stats

import (
	"context"
	"fmt"
	"github.com/avakhov/docker-stats/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
  "io"
	"sync"
	"time"
)

type Container struct {
	expiredAt time.Time
	ID        string
	Up        int
}

type Stats struct {
	tick       int
	mu         sync.Mutex
	containers map[string]Container
}

func NewStats() *Stats {
	return &Stats{
		tick:       0,
		containers: map[string]Container{},
	}
}

func (s *Stats) grabContainers(dockerClient *client.Client) error {
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return util.WrapError(err)
	}
	for _, c := range containers {
    id := c.ID[:8]
    var container Container
    // get
    {
      s.mu.Lock()
      defer s.mu.Unlock()
      var ok bool
      container, ok = s.containers[id]
      if !ok {
        container = Container{
          ID:        id,
          expiredAt: time.Now().Add(3 * time.Minute),
          Up:        0,
        }
      }
    }

    // read info
		if c.State == "running" {
			container.Up = 1
			container.expiredAt = time.Now().Add(3 * time.Minute)
		} else {
			container.Up = 0
		}
    reader, err := dockerClient.ContainerStats(context.Background(), c.ID, false)
    if err != nil {
      return util.WrapError(err)
    }
    defer reader.Body.Close() 
    body, err := io.ReadAll(reader.Body)
    if err != nil {
      return util.WrapError(err)
    }
    fmt.Printf("body: %s\n", string(body))

    // set
    {
      s.mu.Lock()
      defer s.mu.Unlock()
      s.containers[id] = container
    }
	}
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

func (s *Stats) Run() {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	for {
		s.tick++
		fmt.Printf("stats tick: %d\n", s.tick)
		err := s.grabContainers(dockerClient)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
		err = s.cleanExpired()
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
		time.Sleep(5 * time.Second)
	}
}
