package app

// PanelViewerStatus contains observation fields only. It deliberately avoids
// goal/behaviour inspection and owner, configuration, chat and audit data.
func (a *App) PanelViewerStatus() any {
	st := a.status()
	out := map[string]any{"connected": st.Connected, "character": st.Character, "phase": st.Phase, "connection": a.Session.ConnectionID(), "perceptionRadius": a.Config.Perception.Radius}
	if s := a.Session.State(); st.Connected && s != nil {
		out["world"], out["worldUid"], out["yaw"] = s.World, s.WorldUID, s.Yaw
		if s.Phase == "in_world" {
			out["pos"], out["biome"], out["location"], out["flags"], out["time"] = s.Pos, s.Biome, s.Location, s.Flags, s.Time
		}
	}
	return out
}
func (a *App) ViewerMapSnapshot() any {
	out := a.MapSnapshot().(MapSnapshot)
	out.Containers = nil
	out.SurveyLegs = nil
	return out
}
