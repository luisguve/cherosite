package templates

import(
	"html/template"
	"strings"
	"fmt"
)

type Content interface {
	// Render returns the HTML representation of this content, according to its kind
	// and status.
	Render() template.HTML
	// Kind returns the content type, which may be thread, comment or subcomment
	Kind() string
	// Status returns the status of the content, which may be NEW, RELEVANT or TOP
	Status() string
}

// BasicContent is the set of fields that are shared by all the kinds of content:
// threads, comments and subcomments
type BasicContent struct {
	Title         string
	ContentType   string // thread, comment or subcomment
	ContentStatus string // NEW, RELEVANT or TOP
	Thumbnail     string // Thumbnail URL
	Permalink     string // Content URL
	Content       string
	Summary       string
	Upvotes       uint32
	Upvoted       bool // Has the current user topvote'd this content?
	SectionName   string
	Author        string // User alias
	PublishDate   string
	ThreadLink    string // Thread URL. It includes SectionLink
	SectionLink   string // Section URL
}

func (b *BasicContent) Kind() string {
	return b.ContentType
}

func (b *BasicContent) Status() string {
	return b.ContentStatus
}

type ThreadView struct {
	*BasicContent
	Replies uint32
}

func (t *ThreadView) Render() template.HTML {
	var tplName string
	switch t.BasicContent.Status() {
	case "NEW":
		tplName = "newcontent.html"
	case "RELEVANT":
		tplName = "relthread.html"
	case "TOP":
		tplName = "topthread.html"
	}
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, t); err != nil {
		return fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
	}
	return template.HTML(result.String())
}

type CommentView struct {
	*BasicContent
	Id      string
	Replies uint32
}

func (c *CommentView) Render() template.HTML {
	var tplName string
	switch c.BasicContent.Status() {
	case "NEW":
		tplName = "newcontent.html"
	case "RELEVANT":
		tplName = "relcomment.html"
	case "TOP":
		tplName = "topcomment.html"
	}
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, c); err != nil {
		return fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
	}
	return template.HTML(result.String())
}

type SubcommentView struct {
	*BasicContent
	CommentId string
	Id        string
}

func (sc *SubcommentView) Render() template.HTML {
	var tplName string
	switch sc.BasicContent.Status() {
	case "NEW":
		tplName = "newcontent.html"
	case "RELEVANT":
		tplName = "relsubcomment.html"
	case "TOP":
		tplName = "topsubcomment.html"
	}
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, sc); err != nil {
		return fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
	}
	return template.HTML(result.String())
}
