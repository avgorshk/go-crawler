package main

import "fmt"
import "net/http"
import "net/url"
//import "os"
import "io/ioutil"
import "strings"
import "strconv"

var search_url string = "https://motolife-nn.ru/"

const (
	New = iota
	Valid = iota
	Invalid = iota
)

type TreeNode struct {
	name string
	status int
	childs [](*TreeNode)
}

func (t *TreeNode) Name() string {
	return t.name
}

func (t *TreeNode) SetName(name string) {
	t.name = name
}

func (t *TreeNode) SetStatus(status int) {
	t.status = status
}

func (t *TreeNode) Insert(path string) {
	levels := strings.Split(path, "/")
	if len(levels) == 0 {
		return
	}

	target := -1
	for i := 0; i < len(t.childs); i++ {
		if t.childs[i].name == levels[0] {
			target = i
		}
	}

	if target == -1 {
		node := new(TreeNode)
		node.name = levels[0]
		node.status = New
		t.childs = append(t.childs, node)

		if len(levels) > 1 {
			path = levels[1]
			for i := 2; i < len(levels); i++ {
				path = path + "/" + levels[i]
			}
			node.Insert(path)
		}
	} else {
		if len(levels) > 1 {
			path = levels[1]
			for i := 2; i < len(levels); i++ {
				path = path + "/" + levels[i]
			}
			t.childs[target].Insert(path)
		}
	}
}

func (t *TreeNode) Print(base_url string, level int) {
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}
	cur_url := strings.Trim(base_url, "/") + "/" + t.name
	fmt.Println(cur_url + " [" + strconv.Itoa(t.status) + "]")
	for i := 0; i < len(t.childs); i++ {
		t.childs[i].Print(cur_url, level + 1)
	}
}

func (t *TreeNode) GetNewList(base_url string) []string {
	var urls []string
	cur_url := strings.Trim(base_url, "/") + "/" + t.name
	if t.status == New {
		urls = append(urls, cur_url)
	}
	for i := 0; i < len(t.childs); i++ {
		child_urls := t.childs[i].GetNewList(cur_url)
		for j := 0; j < len(child_urls); j++ {
			urls = append(urls, child_urls[j])
		}
	}
	return urls
}

func parse(body string, target_host string) []string {
	var urls []string

	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		index := strings.Index(lines[i], "href=")
		if index != -1 {
			href := lines[i][index + len("href="):]
			if href[0] == '"' {
				start := 1
				end := strings.Index(href[start:], "\"")
				if (end > start) {
					link := href[start:end]
					u, err := url.Parse(link)
					if (err == nil && u.Host == target_host) {
						urls = append(urls, link)
					}
				}
			}
		}
	}

	return urls
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
	base_url := strings.Trim(search_url, "/")
	/*u, err := url.Parse(search_url)
	if (err != nil) {
		fmt.Println(err)
		return
	}
	target_host := u.Host*/

	tree := new(TreeNode)
	tree.SetName("")
	tree.SetStatus(New)

	tree.Insert("aaa/bbb")
	tree.Insert("aaa/ccc")

	tree.Print(base_url, 0)

	urls := tree.GetNewList(base_url)
	for i := 0; i < len(urls); i++ {
		fmt.Println(urls[i])
	}

	return

	/*for true {
		nodes := tree.GetNewList()
		if len(nodes) == 0 {
			break
		}

		for i := 0; i < len(nodes); i++ {
			cur_url := base_url + nodes[i].Name()
			body := grab(cur_url)
			if body == "" {
				nodes[i].SetStatus(Invalid)
			} else {
				nodes[i].SetStatus(Valid)
				urls := parse(body, target_host)
				for j := 0; j < len(urls); j++ {
					u, err = url.Parse(urls[j])
					if err == nil {
						tree.Insert(strings.Trim(u.Path, "/"))
					}
				}
			}
		}
	}*/
}