package prereqs

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/redist"
)

var installArgs = struct {
	plan *string
	pipe *string
}{}

var testArgs = struct {
	prereqs *[]string
}{}

func Register(ctx *mansion.Context) {
	{
		cmd := ctx.App.Command("install-prereqs", "Install prerequisites from an install plan").Hidden()
		installArgs.plan = cmd.Arg("plan", "Path of a .json file containing the plan to follow").Required().String()
		installArgs.pipe = cmd.Flag("pipe", "Named pipe where to write status updates").String()
		ctx.Register(cmd, doInstall)
	}

	{
		cmd := ctx.App.Command("test-prereqs", "Download and install a bunch of prerequisites from their names").Hidden()
		ctx.Register(cmd, doTest)
		testArgs.prereqs = cmd.Arg("prereqs", "Which prereqs to install (space-separated). Leave empty to get a list").Strings()
	}
}

func doInstall(ctx *mansion.Context) {
	ctx.Must(Install(ctx, *installArgs.plan, *installArgs.pipe))
}

func doTest(ctx *mansion.Context) {
	ctx.Must(Test(ctx, *testArgs.prereqs))
}

// PrereqTask describes something the prereq installer has to do
type PrereqTask struct {
	Name    string             `json:"name"`
	WorkDir string             `json:"workDir"`
	Info    redist.RedistEntry `json:"info"`
}

// PrereqPlan contains a list of tasks for the prereq installer
type PrereqPlan struct {
	Tasks []*PrereqTask `json:"tasks"`
}

// PrereqState informs the caller on the current status of a prereq
type PrereqState struct {
	Type   string            `json:"type"`
	Name   string            `json:"name"`
	Status buse.PrereqStatus `json:"status"`
}

// PrereqLogEntry sends an information to the caller on the progress of the task
type PrereqLogEntry struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
