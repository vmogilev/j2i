// FreshBooks API
// based on https://github.com/toggl/go-freshbooks
package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/tambet/oauthplain"
)

type (
	// API - Top level
	API struct {
		apiURL     string
		apiToken   string
		oAuthToken *oauthplain.Token
		perPage    int
		users      []User
		tasks      []Task
		clients    []Client
		projects   []Project
	}
	// Request - controls API request vars
	Request struct {
		XMLName xml.Name `xml:"request"`
		Method  string   `xml:"method,attr"`
		PerPage int      `xml:"per_page"`
		Page    int      `xml:"page"`
	}
	// TimeEntryRequest - time entry specific
	TimeEntryRequest struct {
		XMLName   xml.Name  `xml:"request"`
		Method    string    `xml:"method,attr"`
		TimeEntry TimeEntry `xml:"time_entry"`
	}
	// Response - controls API response vars
	Response struct {
		Error    string      `xml:"error"`
		Clients  ClientList  `xml:"clients"`
		Projects ProjectList `xml:"projects"`
		Tasks    TaskList    `xml:"tasks"`
		Users    UserList    `xml:"staff_members"`
		Invoices InvoiceList `xml:"invoices"`
	}
	// TimeEntryResponse - time entry specific
	TimeEntryResponse struct {
		Status      string `xml:"status,attr"`
		Error       string `xml:"error"`
		Code        string `xml:"code"`
		Field       string `xml:"field"`
		TimeEntryID int    `xml:"time_entry_id"`
	}
	// Pagination - pagination controls
	Pagination struct {
		Page    int `xml:"page,attr"`
		Total   int `xml:"total,attr"`
		PerPage int `xml:"per_page,attr"`
	}
	// ClientList - clients
	ClientList struct {
		Pagination
		Clients []Client `xml:"client"`
	}
	// ProjectList - projects
	ProjectList struct {
		Pagination
		Projects []Project `xml:"project"`
	}
	// TaskList - tasks
	TaskList struct {
		Pagination
		Tasks []Task `xml:"task"`
	}
	// UserList - users
	UserList struct {
		Pagination
		Users []User `xml:"member"`
	}
	// InvoiceList - invoices
	InvoiceList struct {
		Pagination
		Invoices []Invoice `xml:"invoice"`
	}
	// Client - specific client
	Client struct {
		ClientID int    `xml:"client_id"`
		Name     string `xml:"organization"`
	}
	// Project - specific project
	Project struct {
		ProjectID int    `xml:"project_id"`
		ClientID  int    `xml:"client_id"`
		Name      string `xml:"name"`
		TaskIDs   []int  `xml:"tasks>task>task_id"`
		UserIDs   []int  `xml:"staff>staff>staff_id"`
	}
	// Task - specific task
	Task struct {
		TaskID int     `xml:"task_id"`
		Name   string  `xml:"name"`
		Rate   float64 `xml:"rate"`
	}
	// User - specific user
	User struct {
		UserID    int    `xml:"staff_id"`
		Email     string `xml:"email"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
	}
	// TimeEntry - single time entry
	TimeEntry struct {
		TimeEntryID int     `xml:"time_entry_id"`
		ProjectID   int     `xml:"project_id"` // Required
		TaskID      int     `xml:"task_id"`    // Required
		UserID      int     `xml:"staff_id"`   // Required
		Date        string  `xml:"date"`       // Required
		Notes       string  `xml:"notes"`
		Hours       float64 `xml:"hours"`
	}
	// Invoice - specific Invoice
	Invoice struct {
		InvoiceID int     `xml:"invoice_id"`
		Number    string  `xml:"number"`
		Date      string  `xml:"date"`
		PONumber  string  `xml:"po_number"`
		Amount    float64 `xml:"amount"`
	}
)

// NewAPI - sets up new API params
func NewAPI(account string, token interface{}) *API {
	url := fmt.Sprintf("https://%s.freshbooks.com/api/2.1/xml-in", account)
	fb := API{apiURL: url, perPage: 25}
	fb.users = make([]User, 0)
	fb.tasks = make([]Task, 0)
	fb.clients = make([]Client, 0)
	fb.projects = make([]Project, 0)

	switch token.(type) {
	case string:
		fb.apiToken = token.(string)
	case *oauthplain.Token:
		fb.oAuthToken = token.(*oauthplain.Token)
	}
	return &fb
}

// Clients - calls fetchClients
func (a *API) Clients() ([]Client, error) {
	err := a.fetchClients(1)
	return a.clients, err
}

// Projects - calls fetchProjects
func (a *API) Projects() ([]Project, error) {
	err := a.fetchProjects(1)
	return a.projects, err
}

// Tasks - calls fetchTasks
func (a *API) Tasks() ([]Task, error) {
	err := a.fetchTasks(1)
	return a.tasks, err
}

// Users - calls fetchUsers
func (a *API) Users() ([]User, error) {
	err := a.fetchUsers(1)
	return a.users, err
}

func (a *API) fetchClients(page int) error {
	request := &Request{Method: "client.list", Page: page, PerPage: a.perPage}
	result, err := a.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return err
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	a.clients = append(a.clients, parsedInto.Clients.Clients...)
	if parsedInto.Clients.Total > parsedInto.Clients.PerPage*page {
		return a.fetchClients(page + 1)
	}
	return nil
}

func (a *API) fetchProjects(page int) error {
	request := &Request{Method: "project.list", Page: page, PerPage: a.perPage}
	result, err := a.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return (err)
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	a.projects = append(a.projects, parsedInto.Projects.Projects...)
	if parsedInto.Projects.Total > parsedInto.Projects.PerPage*page {
		return a.fetchProjects(page + 1)
	}
	return nil
}

func (a *API) fetchTasks(page int) error {
	request := &Request{Method: "task.list", Page: page, PerPage: a.perPage}
	result, err := a.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return err
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	a.tasks = append(a.tasks, parsedInto.Tasks.Tasks...)
	if parsedInto.Tasks.Total > parsedInto.Tasks.PerPage*page {
		return a.fetchTasks(page + 1)
	}
	return nil
}

func (a *API) fetchUsers(page int) error {
	request := &Request{Method: "staff.list", Page: page, PerPage: a.perPage}
	result, err := a.makeRequest(request)
	if err != nil {
		return err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return err
	}
	if len(parsedInto.Error) > 0 {
		return errors.New(parsedInto.Error)
	}
	a.users = append(a.users, parsedInto.Users.Users...)
	if parsedInto.Users.Total > parsedInto.Users.PerPage*page {
		return a.fetchUsers(page + 1)
	}
	return nil
}

// SaveTimeEntry - updates (if TimeEntryID != 0) of creates time entry
func (a *API) SaveTimeEntry(timeEntry *TimeEntry) (int, error) {
	var method string
	if timeEntry.TimeEntryID != 0 {
		method = "time_entry.update"
	} else {
		method = "time_entry.create"
	}
	request := &TimeEntryRequest{Method: method, TimeEntry: *timeEntry}
	result, err := a.makeRequest(request)
	if err != nil {
		return 0, err
	}
	parsedInto := TimeEntryResponse{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return 0, err
	}
	if parsedInto.Status == "ok" {
		return parsedInto.TimeEntryID, nil
	}
	return 0, errors.New(parsedInto.Error)
}

func (a *API) makeRequest(request interface{}) (*[]byte, error) {
	xmlRequest, err := xml.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, err
	}

	if c.trace {
		fmt.Printf("makeRequest: %v\n", string(xmlRequest))
	}

	req, err := http.NewRequest("POST", a.apiURL, bytes.NewBuffer(xmlRequest))
	if err != nil {
		return nil, err
	}

	if a.apiToken != "" {
		req.SetBasicAuth(a.apiToken, "X")
	} else if a.oAuthToken != nil {
		header := a.oAuthToken.AuthHeader()
		req.Header.Set("Authorization", header)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
