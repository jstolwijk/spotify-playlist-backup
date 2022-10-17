package main

import "net/http"

func rootPath(w http.ResponseWriter, r *http.Request) {
	cookie := getCookie(r)

	if cookie != nil {
		http.Redirect(w, r, "/home", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	}
}
