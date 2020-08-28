package templates

import (
	pbMetadata "github.com/luisguve/cheroproto-go/metadata"
)

var FeedPattern = []pbMetadata.ContentStatus{
	pbMetadata.ContentStatus_TOP, // 1
	pbMetadata.ContentStatus_NEW, // 2
	pbMetadata.ContentStatus_REL, // 3
	pbMetadata.ContentStatus_NEW, // 4
	pbMetadata.ContentStatus_REL, // 5
	pbMetadata.ContentStatus_REL, // 6
	pbMetadata.ContentStatus_NEW, // 7
	pbMetadata.ContentStatus_NEW, // 8
	pbMetadata.ContentStatus_REL, // 9
	pbMetadata.ContentStatus_REL, // 10
	pbMetadata.ContentStatus_REL, // 11
}

var CommentPattern = []pbMetadata.ContentStatus{
	pbMetadata.ContentStatus_TOP, // 1
	pbMetadata.ContentStatus_REL, // 2
	pbMetadata.ContentStatus_REL, // 3
	pbMetadata.ContentStatus_REL, // 4
	pbMetadata.ContentStatus_NEW, // 5
	pbMetadata.ContentStatus_NEW, // 6
	pbMetadata.ContentStatus_NEW, // 7
	pbMetadata.ContentStatus_REL, // 8
}

var CompactPattern = []pbMetadata.ContentStatus{
	pbMetadata.ContentStatus_NEW, // 1
	pbMetadata.ContentStatus_REL, // 2
	pbMetadata.ContentStatus_TOP, // 3
	pbMetadata.ContentStatus_REL, // 4
	pbMetadata.ContentStatus_REL, // 5
	pbMetadata.ContentStatus_NEW, // 6
	pbMetadata.ContentStatus_NEW, // 7
	pbMetadata.ContentStatus_NEW, // 8
}
