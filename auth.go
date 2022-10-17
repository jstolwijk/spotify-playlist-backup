package main

import (
	"context"
	"log"
	"net/http"

	spotify "github.com/zmb3/spotify/v2"
)

func completeAuthPath(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Println(err)
		return
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Printf("State mismatch: %s != %s\n", st, state)
		return
	}
	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))

	cookie := http.Cookie{
		Name:    "spotify",
		Value:   tok.AccessToken,
		Path:    "/",
		Domain:  baseUrl,
		Expires: tok.Expiry,
		Secure:  true,
	}
	http.SetCookie(w, &cookie)

	client.CurrentUser(context.Background())

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func loginPath(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"LoginUrl": auth.AuthURL(state),
	}

	t.ExecuteTemplate(w, "login.html.tmpl", data)
}

func authenticated(closure func(w http.ResponseWriter, r *http.Request, cookie *http.Cookie)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie := getCookie(r)
		if cookie != nil {
			closure(w, r, cookie)
		} else {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		}
	}
}

func getCookie(r *http.Request) *http.Cookie {
	for _, cookie := range r.Cookies() {
		if cookie.Name == "spotify" {
			return cookie
		}
	}
	return nil
}
