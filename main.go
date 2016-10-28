// j2i parses JIRA rss feed and creates an Invoice using FreshBooks API.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

var (
	client    = flag.String("client", "", "Client CODE from ~/.j2i/config.json - maps to JIRA Search Filter ID")
	fbProject = flag.String("fbProject", "", "Fresh Books Project Name")
	fbTask    = flag.String("fbTask", "", "Fresh Books Task")
	doFB      = flag.Bool("doFB", true, "Do a push to FreshBooks")
	doJIRA    = flag.Bool("doJIRA", true, "Do an update back to JIRA")
	trace     = flag.Bool("trace", false, "Trace flag")
)

type appConfig struct {
	JiraAccountName     string            // Account name (i.e. hashjoin - appended to .atlassian.net XML feed for items)
	JiraUname           string            // Username (i.e. admin, not email address)
	JiraPass            string            // Password
	JiraInvoicedTransID string            // Transition ID set on invoiced issues (for example Done=11 on our JIRA Cloud Instance)
	JiraInvoicedPrefix  string            // Invoiced issues are labled with JiraInvoicedPrefix+FB-Invoice#
	ClientSearchIDs     map[string]string // Client Code to JIRA Search Filter ID mapping
	FbAccountName       string
	FbAuthToken         string // Token-Based authentication (deprecated)
	FbConsumerKey       string // OAuth authentication
	FbConsumerSecret    string // OAuth authentication
	FbOAuthToken        string // OAuth authentication
	FbOAuthTokenSecret  string // OAuth authentication
}

type appContext struct {
	client     string
	trace      bool
	doFB       bool
	doJIRA     bool
	reportOnly bool
	cfg        *appConfig
}

var c *appContext

func loadConfig() *appConfig {
	usr, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
		os.Exit(1)
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

	fmt.Printf("\n--- Clients ---\n")
	for _, cl := range fb.clients {
		fmt.Printf("%s\n", cl.Name)
		fb.clientProjects(cl.ClientID)
	}
	fmt.Printf("\n--- Tasks ---\n")
	for _, tk := range fb.tasks {
		fmt.Printf("%s\n", tk.Name)
	}
	fmt.Printf("\n")

}

func main() {
	cfg := loadConfig()
	flag.Parse()
	c = &appContext{
		client: *client,
		trace:  *trace,
		doFB:   *doFB,
		doJIRA: *doJIRA,
		cfg:    cfg,
	}

	if *fbProject == "" || *fbTask == "" {
		c.reportOnly = true
	}

	if *client == "" {
		c.helpFB()
		fmt.Printf("If you only want to see JIRA report - omit fbProject or fbTask or both\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var err error
	url := fmt.Sprintf("https://%s.atlassian.net/sr/jira.issueviews:searchrequest-xml/%s/SearchRequest-%s.xml?tempMax=1000&field=key&field=summary&field=timespent&field=due&os_authType=basic", c.cfg.JiraAccountName, c.cfg.ClientSearchIDs[*client], c.cfg.ClientSearchIDs[*client])
	x, err := c.downloadItems(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
		os.Exit(1)
	}
	allItems := parseXML(x)

	// fmt.Printf("%#v", allItems)
	var totTime float64
	for i, v := range allItems {
		//                                     Mon, 4 Apr 2016 00:00:00 -0700
		allItems[i].DueDate, err = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", v.Due)

		if c.trace {
			fmt.Printf("%#v\n", v.Due)
			fmt.Printf("%#v\n", allItems[i].DueDate)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
			os.Exit(1)
		}
		// %-70s - pads Summary to 70 chars
		fmt.Printf("%v   %s: %-70s%.2f\n", allItems[i].DueDate.Format("2006-Jan-02"), v.Key.Val, v.Summary, float64(v.TimeSpent.Seconds)/60/60)
		totTime += float64(v.TimeSpent.Seconds) / 60 / 60
	}
	fmt.Printf("%96s\n", "-----")
	fmt.Printf("%90s: %.2f\n", "Total Hours", totTime)

	if c.reportOnly {
		os.Exit(0)
	}

	var fb *API
	fb = NewAPI(c.cfg.FbAccountName, c.cfg.FbAuthToken)

	if c.doFB {
		c.printFB(fb.Clients())
		c.printFB(fb.Projects())
		c.printFB(fb.Tasks())
		c.printFB(fb.Users())

		fmt.Printf("\n%87s: %.2f\n", "Task Total", totTime*fb.findTaskRate(*fbTask))

		fmt.Printf("---> FreshBooks.Start\n")
		fb.pushFB(allItems, *fbProject, *fbTask)
		fmt.Printf("<--- FreshBooks.End\n")
	}

	if c.doJIRA {
		c.updateItems(allItems, fb)
	}

}
