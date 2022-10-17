package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	spotify "github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

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
