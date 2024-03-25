package forgefed

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/log"
)

// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4

type WebfingerJRD struct {
	Subject    string           `json:"subject,omitempty"`
	Aliases    []string         `json:"aliases,omitempty"`
	Properties map[string]any   `json:"properties,omitempty"`
	Links      []*WebfingerLink `json:"links,omitempty"`
}

func (w WebfingerJRD) GetAvatar() *WebfingerLink {
	for _, link := range w.Links {
		if link.Rel == "http://webfinger.net/rel/avatar" {
			return link
		}
	}
	return nil
}

func (w WebfingerJRD) GetProfilePage() *WebfingerLink {
	for _, link := range w.Links {
		if link.Rel == "http://webfinger.net/rel/profile-page" && link.Type == "text/html" {
			return link
		}
	}
	return nil
}

func (w WebfingerJRD) GetActorLink() *WebfingerLink {
	for _, link := range w.Links {
		if link.Rel == "self" && link.Type == "application/activity+json" {
			return link
		}
	}
	return nil
}

type WebfingerLink struct {
	Rel        string            `json:"rel,omitempty"`
	Type       string            `json:"type,omitempty"`
	Href       string            `json:"href,omitempty"`
	Titles     map[string]string `json:"titles,omitempty"`
	Properties map[string]any    `json:"properties,omitempty"`
}

func GetHostnameFromResource(resource string) (string, error) {
	r := resource
	if strings.HasPrefix(resource, "@") {
		resource, _ = strings.CutPrefix(resource, "@")
	}
	actor, err := url.Parse(resource)
	if err != nil {
		return "", err
	}

	var hostname string
	switch actor.Scheme {
	case "":
		i := strings.Split(resource, "@")
		if len(i) != 2 {
			log.Error("Invalid webfinger query " + r)
			return "", errors.New("Invalid webfinger query " + r)
		}
		hostname = i[1]
	case "mailto":
		i := strings.Split(resource, "@")
		if len(i) != 2 {
			log.Error("Invalid webfinger query " + r)
			return "", errors.New("Invalid webfinger query " + r)
		}
		hostname = i[1]
	case "https":
		hostname = actor.Host
	default:
		log.Error("Invalid webfinger query " + r)
		return "", errors.New("Invalid webfinger query" + r)

	}
	return hostname, nil
}

// Get Actor object by performing webfinger lookup
func WebFingerLookup(q string) (*WebfingerJRD, error) {
	if strings.HasPrefix(q, "@") {
		q, _ = strings.CutPrefix(q, "@")
	}
	actor, err := url.Parse(q)
	if err != nil {
		return nil, err
	}

	var res string
	switch actor.Scheme {
	case "":
		res = fmt.Sprintf("acct:%s", q)
	case "mailto":
		res = q
	case "https":
		res = q
	default:
		return nil, errors.New("Invalid webfinger query")

	}

	hostname, err := GetHostnameFromResource(q)
	if err != nil {
		return nil, err
	}

	link := fmt.Sprintf("https://%s/.well-known/webfinger?resource=%s", hostname, res)

	r, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	webfingerResponse := new(WebfingerJRD)
	err = json.NewDecoder(r.Body).Decode(webfingerResponse)
	if err != nil {
		return nil, err
	}
	return webfingerResponse, nil
}

func IsFingerable(resource string) bool {
	if strings.HasPrefix(resource, "@") {
		resource, _ = strings.CutPrefix(resource, "@")
	}
	actor, err := url.Parse(resource)
	if err != nil {
		return false
	}

	switch actor.Scheme {
	case "":
		i := strings.Split(resource, "@")
		if len(i) == 2 {
			_ = i[1] // TODO: do len check before referencing element #2
			return true
		}
		return false
	case "mailto":
		i := strings.Split(resource, "@")
		if len(i) == 2 {
			_ = i[1]
			return true
		}
		return false
	case "https":
		return true
	default:
		return false
	}
}
