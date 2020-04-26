package templates

import(
	"html/template"
	
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

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
	ThreadsSaved   []*pb.FullContentData `json="threads_saved"`
	Following      uint32 `json="following"`
	Followers      uint32 `json="followers"`
	Feed           FeedContent `json="feed_content"`
}

type ProfileView struct {
	Username       string
	BasicData      UserInfo
	ThreadsCreated []*pb.FullContentData `json="threads_created"`
	Following      uint32 `json="following"`
	Followers      uint32 `json="followers"`
}

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

func New() *template.Template {
	return template.Must(template.ParseGlob("*.gohtml"))
}
