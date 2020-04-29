package pagination

import(
	"encoding/gob"
)

func RegisterSessionTypes() {
	gob.Register(&DiscardIds{})
}

type DiscardIds struct {
	// UserActivity holds threads, comments and subcomments created by the user that 
	// she has already seen in its feed
	UserActivity   Activity
	// FeedActivity maps user ids of authors of threads, comments and subcomments 
	// to ids of these kinds of content that compose the feed of the current user
	// that she has already seen
	FeedActivity   map[string]Activity
	// SectionThreads maps section names to the threads the user has already seen
	// in the section
	SectionThreads map[string][]string
	// ThreadComments maps thread ids to the comments the user has already seen
	// in the thread
	ThreadComments map[string][]string
	// GeneralThreads maps section names to threads ids.
	GeneralThreads map[string][]string
}

type Activity struct {
	Subcomments    []Subcomment
	Comments       []Comment
	ThreadsCreated []Thread
}

// A thread is in a section and has an id
type Thread struct {
	SectionName string
	Id          string
}

// A comment is in a thread and has an id
type Comment struct {
	Thread
	Id string
}

// A subcomment is in a comment and has an id
type Subcomment struct {
	Comment
	Id string
}

// thread ids are canonical
