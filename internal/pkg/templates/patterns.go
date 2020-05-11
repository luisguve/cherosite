package templates

import(
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

var FeedPattern = []pb.ContentStatus{
	pb.ContentStatus_NEW, // 1
	pb.ContentStatus_NEW, // 2
	pb.ContentStatus_NEW, // 3
	pb.ContentStatus_TOP, // 4
	pb.ContentStatus_NEW, // 5
	pb.ContentStatus_NEW, // 6
	pb.ContentStatus_NEW, // 7
	pb.ContentStatus_NEW, // 8
	pb.ContentStatus_REL, // 9
	pb.ContentStatus_NEW, // 10
	pb.ContentStatus_REL, // 11
	pb.ContentStatus_NEW, // 12
	pb.ContentStatus_NEW, // 13
	pb.ContentStatus_REL, // 14
	pb.ContentStatus_REL, // 15
	pb.ContentStatus_REL, // 16
	pb.ContentStatus_NEW, // 17
	pb.ContentStatus_NEW, // 18
	pb.ContentStatus_NEW, // 19
	pb.ContentStatus_NEW, // 20
	pb.ContentStatus_REL, // 21
	pb.ContentStatus_REL, // 22
	pb.ContentStatus_REL, // 23
	pb.ContentStatus_NEW, // 24
}

var CommentPattern = []pb.ContentStatus{
	pb.ContentStatus_REL, // 1
	pb.ContentStatus_TOP, // 2
	pb.ContentStatus_NEW, // 3
	pb.ContentStatus_REL, // 4
	pb.ContentStatus_NEW, // 5
	pb.ContentStatus_NEW, // 6
	pb.ContentStatus_NEW, // 7
	pb.ContentStatus_REL, // 8
	pb.ContentStatus_NEW, // 9
	pb.ContentStatus_NEW, // 10
	pb.ContentStatus_REL, // 11
	pb.ContentStatus_REL, // 12
	pb.ContentStatus_NEW, // 13
	pb.ContentStatus_NEW, // 14
}

var CompactPattern = []pb.ContentStatus{
	pb.ContentStatus_NEW, // 1
	pb.ContentStatus_NEW, // 2
	pb.ContentStatus_REL, // 3
	pb.ContentStatus_TOP, // 4
	pb.ContentStatus_REL, // 5
	pb.ContentStatus_NEW, // 6
	pb.ContentStatus_NEW, // 7
}
