package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// New URL
	New = iota
	// Valid URL
	Valid = iota
	// Invalid (bad) URL
	Invalid = iota
)

// TreeNode is site URLs tree node
type TreeNode struct {
	name   string
	status int
	childs [](*TreeNode)
}

// Name is a part of path for the URL
func (t *TreeNode) Name() string {
	return t.name
}

// SetName - method to set current part of path for the URL
func (t *TreeNode) SetName(name string) {
	t.name = name
}

// SetStatus - method to set URL status
func (t *TreeNode) SetStatus(status int) {
	t.status = status
}

// Insert - method to insert new URL or change its status
func (t *TreeNode) Insert(path string, status int) {
	path = strings.Trim(path, "/")
	if path == "" {
		if t.status < status {
			t.status = status
		}
		return
	}

	levels := strings.Split(path, "/")
	if len(levels) == 0 {
		return
	}

	target := -1
	for i := 0; i < len(t.childs); i++ {
		if t.childs[i].name == levels[0] {
			target = i
			break
		}
	}

	if target == -1 {
		node := new(TreeNode)
		node.name = levels[0]
		node.status = status
		t.childs = append(t.childs, node)

		if len(levels) > 1 {
			path = levels[1]
			for i := 2; i < len(levels); i++ {
				path = path + "/" + levels[i]
			}
			node.Insert(path, status)
		}
	} else {
		if len(levels) > 1 {
			path = levels[1]
			for i := 2; i < len(levels); i++ {
				path = path + "/" + levels[i]
			}
			t.childs[target].Insert(path, status)
		} else {
			if t.childs[target].status < status {
				t.childs[target].status = status
			}
		}
	}
}

// Print - method to print URL tree
func (t *TreeNode) Print(baseURL string, level int) {
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}
	currentURL := strings.Trim(baseURL, "/") + "/" + t.name
	fmt.Print(currentURL)
	if t.status == New {
		fmt.Println(" [new]")
	}
	if t.status == Valid {
		fmt.Println(" [valid]")
	}
	if t.status == Invalid {
		fmt.Println(" [invalid]")
	}
	for i := 0; i < len(t.childs); i++ {
		t.childs[i].Print(currentURL, level+1)
	}
}

// GetNewList - method to get the list of new URLs in the tree
func (t *TreeNode) GetNewList(baseURL string) []string {
	var urls []string
	currentURL := strings.Trim(baseURL, "/") + "/" + t.name
	if t.status == New {
		urls = append(urls, currentURL)
	}
	for i := 0; i < len(t.childs); i++ {
		childURLs := t.childs[i].GetNewList(currentURL)
		for j := 0; j < len(childURLs); j++ {
			urls = append(urls, childURLs[j])
		}
	}
	return urls
}

func belongToTargetHost(url string, targetHost string) bool {
	return strings.Contains(url, targetHost)
}

func getPath(url string, targetHost string) string {
	pathIndex := strings.Index(url, targetHost)
	if pathIndex == -1 {
		return ""
	}
	return url[pathIndex+len(targetHost):]
}

func parse(body string, targetHost string) []string {
	var urls []string

	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		index := strings.Index(lines[i], "href=")
		if index != -1 {
			href := lines[i][index+len("href="):]
			if href[0] == '"' {
				start := 1
				end := strings.Index(href[start:], "\"")
				if end > start {
					link := href[start : end+1]
					if belongToTargetHost(link, targetHost) {
						urls = append(urls, link)
					}
				}
			}
		}
	}

	return urls
}

func retrieveURLs(url string, targetHost string, urls *([]string)) bool {
	body := grab(url)
	if body == "" {
		return false
	}

	foundURLs := parse(body, targetHost)
	for i := 0; i < len(foundURLs); i++ {
		*urls = append(*urls, foundURLs[i])
	}
	return true
}

func parallelRetrieveURLs(url string, targetHost string, urls *([]string), result chan bool) {
	result <- retrieveURLs(url, targetHost, urls)
}

func grab(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	content := string(bytes)

	return content
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: crawler.exe <URL>")
		return
	}

	baseURL := strings.Trim(os.Args[1], "/")
	targetHost := baseURL

	tree := new(TreeNode)
	tree.SetName("")
	tree.SetStatus(New)

	const threadCount = 8
	const maxURLs = 256

	var start = time.Now()

	step := 0
	for true {
		newURLs := tree.GetNewList(baseURL)
		if len(newURLs) == 0 {
			break
		}

		fmt.Print("Step " + strconv.Itoa(step) + " (" + strconv.Itoa(len(newURLs)) + " new URLs)")

		var threadURLs [threadCount]([]string)
		resChan := make(chan bool)

		i := 0
		for i < len(newURLs) {
			for t := 0; t < threadCount; t++ {
				if i+t < len(newURLs) {
					go parallelRetrieveURLs(newURLs[i+t], targetHost, &(threadURLs[t]), resChan)
				}
			}

			for t := 0; t < threadCount; t++ {
				if i+t < len(newURLs) {
					_ = <-resChan
				}
			}

			for t := 0; t < threadCount; t++ {
				if i+t < len(newURLs) {
					if len(threadURLs[t]) == 0 {
						tree.Insert(getPath(newURLs[i+t], targetHost), Invalid)
					} else {
						tree.Insert(getPath(newURLs[i+t], targetHost), Valid)
						for j := 0; j < len(threadURLs[t]); j++ {
							tree.Insert(getPath(threadURLs[t][j], targetHost), New)
						}
					}
					threadURLs[t] = nil
				}
			}

			fmt.Print(".")
			i = i + threadCount

			if i > maxURLs {
				break
			}
		}

		fmt.Println()
		step = step + 1

		if i > maxURLs {
			break
		}
	}

	duration := time.Since(start)

	fmt.Println("===== Site Structure =====")
	tree.Print(baseURL, 0)
	fmt.Println("==========================")
	fmt.Println(duration)
}
