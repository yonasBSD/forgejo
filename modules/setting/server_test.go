package setting

import (
	"testing"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestDisplayNameDefault(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "Beyond coding. We Forge.")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME}: {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", displayName)
}

func TestDisplayNameUnsetSlogan(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "unset")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME}: {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo", displayName)
}

func TestDisplayNameCustomFormat(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "Beyond coding. We Forge.")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME} - {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo - Beyond coding. We Forge.", displayName)
}