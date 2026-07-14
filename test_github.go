package main

import (
	"fmt"
	"log"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase"
)

func main() {
	cfg := config.Load()
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: cfg.DataDirPath(),
	})

	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	users, _ := app.FindRecordsByFilter(db.CollUsers, "id != ''", "", 100, 0, nil)
	fmt.Printf("\nTotal users in DB: %d\n", len(users))
	for _, u := range users {
		token := u.GetString("github_access_token")
		fmt.Printf("- User ID: %s, Email: %s, HasToken: %v\n", u.Id, u.GetString("email"), token != "")
	}

	projects, _ := app.FindRecordsByFilter(db.CollProjects, "id != ''", "", 1000, 0, nil)
	fmt.Printf("Total projects in DB: %d\n", len(projects))
	for _, u := range users {
		count := 0
		for _, p := range projects {
			if p.GetString("owner") == u.Id {
				count++
			}
		}
		fmt.Printf("  -> User %s owns %d projects\n", u.Id, count)
	}
}
