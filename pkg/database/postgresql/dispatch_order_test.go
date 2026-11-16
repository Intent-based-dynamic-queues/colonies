package postgresql

import (
	"math"
	"sort"
	"testing"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Dispatch order of WAITING processes is strict priority: PRIORITY descending,
// then submission order (PRIORITYTIME, which within one priority is the
// submission time in nanoseconds), then PROCESS_ID. Age never overtakes
// priority. Upstream ordered by PRIORITYTIME alone, where one priority unit is
// worth one day of waiting, so a job one level lower that had waited more than
// a day dispatched first.

// ageProcess moves a process's submission back by the given number of days,
// keeping PRIORITYTIME consistent with it.
func ageProcess(t *testing.T, db *PQDatabase, processID string, days int64) {
	_, err := db.postgresql.Exec(`UPDATE `+db.dbPrefix+`PROCESSES SET SUBMISSION_TIME = SUBMISSION_TIME - make_interval(days => $1::int), PRIORITYTIME = PRIORITYTIME - $2::bigint WHERE PROCESS_ID = $3`,
		days, days*86400*1000000000, processID)
	assert.Nil(t, err)
}

func addProcessWithPriority(t *testing.T, db *PQDatabase, colonyName string, priority int) *core.Process {
	p := utils.CreateTestProcess(colonyName)
	p.FunctionSpec.Conditions.ExecutorNames = nil
	p.FunctionSpec.Conditions.LocationName = ""
	p.FunctionSpec.Priority = priority
	assert.Nil(t, db.AddProcess(p))
	return p
}

func TestDispatchOrderPriorityBeatsAge(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), "test_colony_name")
	assert.Nil(t, db.AddColony(colony))
	executor := utils.CreateTestExecutor(colony.Name)
	executor.LocationName = ""
	assert.Nil(t, db.AddExecutor(executor))

	old := addProcessWithPriority(t, db, colony.Name, 1)
	ageProcess(t, db, old.ID, 2)
	newer := addProcessWithPriority(t, db, colony.Name, 2)

	waiting, err := db.FindWaitingProcesses(colony.Name, "", "", "", 10)
	assert.Nil(t, err)
	assert.Len(t, waiting, 2)
	assert.Equal(t, newer.ID, waiting[0].ID, "listing: higher priority first, however old the lower one is")

	selected, err := schedulerSelect(db, colony.Name, executor, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, newer.ID, selected.ID, "scheduler: higher priority first")

	assigned, err := db.SelectAndAssign(colony.Name, executor.ID, executor.Name, executor.Type, executor.LocationName,
		0, 0, 0, math.MaxInt8, math.MaxInt8, math.MaxInt8, 1)
	assert.Nil(t, err)
	assert.NotNil(t, assigned)
	assert.Equal(t, newer.ID, assigned.ID, "assign path: higher priority first")
}

func TestDispatchOrderLoweredJobStaysBehind(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), "test_colony_name")

	lowered := addProcessWithPriority(t, db, colony.Name, 25)
	ageProcess(t, db, lowered.ID, 3)
	results, err := db.SetProcessPriorities(colony.Name, []core.PriorityUpdate{{ProcessID: lowered.ID, Priority: 0}})
	assert.Nil(t, err)
	assert.Equal(t, core.PriorityUpdated, results[0].Outcome)
	routine := addProcessWithPriority(t, db, colony.Name, 1)

	waiting, err := db.FindWaitingProcesses(colony.Name, "", "", "", 10)
	assert.Nil(t, err)
	assert.Len(t, waiting, 2)
	assert.Equal(t, routine.ID, waiting[0].ID, "a job lowered to 0 stays behind priority 1, even after days")
}

func TestDispatchOrderTieBreakByProcessID(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony := core.CreateColony(core.GenerateRandomID(), "test_colony_name")

	var ids []string
	for i := 0; i < 4; i++ {
		p := addProcessWithPriority(t, db, colony.Name, 50)
		ids = append(ids, p.ID)
	}
	// Same priority and same submission instant: only the ID can decide.
	_, err = db.postgresql.Exec(`UPDATE `+db.dbPrefix+`PROCESSES SET PRIORITYTIME = 0 WHERE TARGET_COLONY_NAME = $1`, colony.Name)
	assert.Nil(t, err)

	waiting, err := db.FindWaitingProcesses(colony.Name, "", "", "", 10)
	assert.Nil(t, err)
	got := make([]string, 0, len(waiting))
	for _, p := range waiting {
		got = append(got, p.ID)
	}
	sort.Strings(ids)
	assert.Equal(t, ids, got, "ties in priority and submission are broken by process ID")
}
