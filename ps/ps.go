package ps

import (
	"fmt"
	"github.com/avakhov/ext"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"time"
)

type Stats struct {
	tick int
	psEl int
	psAx int
}

func NewStats() *Stats {
	return &Stats{
		tick: 0,
	}
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func GetCounts() (int, int, error) {
	procs, err := ioutil.ReadDir("/proc")
	psAx := 0
	psEl := 0
	if err != nil {
		return 0, 0, ext.WrapError(err)
	}
	for _, p := range procs {
		if p.IsDir() && isNumeric(p.Name()) {
			pid := p.Name()
			taskPath := filepath.Join("/proc", pid, "task")
			tasks, err := ioutil.ReadDir(taskPath)
			if err != nil {
				continue // Task directory might not be accessible or exist if process exits
			}
			psAx += 1
			for _, t := range tasks {
				if isNumeric(t.Name()) {
					psEl += 1
				}
			}
		}
	}
	return psAx, psEl, nil
}

func (s *Stats) Run() {
	for {
		s.tick++
		fmt.Printf("[%s] stats tick: %d\n", time.Now().Format("2006-01-02 15:04:05"), s.tick)

		// get ticks
		ax, el, err := GetCounts()
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			s.psAx = 0
			s.psEl = 0
		} else {
			s.psAx = ax
			s.psEl = el
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Stats) GetPsAx() int {
	return s.psAx
}

func (s *Stats) GetPsEl() int {
	return s.psEl
}
