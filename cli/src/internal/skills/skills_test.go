package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallSkill_CreatesSkillDirectory(t *testing.T) {
	// Use a temporary HOME so we don't pollute the real filesystem.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows compat

	err := InstallSkill()
	require.NoError(t, err)

	// Verify the skill directory was created.
	skillDir := filepath.Join(tmpHome, ".copilot", "skills", "azd-rest")
	info, err := os.Stat(skillDir)
	require.NoError(t, err, "Skill directory should exist after install")
	assert.True(t, info.IsDir(), "Skill path should be a directory")
}

func TestInstallSkill_WritesSkillFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	err := InstallSkill()
	require.NoError(t, err)

	// Verify SKILL.md is written.
	skillFile := filepath.Join(tmpHome, ".copilot", "skills", "azd-rest", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	require.NoError(t, err, "SKILL.md should exist after install")
	assert.NotEmpty(t, content, "SKILL.md should have content")
	assert.Contains(t, string(content), "azd-rest", "SKILL.md should reference azd-rest")
}

func TestInstallSkill_Idempotent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Install twice - should not error.
	err := InstallSkill()
	require.NoError(t, err)

	err = InstallSkill()
	require.NoError(t, err, "Second install should succeed (idempotent)")
}

func TestSkillFS_EmbedValid(t *testing.T) {
	// Verify the embedded filesystem contains the expected file.
	entries, err := skillFS.ReadDir("azd-rest")
	require.NoError(t, err, "Embedded FS should have azd-rest directory")
	assert.NotEmpty(t, entries, "Embedded FS should contain files")

	// Find SKILL.md in the entries.
	found := false
	for _, entry := range entries {
		if entry.Name() == "SKILL.md" {
			found = true
			break
		}
	}
	assert.True(t, found, "Embedded FS should contain SKILL.md")
}
