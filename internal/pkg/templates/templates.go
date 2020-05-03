package templates

import(
	"fmt"
	"strings"
	"html/template"
	
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

var tpl *template.Template

type UserInfo struct {
	Alias           string
	Username        string
	PicUrl          string
	About           string
	LastTimeCreated int64
}

type DashboardView struct {
	FullUserData   *pb.FullUserData `json="user_data"`
	ThreadsCreated []*pb.FullContentData `json="threads_created"`
	ThreadsSaved   []*pb.ThreadData `json="threads_saved"`
	Following      uint32 `json="following"`
	Followers      uint32 `json="followers"`
	Feed           FeedContent `json="feed_content"`
}

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

type ProfileData struct {
	Patillavatar string // URL to user profile pic
	Alias        string
	Username     string
	Followers    uint32
	Following    uint32
	Description  string
	Activity     []*Content
}

type Notif struct {
	Permalink string
	Title     string
	Message   string
	Date      string
}

// UserHeader holds information about the user currently logged in
type UserHeader struct {
	Alias         string
	Notifications []*Notif
}

type RecycleType struct {
	// Content type identifier
	Name string
	// Link to send request to recycle content
	Link string
}

// HeaderData holds information to render the header section of a page.
type HeaderData struct {
	User *UserHeader
	// A page shows its content grouped together in different sections, 
	// e.g. the dashboard contains feed, user activity and user saved content, 
	// but profile pages contains only user activity.
	// RecycleTypes holds the possible content types a user can select to recycle.
	RecycleTypes []RecycleType
}

type ProfileView struct {
	HeaderData
	ProfileData
	// IsFollower indicates whether the current user is following another user,
	// in a context in which it is viewing another user's profile or content
	IsFollower    bool
}

type CurrentUserData struct {
	Followers    uint32
	Following    uint32
	Activity     []*Content
	SavedContent []*Content
}

type DashboardView struct {
	HeaderData
	CurrentUserData
	Feed []*Content
}

type ExploreView struct {
	HeaderData
	Feed []*Content
}

/*old below*/

type ThreadView struct {
	Username string `json="username"`
	Content  *pb.FullContentData `json="content"`
	Feed     FeedContent `json="feed_content"`
}

type SectionView struct {
	Username string `json="username"`
	Feed     FeedContent `json="feed_content"`
}

type FeedContent struct {
	Feed
	ContentIds []string `json="-"`
}

type FeedGeneral struct {
	Feed
	ContentIds map[string][]string `json="-"`
}

type Feed struct {
	ContentPattern []*pb.ContentRuleResponse `json="content_pattern"`
	ErrorMsg       string `json="error_message"`
}

type FeedSubcomments struct {
	Subcomments []*pb.Subcomment `json="subcomments"`
	ErrorMsg    string `json="error_message"`
}

func Setup() *template.Template {
	tpl = template.Must(template.ParseGlob("/web/internal/templates/*.html"))
	return template.Must(template.ParseGlob("/web/templates/*.html"))
}
