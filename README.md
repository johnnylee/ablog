# ablog

This is a simple static site generator I wrote for my wife's website, but then modified a bit to add features I'd like like pygments support. 

## Overview

A site consists of two directories. `template` contains html templates for Go's `html/template` package. `content` is the site's root directory. It should contain all the static content, as well as markdown files that will be rendered into the templates. 

## Example

A simple example site might have the following structure: 

```
template/
    index.html
    blog-post.html
    blog-index.html
    
content/
    index.md
    blog/
        2015-06-10-first-post.md
        2015-07-20-last-post.md
```

## Markdown Metadata

Each markdown file under the `content` directory should begin with a header of the following form: 

```js
{
  "Template": "tmpl.html",
  "Tags": ["MyTag"],
  "Title": "My Page Title",
  "Author": "Some Guy",
  "Created": { "Year": 2015, "Month":6, "Day":12 },
  "Modified": { "Year": 2015, "Month":6, "Day":14 }
}
----
```

The data in the header will be available in your templates. 

## Template Context

The context passed to each template when rendering a page is an `AFile` object: 

```go
type AFile struct {
	Parent *ADir // The parent directory.
	Level  int   // The parent directory's nesting level.

	Url          string // The URL of the rendered HTML file.
	RootPrefix   string // The root path prefix.
	RootRelative string // Relative path to root dir.

	// Previous and next files in the directory.
	PrevFile *AFile
	NextFile *AFile

	// The content rendered from the markdown file.
	Content template.HTML

	// Meta-data from the markdown file below.
	Template string   // The template to use to render this file.
	Tags     []string // Tags for your use.
	Title    string   // The title.
	Author   string   // The author.

	// Timestamps for creation / modification of the content.
	Created  struct{ Year, Month, Day int }
	Modified struct{ Year, Month, Day int }
	
	// A few functions are also available:
	
	// UrlRelative: Return the relative path to file from baseFile. 
	UrlRelative(baseFile *AFile) string
	
	// HasTag: Return true if the file has the given tag.
	HasTag(tag string) bool
	
	// HasTagsAll: Return true if the file has all the given tags.
  	HasTagsAll(tags ...string) bool
  
  	// HasTagsAny: Return true if the file has any of the given tags.
	HasTagsAny(tags ...string) bool
	
	// BaseName: Return the bare html filename.
	BaseName() string
  
	// FirstParagraph: Return the first paragraph of the file. The returned HTML
	// will contain the opening and closing <p> tags.
	FirstParagraph() template.HTML
  
	// FormatCreated: Format the creation date using Go's date formatting function.
	// The reference time is "Mon Jan 2 15:04:05 MST 2006"
	FormatCreated(fmt string) string
  
	// FormatModified: The same as FormatCreated, but for the modification date.
	FormatModified(fmt string) string
}
```
