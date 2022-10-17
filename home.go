package main

import (
	"context"
	"net/http"
	"os"

	spotify "github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

func homePath(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) {
	data := getData(cookie)
	t.ExecuteTemplate(w, "index.logged-in.html.tmpl", data)
}

func getData(cookie *http.Cookie) map[string]string {
	tok := oauth2.Token{AccessToken: cookie.Value}
	client := spotify.New(auth.Client(context.Background(), &tok))

	user, err := client.CurrentUser(context.Background())

	var username = user.DisplayName

	if err != nil {
		username = err.Error()
	}

	return map[string]string{
		"Region":   os.Getenv("FLY_REGION"),
		"username": username,
	}
}
