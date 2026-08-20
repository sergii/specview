package web

import "runtime/debug"

func (workspaceData) Revision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	revision := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return "dev"
	}
	if len(revision) > 8 {
		revision = revision[:8]
	}
	if modified {
		revision += "+dirty"
	}
	return revision
}
