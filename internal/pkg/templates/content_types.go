package templates

import(
	"html/template"
	"strings"
	"fmt"
)

// OveriewRenderer is the interface that types must implement in order to be displayed
// in section level pages, i.e. to build the "view"
type OverviewRenderer interface {
	// Render returns the HTML representation of the content, according to its kind
	// and status, with an appropiate class name to place the content accordingly.
	RenderOverview(idx int) template.HTML
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
	Title       string
	Status      string // NEW, RELEVANT or TOP
	ClassName   string
	UpvoteLink  string // URL to post upvote to content
	Thumbnail   string // Thumbnail URL
	Permalink   string // Content URL
	Content     string
	Summary     string
	Upvotes     uint32
	Upvoted     bool // Has the current user topvote'd this content?
	SectionName string
	Author      string // User alias
	Username    string // Author's username
	PublishDate string
	ThreadLink  string // Thread URL. It includes SectionLink
	SectionLink string // Section URL
}

// type for displaying content of a thread in section page level and single
// page level.
type Thread struct {
	*BasicContent
	Replies      uint32
	SaveLink     string // URL to post request to save thread
	UndoSaveLink string
	Saved        bool // Did the current user save this thread?
	ReplyLink    string // URL to post reply
}

func (t *Thread) RenderContent() template.HTML {
	t.BasicContent.ClassName = fmt.Sprintf("%s", t.BasicContent.Status)

	tplName := "thread_content.html"
	result := new(strings.Builder)
	if err := tpl.ExecuteTemplate(result, tplName, t); err != nil {
		errMsg := fmt.Sprintf("Could not execute template %s: %v\n", tplName, err)
		return template.HTML(errMsg)
	}
	return template.HTML(result.String())
}

func (t *Thread) RenderOverview(idx int) template.HTML {
	t.BasicContent.ClassName = fmt.Sprintf("thread %s-%d", t.BasicContent.Status, 
		idx)

	var tplName string
	switch t.BasicContent.Status {
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
	Id        string
	Replies   uint32
	ReplyLink string // URL to post reply
}

func (c *CommentContent) RenderOverview(idx int) template.HTML {
	c.BasicContent.ClassName = fmt.Sprintf("%s-%d", c.BasicContent.Status, idx)

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

func (c *CommentView) RenderOverview(idx int) template.HTML {
	c.BasicContent.ClassName = fmt.Sprintf("comment %s-%d",	c.BasicContent.Status, 
		idx)

	var tplName string
	switch c.BasicContent.Status {
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

// type for displaying content of a subcomment in section level page
type SubcommentView struct {
	*BasicContent
	CommentId string
	Id        string
}

func (sc *SubcommentView) RenderOverview(idx int) template.HTML {
	sc.BasicContent.ClassName = fmt.Sprintf("subcomment %s-%d",	sc.BasicContent.Status,
		idx)
	
	var tplName string
	switch sc.BasicContent.Status {
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
