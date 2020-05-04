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
	// Id returns the Id of the content, used to uniquely identify it
	Id() string
}

// BasicContent is the set of fields that are shared by all the kinds of content:
// threads, comments and subcomments
type BasicContent struct {
	Title         string
	ContentType   string // thread, comment or subcomment
	ContentStatus string // NEW, RELEVANT or TOP
	ContentId     string
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

func (b *BasicContent) Id() string {
	return b.ContentId
}

// type for displaying content of a thread in its page
type ThreadContent struct {
	*BasicContent
	Replies uint32
}

func (t *ThreadContent) Render() template.HTML {
	tplName := "thread_content.html"
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, t); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}

// type for displaying content of a thread in section level page
type ThreadView struct {
	*BasicContent
	Replies uint32
}

func (t *ThreadView) Render() template.HTML {
	var tplName string
	switch t.BasicContent.Status() {
	case "NEW":
		tplName = "new_content.html"
	case "RELEVANT":
		tplName = "rel_thread.html"
	case "TOP":
		tplName = "top_thread.html"
	}
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, t); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}

// type for displaying content of a comment in the page of the thread it belongs to
type CommentContent struct {
	*BasicContent
	Id      string
	Replies uint32
}

func (c *CommentContent) Render() template.HTML {
	tplName := "comment_content.html"
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, c); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}

// type for displaying content of a comment in section level page
type CommentView struct {
	*BasicContent
	Id      string
	Replies uint32
}

func (c *CommentView) Render() template.HTML {
	var tplName string
	switch c.BasicContent.Status() {
	case "NEW":
		tplName = "new_content.html"
	case "RELEVANT":
		tplName = "rel_comment.html"
	case "TOP":
		tplName = "top_comment.html"
	}
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, c); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
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
		tplName = "new_content.html"
	case "RELEVANT":
		tplName = "rel_subcomment.html"
	case "TOP":
		tplName = "top_subcomment.html"
	}
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, sc); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}
