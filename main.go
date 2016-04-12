// j2i parses JIRA rss feed and creates an Invoice using FreshBooks API.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

var (
	jiraSearchID = flag.String("jiraSearchID", "", "JIRA Search Filter ID, for example in ?filter=10101, it's 10101")
	fbProject    = flag.String("fbProject", "", "Fresh Books Project Name")
	fbTask       = flag.String("fbTask", "", "Fresh Books Task")
	trace        = flag.Bool("trace", false, "Trace")
)

type appConfig struct {
	JiraAccountName    string // JIRA account name (i.e. hashjoin - appended to .atlassian.net XML feed for items)
	JiraUname          string // JIRA Username (i.e. admin, not email address)
	JiraPass           string // JIRA password
	FbAccountName      string
	FbAuthToken        string // Token-Based authentication (deprecated)
	FbConsumerKey      string // OAuth authentication
	FbConsumerSecret   string // OAuth authentication
	FbOAuthToken       string // OAuth authentication
	FbOAuthTokenSecret string // OAuth authentication
}

type appContext struct {
	trace bool
	cfg   *appConfig
}

var c appContext

func loadConfig() *appConfig {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	cfgFile := filepath.Join(usr.HomeDir, ".j2i/config.json")
	file, e := ioutil.ReadFile(cfgFile)
	if e != nil {
		fmt.Fprintf(os.Stderr, "Unable to load %s", cfgFile)
		os.Exit(1)
	}
	var config appConfig
	if err := json.Unmarshal(file, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to unmarshal %s", cfgFile)
		os.Exit(1)
	}
	return &config
}

func (c *appContext) printFB(i interface{}, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
		os.Exit(1)
	}
	if c.trace {
		fmt.Printf("%#v\n", i)
	}
}

func (c *appContext) helpFB() {
	fb := NewAPI(c.cfg.FbAccountName, c.cfg.FbAuthToken)
	c.printFB(fb.Clients())
	c.printFB(fb.Projects())
	c.printFB(fb.Tasks())

	fmt.Println("--- Clients ---")
	for _, cl := range fb.clients {
		fmt.Printf("%s\n", cl.Name)
		fb.clientProjects(cl.ClientID)
	}
	fmt.Println("--- Tasks ---")
	for _, tk := range fb.tasks {
		fmt.Printf("\t%s\n", tk.Name)
	}

}

func main() {
	cfg := loadConfig()
	flag.Parse()
	c := appContext{
		trace: *trace,
		cfg:   cfg,
	}

	if *jiraSearchID == "" || *fbProject == "" || *fbTask == "" {
		c.helpFB()
		flag.Usage()
		os.Exit(1)
	}

	var err error
	url := fmt.Sprintf("https://%s.atlassian.net/sr/jira.issueviews:searchrequest-xml/%s/SearchRequest-%s.xml?tempMax=1000&field=key&field=summary&field=timespent&field=due&os_authType=basic", c.cfg.JiraAccountName, *jiraSearchID, *jiraSearchID)
	x, err := c.downloadItems(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
		os.Exit(1)
	}
	allItems := parseXML(x)

	// fmt.Printf("%#v", allItems)
	for i, v := range allItems {
		//                                     Mon, 4 Apr 2016 00:00:00 -0700
		allItems[i].DueDate, err = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", v.Due)
		if err != nil {
			fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
			os.Exit(1)
		}
		// %-67s - pads Summary to 67 chars
		fmt.Printf("%s\t%v\t%s: %-67s%8.2f\n", v.Key.Val, allItems[i].DueDate.Format("2006-JAN-02"), v.Key.Val, v.Summary, float64(v.TimeSpent.Seconds)/60/60)
	}

	fb := NewAPI(c.cfg.FbAccountName, c.cfg.FbAuthToken)
	c.printFB(fb.Clients())
	c.printFB(fb.Projects())
	c.printFB(fb.Tasks())
	c.printFB(fb.Users())
	//fmt.Printf("%#v\n", fb)

	for _, v := range allItems {
		te := &TimeEntry{
			ProjectID: fb.findProject(*fbProject),
			TaskID:    fb.findTask(*fbTask),
			UserID:    1,
			Date:      v.DueDate.Format("2006-01-02"),
			Notes:     fmt.Sprintf("%s: %s", v.Key.Val, v.Summary),
			Hours:     float64(v.TimeSpent.Seconds) / 60 / 60,
		}
		id, err := fb.SaveTimeEntry(te)
		if err != nil {
			fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created Time Entry: ID:%d\n", id)
	}

}
