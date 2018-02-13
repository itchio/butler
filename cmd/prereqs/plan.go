package prereqs

import (
	"github.com/go-errors/errors"
)

func (pc *PrereqsContext) BuildPlan(names []string) (*PrereqPlan, error) {
	plan := &PrereqPlan{}

	for _, name := range names {
		entry, err := pc.GetEntry(name)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		plan.Tasks = append(plan.Tasks, &PrereqTask{
			Name:    name,
			WorkDir: pc.GetEntryDir(name),
			Info:    entry,
		})
	}

	return plan, nil
}
