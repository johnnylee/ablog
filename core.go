package ablog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/johnnylee/util"
	"github.com/russross/blackfriday"
	"github.com/termie/go-shutil"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var RootPrefix = "/"

func Main() {
	// Remove output dir.
	_ = os.RemoveAll("output")

	// Load templates.
	tmpl := template.Must(template.ParseGlob("template/*"))

	// Move static content to output dir.
	fmt.Println("Copying static content...")
	if err := shutil.CopyTree("static", "output", nil); err != nil {
		exitErr(err, "Failed to copy static content.")
	}

	dir := NewDir(nil, "content", 0)
	dir.Render(tmpl)
}

// ----------------------------------------------------------------------------
func exitErr(err error, msg string) {
	fmt.Println(msg)
	fmt.Println("Error:", err.Error())
	os.Exit(1)
}

func glob(pattern string) (matches []string) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		exitErr(err, "Failed to glob files: "+pattern)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	return paths
}

// ----------------------------------------------------------------------------
type ADir struct {
	Parent *ADir // Parent directory.
	Level  int   // The directory nesting level. 0 is root.

	outPath string // The output path for the directory.

	RelPath string // The relative path in the output tree, no beginning "/".

	Files []*AFile // Files in the directory.
	Dirs  []*ADir  // Sub-directories.

	FileTags []string // A sorted list of tags from files within the directory.
}

func NewDir(parent *ADir, dirPath string, level int) *ADir {
	fmt.Println("Processing directory:", dirPath)

	dir := ADir{}
	dir.Parent = parent
	dir.Level = level

	dir.RelPath = dirPath[7:]
	for len(dir.RelPath) > 0 && dir.RelPath[0] == '/' {
		dir.RelPath = dir.RelPath[1:]
	}

	fmt.Println("  Relative path:", dir.RelPath)

	dir.outPath = filepath.Join("output", dir.RelPath)
	fmt.Println("  Output path:", dir.outPath)

	// Load pages.
	for _, path := range glob(filepath.Join(dirPath, "*.md")) {
		fmt.Println("  Processing file:", path)
		dir.Files = append(dir.Files, NewAFile(&dir, path))
	}

	// Load and sort tags.
	var tags map[string]struct{}
	for _, file := range dir.Files {
		for _, tag := range file.Tags {
			tags[tag] = struct{}{}
		}
	}

	for k := range tags {
		dir.FileTags = append(dir.FileTags, k)
	}
	sort.Strings(dir.FileTags)

	// Set file prev/next pointers.
	for i, file := range dir.Files {
		if i != 0 {
			file.PrevFile = dir.Files[i-1]
		}
		if i < len(dir.Files)-1 {
			file.NextFile = dir.Files[i+1]
		}
	}

	// Load dirs.
	for _, path := range glob(filepath.Join(dirPath, "*/")) {
		if !util.IsDir(path) {
			continue
		}
		dir.Dirs = append(dir.Dirs, NewDir(&dir, path, level+1))
	}

	return &dir
}

// Get a sub-directory by name.
func (dir *ADir) SubDir(name string) *ADir {
	for _, dir := range dir.Dirs {
		if filepath.Base(dir.RelPath) == name {
			return dir
		}
	}
	return nil
}

// Return all files in the directory having the given tag.
func (dir *ADir) TaggedFiles(tag string) (files []*AFile) {
	for _, file := range dir.Files {
		if file.HasTag(tag) {
			files = append(files, file)
		}
	}
	return
}

// Return all files in the directory and subdirectories.
func (dir *ADir) FilesRecursive() (files []*AFile) {
	files = append(files, dir.Files...)
	for _, subDir := range dir.Dirs {
		files = append(files, subDir.Files...)
	}
	return
}

func (dir *ADir) TaggedFilesRecursive(tag string) (files []*AFile) {
	for _, file := range dir.FilesRecursive() {
		if file.HasTag(tag) {
			files = append(files, file)
		}
	}
	return
}

func (dir *ADir) Render(tmpl *template.Template) {
	fmt.Println("Rendering site...")

	for _, file := range dir.Files {
		file.Render(tmpl)
	}

	for _, subDir := range dir.Dirs {
		subDir.Render(tmpl)
	}
}

// ----------------------------------------------------------------------------
type AFile struct {
	Parent *ADir // The parent directory.
	Level  int   // The parent directory's nesting level.

	mdPath  string // The path to the markdown content file.
	outPath string // The html output path.

	Url        string // The URL of the rendered HTML file.
	RelPath    string // The relative path of HTML file, no leading "/"
	RootPrefix string // The root path prefix.

	// Previous and next files in the directory.
	PrevFile *AFile
	NextFile *AFile

	// The content rendered from the markdownfile.
	Content template.HTML

	// Meta-data from the markdown file below.
	Template string   // The template to use to render this file.
	Tags     []string // Tags for your use.
	Title    string   // The title.
	Author   string   // The author.

	// Timestamps for creation / modification of the content.
	Created  struct{ Year, Month, Day int }
	Modified struct{ Year, Month, Day int }
}

func NewAFile(parent *ADir, mdPath string) *AFile {
	file := AFile{}
	file.Parent = parent
	file.Level = parent.Level

	file.mdPath = mdPath

	// Set the output path.
	file.outPath = filepath.Join(parent.outPath, filepath.Base(mdPath))
	file.outPath = file.outPath[:len(file.outPath)-2] + "html"

	file.Url = filepath.Join(RootPrefix, file.outPath[7:])
	file.RootPrefix = RootPrefix
	file.RelPath = filepath.Join(parent.RelPath, filepath.Base(file.outPath))

	// Load metadata and content from markdown file.
	data, err := ioutil.ReadFile(mdPath)
	if err != nil {
		exitErr(err, "When reading file: "+mdPath)
	}

	meta := bytes.SplitN(data, []byte("----"), 2)[0]
	if err = json.Unmarshal(meta, &file); err != nil {
		exitErr(err, "When reading metadata for file: "+mdPath)
	}

	input := bytes.SplitAfterN(data, []byte("----"), 2)[1]
	file.Content = template.HTML(blackfriday.MarkdownCommon(input))

	return &file
}

func (file *AFile) HasTag(tag string) bool {
	for _, t := range file.Tags {
		if tag == t {
			return true
		}
	}
	return false
}

func (file *AFile) BaseName() string {
	return filepath.Base(file.outPath)
}

func (file *AFile) FirstParagraph() template.HTML {
	return template.HTML(
		bytes.SplitAfterN(
			[]byte(file.Content), []byte("</p>"), 2)[0])
}

func (file *AFile) Render(tmpl *template.Template) {
	fmt.Println("Rendering file:", file.mdPath)
	fmt.Println("  Output path:", file.outPath)
	fmt.Println("  Template:   ", file.Template)
	fmt.Println("  URL:        ", file.Url)
	fmt.Println("  RelPath:    ", file.Url)

	// Make sure the directory exists.
	_ = os.MkdirAll(filepath.Dir(file.outPath), 0777)

	// Open output file for writing.
	f, err := os.Create(file.outPath)
	if err != nil {
		exitErr(err, "Failed to create output file: "+file.outPath)
	}
	defer f.Close()

	if err = tmpl.ExecuteTemplate(f, file.Template, file); err != nil {
		exitErr(err, "Failed to render template: "+file.Template)
	}
}

// Reference time is "Mon Jan 2 15:04:05 MST 2006"
func (file *AFile) FormatCreated(fmt string) string {
	d := time.Date(
		file.Created.Year, time.Month(file.Created.Month), file.Created.Day,
		12, 0, 0, 0, time.UTC)
	return d.Format(fmt)
}

func (file *AFile) FormatModified(fmt string) string {
	d := time.Date(
		file.Modified.Year, time.Month(file.Modified.Month), file.Modified.Day,
		12, 0, 0, 0, time.UTC)
	return d.Format(fmt)
}
