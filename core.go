package ablog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/johnnylee/util"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The default root prefix is "/".
var RootPrefix = "/"

func Main() {
	// Remove output dir.
	_ = os.RemoveAll("output")

	// Walk content directory, copying directories and static content.
	fmt.Println("Copying static content...")
	walk := func(path string, info os.FileInfo, err error) error {
		outPath := filepath.Join("output", path[7:])

		if info.IsDir() {
			fmt.Println("  Creating directory:", outPath)
			if err := os.MkdirAll(outPath, 0777); err != nil {
				exitErr(err, "Failed to create directory: "+outPath)
			}
			return nil
		}

		if path[len(path)-3:] != ".md" {
			fmt.Println("  Linking file:", outPath)
			if err := os.Link(path, outPath); err != nil {
				exitErr(err, "Failed to link file: "+path)
			}
		}

		return nil
	}

	if err := filepath.Walk("content", walk); err != nil {
		exitErr(err, "Failed to walk content directory.")
	}

	// Load templates.
	tmpl := template.Must(template.ParseGlob("template/*"))

	dir := NewDir(nil, "content", 0)
	dir.render(tmpl)
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
	contentPath string // The path to directory under "content".
	outPath     string // The output path for the directory.

	RelPath string // The relative path in the output tree, beginning "/".

	Parent *ADir // Parent directory.
	Level  int   // The directory nesting level. 0 is root.

	Files []*AFile // Files in the directory.
	Dirs  []*ADir  // Sub-directories.

	// Previous and next directories.
	PrevDir *ADir
	NextDir *ADir

	// Sorted list of file tags in current directory, and recursively.
	FileTags          []string
	FileTagsRecursive []string
}

func NewDir(parent *ADir, dirPath string, level int) *ADir {
	fmt.Println("Processing directory:", dirPath)

	dir := ADir{}
	dir.contentPath = dirPath
	dir.outPath = filepath.Join("output", dirPath[7:])
	fmt.Println("  Output path:", dir.outPath)

	dir.Parent = parent
	dir.Level = level

	// Loading.
	dir.loadFiles()
	dir.loadDirs()
	dir.loadTags()

	return &dir
}

func (dir *ADir) loadFiles() {
	for _, path := range glob(filepath.Join(dir.contentPath, "*.md")) {
		fmt.Println("  Processing file:", path)
		dir.Files = append(dir.Files, NewAFile(dir, path))
	}

	// Set file prev/next pointers.
	for i, file := range dir.Files {
		if i != 0 {
			file.PrevFile = dir.Files[i-1]
		}
		if i < len(dir.Files)-1 {
			file.NextFile = dir.Files[i+1]
		}
	}
}

func (dir *ADir) loadDirs() {
	for _, path := range glob(filepath.Join(dir.contentPath, "*/")) {
		if !util.IsDir(path) {
			continue
		}
		dir.Dirs = append(dir.Dirs, NewDir(dir, path, dir.Level+1))
	}

	// Set dir prev/next pointers.
	for i, d := range dir.Dirs {
		if i != 0 {
			d.PrevDir = dir.Dirs[i-1]
		}
		if i < len(dir.Dirs)-1 {
			d.NextDir = dir.Dirs[i+1]
		}
	}
}

func (dir *ADir) loadTags() {
	fmt.Println("  Loading tags...")

	distinct := func(tagsList ...[]string) (out []string) {
		var m map[string]struct{}

		for _, tags := range tagsList {
			for _, tag := range tags {
				m[tag] = struct{}{}
			}
		}

		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return
	}

	var tagsList [][]string
	for _, file := range dir.Files {
		tagsList = append(tagsList, file.Tags)
	}

	dir.FileTags = distinct(tagsList...)

	for _, subDir := range dir.Dirs {
		tagsList = append(tagsList, subDir.FileTagsRecursive)
	}

	dir.FileTagsRecursive = distinct(tagsList...)
}

// SubDir: Get a sub-directory by name.
func (dir *ADir) SubDir(name string) *ADir {
	for _, dir := range dir.Dirs {
		if filepath.Base(dir.outPath) == name {
			return dir
		}
	}
	return nil
}

// FilesRecursive: Return files in this and any sub directory.
func (dir *ADir) FilesRecursive() (files []*AFile) {
	files = append(files, dir.Files...)
	for _, subDir := range dir.Dirs {
		files = append(files, subDir.Files...)
	}
	return
}

// TaggedFilesAll: Return files in directory having all the given tags.
func (dir *ADir) TaggedFilesAll(tags ...string) (files []*AFile) {
	for _, file := range dir.Files {
		if file.HasTagsAll(tags...) {
			files = append(files, file)
		}
	}
	return
}

// TaggedFilesAllRecursive: Recursive version of TaggedFilesAll.
func (dir *ADir) TaggedFilesAllRecursive(tags ...string) (files []*AFile) {
	for _, file := range dir.FilesRecursive() {
		if file.HasTagsAll(tags...) {
			files = append(files, file)
		}
	}
	return
}

// TaggedFilesAny: Return files in directory having any the given tags.
func (dir *ADir) TaggedFilesAny(tags ...string) (files []*AFile) {
	for _, file := range dir.Files {
		if file.HasTagsAny(tags...) {
			files = append(files, file)
		}
	}
	return
}

// TaggedFilesAnyRecursive: Recursive version of TaggedFilesAny.
func (dir *ADir) TaggedFilesAnyRecursive(tags ...string) (files []*AFile) {
	for _, file := range dir.FilesRecursive() {
		if file.HasTagsAny(tags...) {
			files = append(files, file)
		}
	}
	return
}

func (dir *ADir) render(tmpl *template.Template) {
	fmt.Println("Rendering directory: " + dir.outPath)

	for _, file := range dir.Files {
		file.render(tmpl)
	}

	for _, subDir := range dir.Dirs {
		subDir.render(tmpl)
	}
}

// ----------------------------------------------------------------------------
type AFile struct {
	Parent *ADir // The parent directory.
	Level  int   // The parent directory's nesting level.

	mdPath  string // The path to the markdown content file.
	outPath string // The html output path.

	Url          string // The URL of the rendered HTML file.
	RootPrefix   string // The root path prefix.
	RootRelative string // Relative path to root dir.

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
	file.RootRelative = strings.Repeat("../", file.Level)

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
	file.Content = template.HTML(markdown(input))

	return &file
}

func (file *AFile) UrlRelative(baseFile *AFile) string {
	path, err := filepath.Rel(baseFile.Parent.outPath, file.outPath)
	if err != nil {
		exitErr(err, "When computing relative path")
	}
	return path
}

// HasTag: Return true if the file has the given tag.
func (file *AFile) HasTag(tag string) bool {
	for _, t := range file.Tags {
		if tag == t {
			return true
		}
	}
	return false
}

// HasTagsAll: Return true if the file has all the given tags.
func (file *AFile) HasTagsAll(tags ...string) bool {
	for _, t := range tags {
		if !file.HasTag(t) {
			return false
		}
	}
	return true
}

// HasTagsAny: Return true if the file has any of the given tags.
func (file *AFile) HasTagsAny(tags ...string) bool {
	for _, t := range tags {
		if file.HasTag(t) {
			return true
		}
	}
	return false
}

// BaseName: Return the bare html filename.
func (file *AFile) BaseName() string {
	return filepath.Base(file.outPath)
}

// FirstParagraph: Return the first paragraph of the file. The returned HTML
// will contain the opening and closing <p> tags.
func (file *AFile) FirstParagraph() template.HTML {
	return template.HTML(
		bytes.SplitAfterN(
			[]byte(file.Content), []byte("</p>"), 2)[0])
}

// FormatCreated: Format the creation date using Go's date formatting function.
// The reference time is "Mon Jan 2 15:04:05 MST 2006"
func (file *AFile) FormatCreated(fmt string) string {
	d := time.Date(
		file.Created.Year, time.Month(file.Created.Month), file.Created.Day,
		12, 0, 0, 0, time.UTC)
	return d.Format(fmt)
}

// FormatModified: The same as FormatCreated, but for the modification date.
func (file *AFile) FormatModified(fmt string) string {
	d := time.Date(
		file.Modified.Year, time.Month(file.Modified.Month), file.Modified.Day,
		12, 0, 0, 0, time.UTC)
	return d.Format(fmt)
}

func (file *AFile) render(tmpl *template.Template) {
	fmt.Println("Rendering file:", file.mdPath)
	fmt.Println("  Output path:", file.outPath)
	fmt.Println("  Template:   ", file.Template)
	fmt.Println("  URL:        ", file.Url)

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
