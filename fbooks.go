package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
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

func (a *API) findTaskRate(name string) float64 {
	for _, v := range a.tasks {
		if v.Name == name {
			return v.Rate
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

func (a *API) invoiceByNum(invNumber string) (Invoice, error) {

	req := struct {
		XMLName xml.Name `xml:"request"`
		Method  string   `xml:"method,attr"`
		PerPage int      `xml:"per_page"`
		Page    int      `xml:"page"`
		Number  string   `xml:"number"`
	}{
		Method:  "invoice.list",
		Page:    1,
		PerPage: a.perPage,
		Number:  invNumber,
	}

	result, err := a.makeRequest(&req)
	if err != nil {
		return Invoice{}, err
	}
	parsedInto := Response{}
	if err := xml.Unmarshal(*result, &parsedInto); err != nil {
		return Invoice{}, (err)
	}
	if len(parsedInto.Error) > 0 {
		return Invoice{}, errors.New(parsedInto.Error)
	}

	if c.trace {
		fmt.Printf("makeRequest: %#v\n", parsedInto)
	}

	if len(parsedInto.Invoices.Invoices) > 0 {
		return parsedInto.Invoices.Invoices[0], nil
	}

	return Invoice{}, errors.New("Invoice Number: " + invNumber + " can't be located")

}

func (a *API) invoicePDF(invNumber string, saveTo string) error {
	var err error
	inv, err := a.invoiceByNum(invNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\t%-15s: %d\n", "ID", inv.InvoiceID)
	fmt.Printf("\t%-15s: %s\n", "Number", inv.Number)
	fmt.Printf("\t%-15s: %s\n", "Date", inv.Date)
	fmt.Printf("\t%-15s: %s\n", "PO", inv.PONumber)
	fmt.Printf("\t%-15s: %.2f\n", "Amount", inv.Amount)

	fmt.Printf("\tDownloading Invoice PDF to: %s\n", saveTo)
	req := struct {
		XMLName   xml.Name `xml:"request"`
		Method    string   `xml:"method,attr"`
		InvoiceID int      `xml:"invoice_id"`
	}{
		Method:    "invoice.getPDF",
		InvoiceID: inv.InvoiceID,
	}

	result, err := a.makeRequest(&req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: %v\n", err)
		os.Exit(1)
	}

	dst, err := os.OpenFile(saveTo, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "j2i: can't save invoice! %v\n", err)
		os.Exit(1)
	}
	defer dst.Close()

	_, err = io.Copy(dst, bytes.NewReader(*result))
	if err != nil {
		// to close the dst via defer ...
		return err
	}
	return nil
}
