package federation

import (
	"bufio"
	"bytes"
	"image/png"
	"io"
	"testing"

	"code.gitea.io/gitea/modules/avatar"

	ap "github.com/go-ap/activitypub"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

func TestGetPerson(t *testing.T) {
	defer gock.Off()

	username := "john"
	actorID := "https://example.com/john"

	person := ap.PersonNew(ap.IRI(actorID))
	_ = person.PreferredUsername.Set("en", ap.Content(username))
	gock.New("https://example.com").Get("/john").Times(1).Reply(200).JSON(person)
	respPerson, _ := GetActor(actorID)

	assert.Equal(t, respPerson.ID.String(), actorID)
	assert.Equal(t, respPerson.PreferredUsername.First().Value.String(), username)
}

func TestGetPersonAvatar(t *testing.T) {
	defer gock.Off()

	username := "john"
	actorID := "https://example.com/john"

	img, _ := avatar.RandomImage([]byte(actorID))

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	r := bufio.NewReader(&b)
	_ = png.Encode(w, img)

	gock.New("https://example.com").Get("/john/avatar").Times(1).Reply(200).Body(r)

	person := ap.PersonNew(ap.IRI(actorID))
	_ = person.PreferredUsername.Set("en", ap.Content(username))
	gock.New("https://example.com").Get("/john").Times(1).Reply(200).JSON(person)

	person.Icon = ap.Image{
		Type:      ap.ImageType,
		MediaType: "image/png",
		URL:       ap.IRI("https://example.com/john/avatar"),
	}

	res, _ := GetPersonAvatar(person)
	imgBytes, _ := io.ReadAll(r)
	assert.Equal(t, res, imgBytes)
}
