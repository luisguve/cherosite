package templates

import(
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

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
