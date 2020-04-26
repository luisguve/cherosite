package templates

import(
	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
)

type DashboardView struct {
	FullUserData   *pb.FullUserData `json="user_data"`
	ThreadsCreated []*pb.FullContentData `json="threads_created"`
	ThreadsSaved   []*pb.FullContentData `json="threads_saved"`
	Following      uint32 `json="following"`
	Followers      uint32 `json="followers"`
	Feed           FeedContent `json="feed_content"`
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
	ContentPatternResponse []*pb.ContentRuleResponse `json="content_pattern"`
	ErrorMsg               string `json="error_message"`
}

type FeedSubcomments struct {
	Subcomments []*pb.Subcomment `json="subcomments"`
	ErrorMsg    string `json="error_message"`
}

var FeedPattern = []*pb.ContentRuleRequest{
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_TOP},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_RELEVANT},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
	&pb.ContentRuleRequest{Status: pb.ContentStatus_NEW},
}