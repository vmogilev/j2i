package main

import (
	"encoding/json"
	"fmt"
)

// Credit - https://github.com/pcrawfor/jira

// IssueService is a jira client with functions for operating on issues
type IssueService struct {
	client *Jira
}

// Issue is the type representing of a Jira Issue
type Issue struct {
	ID     string                 `json:"id,omitempty"`
	Key    string                 `json:"key,omitempty"`
	Self   string                 `json:"self,omitempty"`
	Expand string                 `json:"expand,omitempty"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// IssueList is the type representing of a list of Jira Issues as defined by their API response structure
type IssueList struct {
	Expand     string   `json:"expand,omitempty"`
	StartAt    int      `json:"starts_at,omitempty"`
	MaxResults int      `json:"max_results,omitempty"`
	Total      int      `json:"total,omitempty"`
	Issues     []*Issue `json:"issues,omitempty"`
	//Pagination *Pagination
}

// TransitionList is the type representing of a list of Jira transitions as defined by their API response structure
type TransitionList struct {
	Expand      string       `json:"expand,omitempty"`
	Transitions []Transition `json:"transitions,omitempty"`
}

// Transition is the type representing of a Jira Transition
type Transition struct {
	ID     string                 `json:"id,omitempty"`
	Name   string                 `json:"name,omitempty"`
	To     map[string]interface{} `json:"to,omitempty"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

var issueBasePath = restPath + "issue/"

// Label lables (sets/overwites) ISSUE with labelID
func (i *IssueService) Label(key, labelID string) ([]byte, error) {
	url := issueBasePath + key

	l := map[string]string{
		"add": labelID,
	}

	c := map[string]interface{}{
		"add": map[string]string{
			"body": "invoicebot: set label to: " + labelID,
		},
	}

	params := map[string]interface{}{
		"update": map[string]interface{}{
			"labels":  []interface{}{l},
			"comment": []interface{}{c},
		},
	}

	return i.client.execRequest(mPut, i.client.baseurl+url, params)
}

// Transition executes a transition for the given issue key to the given transition ID or returns an error
func (i *IssueService) Transition(key, transitionID string) ([]byte, error) {
	url := issueBasePath + key + "/transitions"

	// Comments don't work during transistions - moved them to Label()

	// c := map[string]interface{}{
	// 	"add": map[string]string{
	// 		"body": "invoicebot transition",
	// 	},
	// }

	// params := map[string]interface{}{
	// 	"update": map[string]interface{}{
	// 		"comment": []interface{}{c},
	// 	},
	// 	"transition": map[string]string{
	// 		"id": transitionID,
	// 	},
	// }

	params := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	return i.client.execRequest(mPost, i.client.baseurl+url, params)
}

// GetTransitions loads the available transitions for a given issue key
func (i *IssueService) GetTransitions(key string) (*TransitionList, error) {
	url := "issue/" + key + "/transitions?expand=transitions.fields"
	b, e := i.client.apiRequest(mGet, url, nil)
	if e != nil {
		return nil, e
	}

	transitions := TransitionList{}
	terr := json.Unmarshal(b, &transitions)
	if terr != nil {
		fmt.Println("Transitions error: ", terr)
		return nil, terr
	}

	return &transitions, nil
}
