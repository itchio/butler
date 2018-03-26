package prereqs

import (
	"github.com/pkg/errors"
)

func (pc *PrereqsContext) BuildPlan(names []string) (*PrereqPlan, error) {
	plan := &PrereqPlan{}

	for _, name := range names {
		entry, err := pc.GetEntry(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		plan.Tasks = append(plan.Tasks, &PrereqTask{
			Name:    name,
			WorkDir: pc.GetEntryDir(name),
			Info:    entry,
		})
	}

	return plan, nil
}
