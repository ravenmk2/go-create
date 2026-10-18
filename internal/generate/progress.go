package generate

// Step identifies a generation phase, in execution order.
type Step int

const (
	StepRender Step = iota + 1
	StepWrite
	StepGit
	StepTidy
	StepCount = int(StepTidy)
)

func (s Step) String() string {
	switch s {
	case StepRender:
		return "Render template"
	case StepWrite:
		return "Write files"
	case StepGit:
		return "Initialize Git"
	case StepTidy:
		return "Resolve dependencies"
	default:
		return "Generate project"
	}
}

// Status is a phase's current or terminal state.
type Status string

const (
	Running   Status = "running"
	Completed Status = "completed"
	Skipped   Status = "skipped"
	Warning   Status = "warning"
	Failed    Status = "failed"
)

// Progress is emitted synchronously as work happens. Observers must not block
// or mutate the generator's inputs. No terminal formatting belongs here.
type Progress struct {
	Step     Step
	Status   Status
	Message  string
	Warnings []string
}
