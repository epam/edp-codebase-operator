package git

import (
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"log"
)

func CheckPermissions(repo string, user string, pass string) (accessible bool) {
	log.Printf("Checking permissions for user: %v in %v repository", user, repo)
	r, _ := git.Init(memory.NewStorage(), nil)
	remote, _ := r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo},
	})
	rfs, err := remote.List(&git.ListOptions{
		Auth: &http.BasicAuth{
			Username: user,
			Password: pass,
		}})
	if err != nil {
		log.Println(err)
		log.Printf("User %v do not have access to %v repository", user, repo)
		return false
	}
	return len(rfs) != 0
}

func CloneRepo(repo string, user string, pass string, destination string) error {
	_, err := git.PlainClone(destination, true, &git.CloneOptions{
		URL: repo,
		Auth: &http.BasicAuth{
			Username: user,
			Password: pass,
		}})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
