package prereqs

import (
	"github.com/pkg/errors"
)

func (h *handler) BuildPlan(names []string) (*PrereqPlan, error) {
	plan := &PrereqPlan{}

	for _, name := range names {
		entry, err := h.GetEntry(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		plan.Tasks = append(plan.Tasks, &PrereqTask{
			Name:    name,
			WorkDir: h.GetEntryDir(name),
			Info:    entry,
		})
	}

	return plan, nil
}
