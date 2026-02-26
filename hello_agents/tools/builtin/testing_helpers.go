package builtin

func (t *DevLogTool) ExportGetSessionID() string      { return t.sessionID }
func (t *DevLogTool) ExportGetAgentName() string       { return t.agentName }
func (t *DevLogTool) ExportGetPersistenceDir() string  { return t.persistenceDir }
func (t *DevLogTool) ExportSetPersistenceDir(d string) { t.persistenceDir = d }

func (t *WriteTool) ExportResolvePath(path string) string     { return t.resolvePath(path) }
func (t *EditTool) ExportResolvePath(path string) string      { return t.resolvePath(path) }
func (t *MultiEditTool) ExportResolvePath(path string) string { return t.resolvePath(path) }
