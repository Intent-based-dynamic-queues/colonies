package scheduler

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/database"
)

type byLowestPriorityTime []*core.Process

func (c byLowestPriorityTime) Len() int {
	return len(c)
}

// Less is the dispatch order, the same as the database's ORDER BY PRIORITY
// DESC, PRIORITYTIME ASC, PROCESS_ID ASC: strict priority, then submission
// order, then process ID. Age never overtakes priority.
func (c byLowestPriorityTime) Less(i, j int) bool {
	if c[i].FunctionSpec.Priority != c[j].FunctionSpec.Priority {
		return c[i].FunctionSpec.Priority > c[j].FunctionSpec.Priority
	}
	if c[i].PriorityTime != c[j].PriorityTime {
		return c[i].PriorityTime < c[j].PriorityTime
	}
	return c[i].ID < c[j].ID
}

func (c byLowestPriorityTime) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type Scheduler struct {
	db database.ProcessLookup
}

func CreateScheduler(db database.ProcessLookup) *Scheduler {
	return &Scheduler{db: db}
}

func (scheduler *Scheduler) printCandidates(candidates []*core.Process) {
	for _, c := range candidates {
		fmt.Println(c.ID)
		fmt.Println(c.PriorityTime)
	}
}

func (scheduler *Scheduler) Select(colonyName string, executor *core.Executor, cpu int64, memory int64) (*core.Process, error) {
	prioritizedProcesses, err := scheduler.Prioritize(colonyName, executor, cpu, memory, 1)
	if err != nil {
		return nil, err
	}

	if len(prioritizedProcesses) < 1 {
		return nil, errors.New("No processes can be selected for executor with Id <" + executor.ID + ">")
	}

	return prioritizedProcesses[0], nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (scheduler *Scheduler) Prioritize(colonyName string, executor *core.Executor, cpu int64, memory int64, count int) ([]*core.Process, error) {

	// These conditions are not yet implemented
	storage := int64(0)
	nodes := math.MaxInt8
	processes := math.MaxInt8
	processesPerNode := math.MaxInt8

	candidates, err := scheduler.db.FindCandidatesByName(colonyName, executor.Name, executor.Type, executor.LocationName, cpu, memory, storage, nodes, processes, processesPerNode, count)
	if err != nil {
		return nil, err
	}

	candidates2, err := scheduler.db.FindCandidates(colonyName, executor.Type, executor.LocationName, cpu, memory, storage, nodes, processes, processesPerNode, count)
	if err != nil {
		return nil, err
	}

	candidates = append(candidates, candidates2...)

	if len(candidates) == 0 {
		return []*core.Process{}, nil
	}

	c := byLowestPriorityTime(candidates)
	sort.Sort(&c)
	return c[:min(count, len(candidates))], nil
}
