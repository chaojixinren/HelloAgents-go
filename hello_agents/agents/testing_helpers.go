package agents

func (a *SimpleAgent) ExportBuildMessages(inputText string) []map[string]any {
	return a.buildMessages(inputText)
}

var ExportPlannerStepsFromArgs = plannerStepsFromArgs
