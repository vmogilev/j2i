package main

import "fmt"

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
			fmt.Printf("\tProject ID: %d\tName: %s\n", pr.ProjectID, pr.Name)
		}

	}
}
