package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Post struct {
	ID        int
	Title     string
	Body      string
	CreatedAt time.Time
}

var (
	posts   []Post
	nextID  int = 1
	mu      sync.RWMutex
)

func main() {
	http.HandleFunc("GET /", handleIndex)
	http.HandleFunc("GET /post/{id}", handleShowPost)
	http.HandleFunc("GET /new", handleNewForm)
	http.HandleFunc("POST /new", handleCreatePost)

	log.Println("blog listening on :9984")
	log.Fatal(http.ListenAndServe(":9984", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html><head><title>Blog</title></head><body>`)
	fmt.Fprint(w, `<h1>Blog</h1><p><a href="/new">New Post</a></p>`)
	if len(posts) == 0 {
		fmt.Fprint(w, `<p>No posts yet.</p>`)
	}
	for i := len(posts) - 1; i >= 0; i-- {
		p := posts[i]
		fmt.Fprintf(w, `<div><h2><a href="/post/%d">%s</a></h2><small>%s</small></div>`,
			p.ID, p.Title, p.CreatedAt.Format("2006-01-02 15:04"))
	}
	fmt.Fprint(w, `</body></html>`)
}

func handleShowPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	mu.RLock()
	defer mu.RUnlock()

	for _, p := range posts {
		if fmt.Sprint(p.ID) == id {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><title>%s</title></head><body>`, p.Title)
			fmt.Fprintf(w, `<h1>%s</h1><small>%s</small><p>%s</p>`,
				p.Title, p.CreatedAt.Format("2006-01-02 15:04"), p.Body)
			fmt.Fprint(w, `<p><a href="/">Back</a></p></body></html>`)
			return
		}
	}
	http.NotFound(w, r)
}

func handleNewForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html><head><title>New Post</title></head><body>
<h1>New Post</h1>
<form method="POST" action="/new">
<p><label>Title: <input name="title" required></label></p>
<p><label>Body:<br><textarea name="body" rows="10" cols="60" required></textarea></label></p>
<p><button type="submit">Create</button></p>
</form>
<p><a href="/">Back</a></p>
</body></html>`)
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	body := r.FormValue("body")

	mu.Lock()
	p := Post{
		ID:        nextID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
	nextID++
	posts = append(posts, p)
	mu.Unlock()

	http.Redirect(w, r, fmt.Sprintf("/post/%d", p.ID), http.StatusSeeOther)
}
