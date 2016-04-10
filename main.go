// j2j parses JIRA rss feed and creates an Invoice using FreshBooks API.
package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"time"
)

// ItemKey is part of item
type ItemKey struct {
	ID  int64  `xml:"id,attr"`
	Val string `xml:",chardata"`
}

// ItemTimeSpent is part of item
type ItemTimeSpent struct {
	Seconds int64  `xml:"seconds,attr"`
	Val     string `xml:",chardata"`
}

// Item is the top level item
type Item struct {
	Key       ItemKey       `xml:"key"`
	Summary   string        `xml:"summary"`
	Due       string        `xml:"due"`
	TimeSpent ItemTimeSpent `xml:"timespent"`
}

// Items are collection of Item
type Items []Item

func main() {
	var allItems Items
	dec := xml.NewDecoder(os.Stdin)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "j2j: %v\n", err)
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
	// fmt.Printf("%#v", allItems)
	for _, v := range allItems {
		//               Mon, 4 Apr 2016 00:00:00 -0700
		d, _ := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", v.Due)
		// %-67s - pads Summary to 67 chars
		fmt.Printf("%s\t%v\t%-67s%8.2f\n", v.Key.Val, d.Format("2006-JAN-02"), v.Summary, float64(v.TimeSpent.Seconds)/60/60)
	}

}
