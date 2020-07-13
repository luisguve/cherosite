package templates

import (
	pbMetadata "github.com/luisguve/cheroproto-go/metadata"
)

var FeedPattern = []pbMetadata.ContentStatus{
	pbMetadata.ContentStatus_NEW, // 1
	pbMetadata.ContentStatus_NEW, // 2
	pbMetadata.ContentStatus_NEW, // 3
	pbMetadata.ContentStatus_TOP, // 4
	pbMetadata.ContentStatus_NEW, // 5
	pbMetadata.ContentStatus_NEW, // 6
	pbMetadata.ContentStatus_NEW, // 7
	pbMetadata.ContentStatus_NEW, // 8
	pbMetadata.ContentStatus_REL, // 9
	pbMetadata.ContentStatus_NEW, // 10
	pbMetadata.ContentStatus_REL, // 11
	pbMetadata.ContentStatus_NEW, // 12
	pbMetadata.ContentStatus_NEW, // 13
	pbMetadata.ContentStatus_REL, // 14
	pbMetadata.ContentStatus_REL, // 15
	pbMetadata.ContentStatus_REL, // 16
	pbMetadata.ContentStatus_NEW, // 17
	pbMetadata.ContentStatus_NEW, // 18
	pbMetadata.ContentStatus_NEW, // 19
	pbMetadata.ContentStatus_NEW, // 20
	pbMetadata.ContentStatus_REL, // 21
	pbMetadata.ContentStatus_REL, // 22
	pbMetadata.ContentStatus_REL, // 23
	pbMetadata.ContentStatus_NEW, // 24
}

var CommentPattern = []pbMetadata.ContentStatus{
	pbMetadata.ContentStatus_REL, // 1
	pbMetadata.ContentStatus_TOP, // 2
	pbMetadata.ContentStatus_NEW, // 3
	pbMetadata.ContentStatus_REL, // 4
	pbMetadata.ContentStatus_NEW, // 5
	pbMetadata.ContentStatus_NEW, // 6
	pbMetadata.ContentStatus_NEW, // 7
	pbMetadata.ContentStatus_REL, // 8
	pbMetadata.ContentStatus_NEW, // 9
	pbMetadata.ContentStatus_NEW, // 10
	pbMetadata.ContentStatus_REL, // 11
	pbMetadata.ContentStatus_REL, // 12
	pbMetadata.ContentStatus_NEW, // 13
	pbMetadata.ContentStatus_NEW, // 14
}

var CompactPattern = []pbMetadata.ContentStatus{
	pbMetadata.ContentStatus_NEW, // 1
	pbMetadata.ContentStatus_NEW, // 2
	pbMetadata.ContentStatus_REL, // 3
	pbMetadata.ContentStatus_TOP, // 4
	pbMetadata.ContentStatus_REL, // 5
	pbMetadata.ContentStatus_NEW, // 6
	pbMetadata.ContentStatus_NEW, // 7
}
