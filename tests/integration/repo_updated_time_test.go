package integration

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoUpdatedTime(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := "user2"
		session := loginUser(t, user)

		req := NewRequest(t, "GET", path.Join(user))
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		text := strings.TrimSpace(doc.doc.Find(".flex-item-body").First().Text())

		assert.Equal(t, "Updated 1970-01-01 01:00:00 +01:00", text)
	})
}
