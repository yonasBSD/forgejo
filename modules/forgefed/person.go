package forgefed

import (
	"fmt"

	ap "github.com/go-ap/activitypub"
	"github.com/valyala/fastjson"
)

func CreatePersonFromParsedJson(parsed *fastjson.Value) (*ap.Person, error) {

	iriString := ap.JSONGetString(parsed, "id")
	preferredUsername := ap.JSONGetString(parsed, "preferredUsername")
	url := ap.JSONGetString(parsed, "url")
	icon := ap.JSONGetString(parsed.Get("icon"), "url")
	inbox := ap.JSONGetString(parsed, "inbox")
	outbox := ap.JSONGetString(parsed, "outbox")
	publicKeyId := ap.JSONGetString(parsed.Get("publicKey"), "id")
	publicKeyOwner := ap.JSONGetString(parsed.Get("publicKey"), "owner")
	publicKeyPem := ap.JSONGetString(parsed.Get("publicKey"), "publicKeyPem")

	person := *ap.PersonNew(ap.IRI(iriString))

	person.Name = ap.NaturalLanguageValuesNew()
	err := person.Name.Set("en", ap.Content(preferredUsername))
	if err != nil {
		return ap.PersonNew(""), fmt.Errorf("set name: %v", err)
	}

	person.URL = ap.IRI(url)

	person.Icon = ap.Image{
		Type:      ap.ImageType,
		MediaType: "image/png",
		URL:       ap.IRI(icon),
	}

	person.Inbox = ap.IRI(inbox)
	person.Outbox = ap.IRI(outbox)

	person.PublicKey.ID = ap.IRI(publicKeyId)
	person.PublicKey.Owner = ap.IRI(publicKeyOwner)
	person.PublicKey.PublicKeyPem = publicKeyPem

	return &person, nil
}

func ParsePersonJson(data []byte) (*ap.Person, error) {
	parser := fastjson.Parser{}
	parsed, err := parser.ParseBytes(data)
	if err != nil {
		return ap.PersonNew(""), err
	}

	return CreatePersonFromParsedJson(parsed)
}
