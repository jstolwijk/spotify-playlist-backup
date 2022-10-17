package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	spotify "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

const redirectURI = "/callback"

var (
	auth    = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadCollaborative, spotifyauth.ScopePlaylistModifyPrivate, spotifyauth.ScopePlaylistReadPrivate))
	state   = uuid.New().String()
	baseUrl = "http://localhost:8080"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	bu := os.Getenv("BASE_URL")

	if bu != "" {
		baseUrl = bu
	}

	auth = spotifyauth.New(spotifyauth.WithRedirectURL(baseUrl+redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadCollaborative, spotifyauth.ScopePlaylistModifyPrivate, spotifyauth.ScopePlaylistReadPrivate))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, cookie := range r.Cookies() {
			if cookie.Name == "spotify" {
				data := getData(cookie)

				t.ExecuteTemplate(w, "index.logged-in.html.tmpl", data)
				return
			}
		}

		data := map[string]string{
			"LoginUrl": auth.AuthURL(state),
		}

		t.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	http.HandleFunc("/backup", func(w http.ResponseWriter, r *http.Request) {
		for _, cookie := range r.Cookies() {
			if cookie.Name == "spotify" {
				tok := oauth2.Token{AccessToken: cookie.Value}
				client := spotify.New(auth.Client(context.Background(), &tok))

				user, err := client.CurrentUser(context.Background())

				if err != nil {
					log.Println("Error: ", err)
					return
				}

				year, week := time.Now().ISOWeek()

				playlists, err := client.Search(context.Background(), "Discover Weekly", spotify.SearchTypePlaylist)

				if err != nil {
					log.Println("Error: ", err)
					return
				}

				if playlists.Playlists == nil || playlists.Playlists.Playlists == nil || len(playlists.Playlists.Playlists) == 0 {
					log.Println("Error: ", err)
					return
				}

				backupPlaylist(client, playlists.Playlists.Playlists[0].ID, fmt.Sprintf("Discover Weekly %d-%d", year, week), user.ID)

				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/callback", completeAuth)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func backupPlaylist(client *spotify.Client, playlistId spotify.ID, newPlaylistName string, userId string) {
	ctx := context.Background()

	playlist, err := client.GetPlaylist(ctx, playlistId)
	if err != nil {
		panic("Failed to get playlist")
	}
	newPlaylist, err := client.CreatePlaylistForUser(ctx, userId, newPlaylistName, "This is a backup of: \""+playlist.Name+"\" made on "+time.Now().Local().String(), false, false)

	if err != nil {
		panic("Failed to create playlist")
	}
	trackPage, err := client.GetPlaylistTracks(ctx, playlistId)
	if err != nil {
		panic("Failed to get GetPlaylistTracks")
	}

	for page := 1; ; page++ {
		var trackIds []spotify.ID
		for _, track := range trackPage.Tracks {
			trackIds = append(trackIds, track.Track.ID)
		}

		_, err := client.AddTracksToPlaylist(ctx, newPlaylist.ID, trackIds...)

		if err != nil {
			panic("Failed to add to playlist")
		}

		err = client.NextPage(ctx, trackPage)
		if err != nil && err.Error() == "spotify: no more pages" {
			break
		}
	}
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

func completeAuth(w http.ResponseWriter, r *http.Request) {
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
