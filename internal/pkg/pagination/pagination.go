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
	UserActivity   map[string]Activity
	// FeedActivity maps user ids of authors of threads, comments and subcomments 
	// to ids of these kinds of content that compose the feed of the current user
	// that she has already seen
	FeedActivity   map[string]Activity
	// ThreadsSaved maps section names to the threads the user has already seen in its
	// saved area
	ThreadsSaved   map[string][]string
	// SectionThreads maps section names to the threads the user has already seen
	// in the section
	SectionThreads map[string][]string
	// ThreadComments maps thread ids to the comments the user has already seen
	// in the thread
	ThreadComments map[string][]string
	// GeneralThreads maps section names to threads ids.
	GeneralThreads map[string][]string
}

func (d *DiscardIds) FormatUserActivity(userId string) 
map[string]*pb.FullUserData_Activity {
	var fudActivity *pb.FullUserData_Activity

	// Set threads
	for t := range d.UserActivity[userId].ThreadsCreated {
		pbThread := &pb.Context_Thread{
			Id:         t.Id,
			SectionCtx: &pb.Context_section{
				Name: t.SectionName,
			},
		}
		fudActivity.ThreadsCreated = append(fudActivity.ThreadsCreated, pbThread)
	}
	// Set comments
	for c := range d.UserActivity[userId].Comments {
		pbComment := &pb.Context_Comment{
			Id:        c.Id,
			ThreadCtx: &pb.Context_Thread{
				Id:         c.Thread.Id,
				SectionCtx: &pb.Context_section{
					Name: c.Thread.SectionName,
				},
			},
		}
		fudActivity.Comments = append(fudActivity.Comments, pbComment)
	}
	// Set subcomments
	for sc := range d.UserActivity[userId].Subcomments {
		pbSubcomment := &pb.Context_Subcomment{
			Id:         sc.Id,
			CommentCtx: &pb.Context_Comment{
				Id:        sc.Comment.Id,
				ThreadCtx: &pb.Context_Thread{
					Id: sc.Comment.Thread.Id,
					SectionCtx: &pb.Context_section{
						Name: sc.Comment.Thread.SectionName,
					},
				},
			},
		}
		fudActivity.Subcomments = append(fudActivity.Subcomments, pbSubcomment)
	}
	return map[string]*pb.FullUserData_Activity{
		userId: fudActivity
	}
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
