package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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
	fmt.Println(string(result))
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
