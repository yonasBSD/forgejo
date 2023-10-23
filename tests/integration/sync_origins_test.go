package integration

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestSyncOrigins(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	req := NewRequest(t, "GET", fmt.Sprintf("/"))
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", "/repo/setup_origin", map[string]string{
		"_csrf":           htmlDoc.GetCSRF(),
		"remote_username": "Codeberg.org",
		"type":            "codeberg_starred",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)
	unittest.AssertExistsAndLoadBean(t, &models.Origin{
		UserID:         user.ID,
		RemoteUsername: "Codeberg.org",
		Type:           "codeberg_starred",
	})

	req = NewRequestWithValues(t, "POST", "/repo/sync_origins/1", nil)
	session.MakeRequest(t, req, http.StatusSeeOther)
	time.Sleep(1 * time.Second)

	exist, err := repo.IsRepositoryModelExist(context.Background(), user, "tegfs")
	assert.NoError(t, err)
	assert.True(t, exist)

}
