package main

import (
	"fmt"
	"os"
)

func (a *API) findProject(name string) int {
	for _, v := range a.projects {
		if v.Name == name {
			return v.ProjectID
		}
	}
	return 0
}

func (a *API) findTask(name string) int {
	for _, v := range a.tasks {
		if v.Name == name {
			return v.TaskID
		}
	}
	return 0
}

func (a *API) clientProjects(id int) {
	for _, pr := range a.projects {
		if pr.ClientID == id {
			fmt.Printf("\tProject Name: %s\n", pr.Name)
		}

	}
}

func (a *API) pushFB(allItems Items, fbProject string, fbTask string) {
	for _, v := range allItems {
		te := &TimeEntry{
			ProjectID: a.findProject(fbProject),
			TaskID:    a.findTask(fbTask),
			UserID:    1,
			Date:      v.DueDate.Format("2006-01-02"),
			Notes:     fmt.Sprintf("%s: %s", v.Key.Val, v.Summary),
			Hours:     float64(v.TimeSpent.Seconds) / 60 / 60,
		}
		id, err := a.SaveTimeEntry(te)
		if err != nil {
			fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\tCreated Time Entry: ID:%d\n", id)
	}
}
