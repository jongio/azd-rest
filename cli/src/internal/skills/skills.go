package skills

import (
	"embed"

	"github.com/jongio/azd-core/copilotskills"
	"github.com/jongio/azd-rest/src/internal/version"
)

//go:embed azd-rest/SKILL.md
var skillFS embed.FS

// InstallSkill installs the azd-rest skill to ~/.copilot/skills/azd-rest.
func InstallSkill() error {
	return copilotskills.Install("azd-rest", version.Version, skillFS, "azd-rest")
}
