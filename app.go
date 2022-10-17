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

var (
	state   = uuid.New().String()
	port    = getEnvOrFallback("PORT", "8080")
	baseUrl = getEnvOrFallback("BASE_URL", "http://localhost:8080")
	auth    = spotifyauth.New(spotifyauth.WithRedirectURL(baseUrl+"/callback"), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadCollaborative, spotifyauth.ScopePlaylistModifyPrivate, spotifyauth.ScopePlaylistReadPrivate))
)

func main() {
	setupHandlers()

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func setupHandlers() {
	http.HandleFunc("/", rootPath)

	http.HandleFunc("/login", loginPath)
	http.HandleFunc("/callback", completeAuthPath)

	http.HandleFunc("/home", authenticated(homePath))
	http.HandleFunc("/backup", authenticated(backupPath))
}

func getEnvOrFallback(name string, fallback string) string {
	val := os.Getenv(name)

	if val != "" {
		return val
	}

	return fallback
}

func rootPath(w http.ResponseWriter, r *http.Request) {
	cookie := getCookie(r)

	if cookie != nil {
		http.Redirect(w, r, "/home", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	}
}

func homePath(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) {
	data := getData(cookie)
	t.ExecuteTemplate(w, "index.logged-in.html.tmpl", data)
}

func backupPath(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) {
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
	items, err := client.GetPlaylistItems(ctx, playlistId)
	if err != nil {
		panic("Failed to get GetPlaylistTracks")
	}

	for page := 1; ; page++ {
		var trackIds []spotify.ID
		for _, item := range items.Items {
			if item.Track.Track != nil {
				trackIds = append(trackIds, item.Track.Track.ID)
			}
		}

		_, err := client.AddTracksToPlaylist(ctx, newPlaylist.ID, trackIds...)

		if err != nil {
			panic("Failed to add to playlist")
		}

		err = client.NextPage(ctx, items)
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
