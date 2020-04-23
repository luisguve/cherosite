package pagination

import(
	"encoding/gob"
)

func RegisterSessionTypes() {
	gob.Register(&DiscardIds{})
}

type DiscardIds struct {
	// FeedThreads holds the set of thread ids the user has already seen in its 
	// dashboard feed.
	FeedThreads    []string
	// SectionThreads maps section names to the threads the user has already seen
	// in the section
	SectionThreads map[string][]string
	// ThreadComments maps thread ids to the comments the user has already seen
	// in the thread
	ThreadComments map[string][]string
	// GeneralThreads maps section names to threads ids.
	GeneralThreads map[string][]string
}

// thread ids are canonical
