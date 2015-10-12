package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileTree struct {
	ModifiedTime time.Time
	IsLink       bool
	IsDir        bool
	LinksTo      string
	Size         int64
	Name         string
	path         string
	Children     []*FileTree
}

func GetTree(filePath string) *FileTree {
	info, err := os.Lstat(filePath)
	if err != nil {
		fmt.Println("Error getting stats on file:", filePath, "\nError:", err)
		os.Exit(2)
	}

	f := &FileTree{
		path:         filePath,
		ModifiedTime: info.ModTime(),
		Size:         info.Size(),
		IsDir:        info.IsDir(),
		Name:         info.Name(),
		IsLink:       (info.Mode()&os.ModeSymlink != 0),
	}

	f.path = filePath
	f.ModifiedTime = info.ModTime()
	f.Size = info.Size()
	f.IsDir = info.IsDir()
	f.Name = info.Name()
	f.IsLink = (info.Mode()&os.ModeSymlink != 0)
	if f.IsLink {
		link, err := os.Readlink(filePath)
		if err != nil {
			fmt.Println("Error reading link for file:", f.Name)
			os.Exit(2)
		}
		linkPath, err := filepath.Abs(filepath.Join(filepath.Dir(f.path), link))
		if err != nil {
			fmt.Println("Error getting link path:\n", err)
		}
		f.LinksTo = linkPath
	}
	return f
}

func (f *FileTree) GetChildren(recursive bool) {
	if f.IsDir && !f.IsLink {
		children, err := ioutil.ReadDir(f.path)
		if err != nil {
			fmt.Println("Error reading directory:", f.path)
			os.Exit(2)
		}
		var childPath string
		for _, child := range children {
			childPath = filepath.Join(f.path, child.Name())
			f.Children = append(f.Children, GetTree(childPath))
		}
		if recursive {
			for _, childInfo := range f.Children {
				childInfo.GetChildren(recursive)
			}
		}
	}
}

func (f *FileTree) ToText() string {
	var buf bytes.Buffer
	f.writeText(0, &buf)
	return buf.String()
}

func (f *FileTree) writeText(depth int, buf *bytes.Buffer) {
	prefix := strings.Repeat("  ", depth)
	var suffix string
	if f.IsLink {
		suffix = "* (" + f.LinksTo + ")"
	} else if f.IsDir {
		suffix = "/"
	}
	name := f.Name
	if depth == 0 {
		name = f.path
	}
	buf.WriteString(prefix + name + suffix + "\n")
	for _, child := range f.Children {
		child.writeText(depth+1, buf)
	}
}

func (f *FileTree) ToJson() string {
	b, err := json.MarshalIndent(f.Children, "", "    ")
	if err != nil {
		fmt.Println("Problem serializing to JSON:\n", err)
	}
	return string(b)
}

func (f *FileTree) ToYaml() string {
	b, err := yaml.Marshal(f.Children)
	if err != nil {
		fmt.Println("Error converting to yaml:\n", err)
	}
	return string(b)
}

func main() {

	// Specify/Parse/Validate CLI args
	pth := flag.String("path", "", "path to folder")
	recursive := flag.Bool("recursive", false, "list files recursively")
	output := flag.String("output", "text", "<json|yaml|text>")
	flag.Parse()

	if *pth == "" {
		fmt.Println("Must specify path.")
		os.Exit(1)
	}

	if *output != "yaml" && *output != "json" && *output != "text" {
		fmt.Println("Invalid output format: ", *output)
		os.Exit(2)
	}

	// Build file tree structure
	tree := GetTree(*pth)
	tree.GetChildren(*recursive)

	// Display Result
	var toPrint string
	switch *output {
	case "json":
		toPrint = tree.ToJson()
	case "text":
		toPrint = tree.ToText()
	case "yaml":
		toPrint = tree.ToYaml()
	default:
		// Never reached because of validation above
	}
	fmt.Println(toPrint)
}
