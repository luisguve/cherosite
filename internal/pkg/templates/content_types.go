package templates

import(
	"html/template"
	"strings"
	"fmt"
)

type Content interface {
	// Render returns the HTML representation of this content, according to its kind
	// and status.
	Render(idx int) template.HTML
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
	PublishDate string
	ThreadLink  string // Thread URL. It includes SectionLink
	SectionLink string // Section URL
}

// type for displaying content of a thread in its page
type ThreadContent struct {
	*BasicContent
	Replies  uint32
	SaveLink string // URL to post request to save thread
}

func (t *ThreadContent) Render(idx int) template.HTML {
	t.BasicContent.ClassName = fmt.Sprintf("%s-%d", t.BasicContent.Status, idx)
	t.BasicContent.UpvoteLink = fmt.Sprintf("%s/upvote/", t.BasicContent.ThreadLink)
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
	SaveLink string // URL to post request to save thread
	Replies  uint32
}

func (t *ThreadView) Render(idx int) template.HTML {
	t.BasicContent.ClassName = fmt.Sprintf("thread %s-%d", t.BasicContent.Status, 
		idx)
	t.BasicContent.UpvoteLink = fmt.Sprintf("%s/upvote/", t.BasicContent.ThreadLink)
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
	Id      string
	Replies uint32
}

func (c *CommentContent) Render(idx int) template.HTML {
	c.BasicContent.ClassName = fmt.Sprintf("%s-%d", c.BasicContent.Status, idx)
	c.BasicContent.UpvoteLink = fmt.Sprintf("%s/upvote/?c_id=%s", 
		c.BasicContent.ThreadLink, c.Id)
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

func (c *CommentView) Render(idx int) template.HTML {
	c.BasicContent.ClassName = fmt.Sprintf("comment %s-%d",	c.BasicContent.Status, 
		idx)
	c.BasicContent.UpvoteLink = fmt.Sprintf("%s/upvote/?c_id=%s", 
		c.BasicContent.ThreadLink, c.Id)
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

func (sc *SubcommentView) Render(idx int) template.HTML {
	sc.BasicContent.ClassName = fmt.Sprintf("subcomment %s-%d",	sc.BasicContent.Status,
		idx)
	sc.BasicContent.UpvoteLink = fmt.Sprintf("%s/upvote/?c_id=%s&sc_id=%s", 
		sc.BasicContent.ThreadLink, sc.CommentId, sc.Id)
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
