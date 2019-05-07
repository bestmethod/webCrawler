package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)
import "./crawler"

func TestCallback(t *testing.T) {
	cb := new(Callback)
	cb.indent = false
	cb.errStderr = false
	u := crawler.FoundUrls{CrawlUrl:"testA",FoundUrls:[]string{"testB"},Err:errors.New("testC")}
	stdout := os.Stdout
	stderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.FailNow()
	}
	os.Stdout = wOut
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.FailNow()
	}
	os.Stderr = wErr
	cb.callback(&u)
	os.Stdout = stdout
	os.Stderr = stderr
	_ = wOut.Close()
	_ = wErr.Close()
	var bufOut bytes.Buffer
	_, err = io.Copy(&bufOut, rOut)
	if err != nil {
		t.FailNow()
	}
	var bufErr bytes.Buffer
	_, err = io.Copy(&bufErr, rErr)
	if err != nil {
		t.FailNow()
	}
	_ = rOut.Close()
	_ = rErr.Close()
	out := bufOut.String()
	a := `{"CrawledUrl":"testA","FoundUrls":["testB"],"Depth":0,"Error":"testC"},`
	a = fmt.Sprintf("%s\n", a)
	if out != a {
		t.Errorf("stdout: %s", out)
		t.FailNow()
	}
	erra := bufErr.String()
	if erra != "" {
		t.Errorf("stderr: %s", erra)
		t.FailNow()
	}
}

func TestCallbackIndent(t *testing.T) {
	cb := new(Callback)
	cb.indent = true
	cb.errStderr = false
	u := crawler.FoundUrls{CrawlUrl:"testA",FoundUrls:[]string{"testB"},Err:errors.New("testC")}
	stdout := os.Stdout
	stderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.FailNow()
	}
	os.Stdout = wOut
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.FailNow()
	}
	os.Stderr = wErr
	cb.callback(&u)
	os.Stdout = stdout
	os.Stderr = stderr
	_ = wOut.Close()
	_ = wErr.Close()
	var bufOut bytes.Buffer
	_, err = io.Copy(&bufOut, rOut)
	if err != nil {
		t.FailNow()
	}
	var bufErr bytes.Buffer
	_, err = io.Copy(&bufErr, rErr)
	if err != nil {
		t.FailNow()
	}
	_ = rOut.Close()
	_ = rErr.Close()
	out := bufOut.String()
	a := `{
	"CrawledUrl": "testA",
	"FoundUrls": [
		"testB"
	],
	"Depth": 0,
	"Error": "testC"
},
`
	if out != a {
		t.Errorf("stdout: %s", out)
		t.FailNow()
	}
	erra := bufErr.String()
	if erra != "" {
		t.Errorf("stderr: %s", erra)
		t.FailNow()
	}
}

func TestCallbackIndentStderr(t *testing.T) {
	cb := new(Callback)
	cb.indent = true
	cb.errStderr = true
	u := crawler.FoundUrls{CrawlUrl:"testA",FoundUrls:[]string{"testB"},Err:errors.New("testC")}
	stdout := os.Stdout
	stderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.FailNow()
	}
	os.Stdout = wOut
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.FailNow()
	}
	os.Stderr = wErr
	cb.callback(&u)
	os.Stdout = stdout
	os.Stderr = stderr
	_ = wOut.Close()
	_ = wErr.Close()
	var bufOut bytes.Buffer
	_, err = io.Copy(&bufOut, rOut)
	if err != nil {
		t.FailNow()
	}
	var bufErr bytes.Buffer
	_, err = io.Copy(&bufErr, rErr)
	if err != nil {
		t.FailNow()
	}
	_ = rOut.Close()
	_ = rErr.Close()
	out := bufOut.String()
	a := `{
	"CrawledUrl": "testA",
	"FoundUrls": [
		"testB"
	],
	"Depth": 0,
	"Error": "testC"
},
`
	if out != a {
		t.Errorf("stdout: %s", out)
		t.FailNow()
	}
	erra := bufErr.String()
	if erra != "ERROR in `testA`: testC\n" {
		t.Errorf("stderr: %s", erra)
		t.FailNow()
	}
}
