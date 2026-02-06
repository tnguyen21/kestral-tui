package theme

// Agent status icons with semantic colors.
var (
	IconWorking = PassStyle.Render("●")
	IconStale   = WarnStyle.Render("◐")
	IconStuck   = FailStyle.Render("○")
	IconIdle    = MutedStyle.Render("○")
)

// Role icons for agent types.
const (
	RoleMayor    = "👑"
	RoleWitness  = "👁"
	RoleRefinery = "🔧"
	RolePolecat  = "🦨"
	RoleCrew     = "👷"
)
