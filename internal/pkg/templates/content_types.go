package templates

import (
	"fmt"
	"html/template"
	"strings"
)

// OveriewRenderer is the interface that types must implement in order to be displayed
// in section level pages, i.e. to build the "view"
type OverviewRenderer interface {
	// Render returns the HTML representation of the content, according to its kind
	// and status, with an appropiate class name to place the content accordingly.
	RenderOverview(idx int, showSection bool) template.HTML
}

// ContentRenderer is the interface that types must implement in order to be displayed
// in single page level, i.e. to build the "content". It is intended for types that
// have their own page.
type ContentRenderer interface {
	// Render returns the HTML representation of the content, according to its kind
	// and status. Note that it does not receive an idx to place the content in
	// the page, since the whole page is dedicated to the content being rendered.
	RenderContent() template.HTML
}

// BasicContent is the set of fields that are shared by all the kinds of content:
// threads, comments and subcomments
type BasicContent struct {
	Title          string
	Status         string // NEW, REL or TOP
	ClassName      string
	UpvoteLink     string // URL to post upvote to content
	UndoUpvoteLink string // URL to post undo content upvote
	Thumbnail      string // Thumbnail URL
	Permalink      string // Content URL
	Content        string
	Summary        string
	LongerSummary  string
	Upvotes        uint32
	Upvoted        bool // Has the current user topvoted this content?
	ShowSection    bool // Whether to show the section name and link
	SectionName    string
	Author         string // User alias
	Username       string // Author's username
	PublishDate    string
	ThreadLink     string // Thread URL. It includes SectionLink
	SectionLink    string // Section URL
}

// type for displaying content of a thread in section page level and single
// page level.
type Thread struct {
	*BasicContent
	Replies        uint32
	SaveLink       string // URL to post request to save thread
	UndoSaveLink   string
	ShowSaveOption bool   // Whether to render the save button
	Saved          bool   // Did the current user save this thread?
	ReplyLink      string // URL to post reply
}

func (t *Thread) RenderContent() template.HTML {
	t.BasicContent.ClassName = "thread"

	tplName := "thread_content.html"
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, t); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}

func (t *Thread) RenderOverview(idx int, showSection bool) template.HTML {
	var idxS string
	if idx < 10 {
		idxS = fmt.Sprintf("0%d", idx)
	} else {
		idxS = fmt.Sprintf("%d", idx)
	}
	t.BasicContent.ClassName = fmt.Sprintf("thread %s-%s", t.BasicContent.Status,
		idxS)
	t.BasicContent.ShowSection = showSection

	var tplName string
	switch t.BasicContent.Status {
	case "NEW":
		tplName = "new_content.html"
	case "REL":
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
	Id        string
	Replies   uint32
	ReplyLink string // URL to post reply
}

func (c *CommentContent) RenderOverview(idx int, showSection bool) template.HTML {
	var idxS string
	if idx < 10 {
		idxS = fmt.Sprintf("0%d", idx)
	} else {
		idxS = fmt.Sprintf("%d", idx)
	}
	c.BasicContent.ClassName = fmt.Sprintf("%s-%s", c.BasicContent.Status, idxS)
	c.BasicContent.ShowSection = showSection

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

func (c *CommentView) RenderOverview(idx int, showSection bool) template.HTML {
	var idxS string
	if idx < 10 {
		idxS = fmt.Sprintf("0%d", idx)
	} else {
		idxS = fmt.Sprintf("%d", idx)
	}
	c.BasicContent.ClassName = fmt.Sprintf("comment %s-%s", c.BasicContent.Status,
		idxS)
	c.BasicContent.ShowSection = showSection

	var tplName string
	switch c.BasicContent.Status {
	case "NEW":
		tplName = "new_content.html"
	case "REL":
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

// type for displaying content of a subcomment in section level page
type SubcommentView struct {
	*BasicContent
	CommentId string
	Id        string
}

func (sc *SubcommentView) RenderOverview(idx int, showSection bool) template.HTML {
	var idxS string
	if idx < 10 {
		idxS = fmt.Sprintf("0%d", idx)
	} else {
		idxS = fmt.Sprintf("%d", idx)
	}
	sc.BasicContent.ClassName = fmt.Sprintf("subcomment %s-%s", sc.BasicContent.Status,
		idxS)
	sc.BasicContent.ShowSection = showSection

	var tplName string
	switch sc.BasicContent.Status {
	case "NEW":
		tplName = "new_content.html"
	case "REL":
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

// type to be used when a given ContentRule has no data.
type NoContent struct {
	ClassName string
}

func (nc *NoContent) RenderOverview(idx int, _ bool) template.HTML {
	nc.ClassName = fmt.Sprintf("%s-%d", "no-content", idx)
	tplName := "no_content.html"

	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, nc); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}

func (nc *NoContent) RenderContent() template.HTML {
	nc.ClassName = "no-content"
	tplName := "no_content.html"

	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, nc); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}
