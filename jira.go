package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// ItemKey is part of the Item
type ItemKey struct {
	ID  int64  `xml:"id,attr"`
	Val string `xml:",chardata"`
}

// ItemTimeSpent is part of the Item
type ItemTimeSpent struct {
	Seconds int64  `xml:"seconds,attr"`
	Val     string `xml:",chardata"`
}

// Item is the top level item
type Item struct {
	Key       ItemKey `xml:"key"`
	Summary   string  `xml:"summary"`
	Due       string  `xml:"due"`
	DueDate   time.Time
	TimeSpent ItemTimeSpent `xml:"timespent"`
}

// Items are collection of Item
type Items []Item

func (c *appContext) downloadItems(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.cfg.JiraUname, c.cfg.JiraPass)

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
	if c.trace {
		fmt.Println(string(result))
	}
	return result, nil

}

func parseXML(x []byte) Items {
	var allItems Items
	//dec := xml.NewDecoder(os.Stdin)
	dec := xml.NewDecoder(bytes.NewReader(x))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
			os.Exit(1)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "item" {
				// "this" has to be inside since timespent can be missing
				// and when it is it gets it's value from last iteration
				var this Item
				dec.DecodeElement(&this, &tok)
				allItems = append(allItems, this)
			}
		}
	}
	return allItems
}

func (c *appContext) updateTrans(v Item, j *Jira) {
	r, err := j.IssuesService.Transition(v.Key.Val, c.cfg.JiraInvoicedTransID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: Resp: %v\n", string(r))
		fmt.Fprintf(os.Stderr, "j2i: Error: %v\n", err)
		os.Exit(1)
	}
	if c.trace {
		fmt.Printf("%v\n", string(r))
	}
	fmt.Printf("\tTransitioned ISSUE:%s to ID:%s\n", v.Key.Val, c.cfg.JiraInvoicedTransID)
}

func (c *appContext) updateLabel(v Item, j *Jira, invoice string) {
	r, err := j.IssuesService.Label(v.Key.Val, c.cfg.JiraInvoicedPrefix+invoice)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: Resp: %v\n", string(r))
		fmt.Fprintf(os.Stderr, "j2i: Error: %v\n", err)
		os.Exit(1)
	}
	if c.trace {
		fmt.Printf("%v\n", string(r))
	}
	fmt.Printf("\tLabeled ISSUE:%s as %s\n", v.Key.Val, c.cfg.JiraInvoicedPrefix+invoice)

}

func (c *appContext) updateItems(allItems Items, a *API) {
	url := fmt.Sprintf("https://%s.atlassian.net", c.cfg.JiraAccountName)
	j := NewJiraClient(url, c.cfg.JiraUname, c.cfg.JiraPass, 1500)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n\nGo to FreshBooks and create invoice, then come back here and enter Invoice#: ")
	invoice, _ := reader.ReadString('\n')
	fmt.Printf("Setting Invoice to: %s", invoice)

	// need to trim \n! - it gets translated to &#xA; in XML call to FB!
	invoice = strings.TrimSpace(invoice)

	usr, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get current user %s", err)
		os.Exit(1)
	}

	a.invoicePDF(invoice, filepath.Join(usr.HomeDir, "Desktop", "Invoice_"+invoice+".pdf"))

	for _, v := range allItems {
		c.updateTrans(v, j)
		c.updateLabel(v, j, invoice)
	}
}
