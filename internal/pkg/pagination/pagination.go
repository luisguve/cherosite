package pagination

import(
	"encoding/gob"
)

func init() {
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
	// SavedThreads maps section names to the threads the user has already seen in its
	// saved area
	SavedThreads   map[string][]string
	// SectionThreads maps section names to the threads the user has already seen
	// in the section
	SectionThreads map[string][]string
	// ThreadComments maps thread ids to the comments the user has already seen
	// in the thread
	ThreadComments map[string][]string
	// GeneralThreads maps section names to threads ids.
	GeneralThreads map[string][]string
}

// FormatSectionThreads is an utility function to get and return the thread
// ids on a given section name (SectionThreads). Alternatively, you can access
// SectionThreads on a DiscardIds instance and get the threads by using
// the section name as the key.
func(d *DiscardIds) FormatSectionThreads(sectionName string) []string {
	return d.SectionThreads[sectionName]
}

// FormatThreadComments is an utility function to get and return the comment
// ids on a given thread id (ThreadComments). Alternatively, you can access
// ThreadComments on a DiscardIds instance and get the comments by using
// the threadId as the key.
func (d *DiscardIds) FormatThreadComments(threadId string) []string {
	return d.ThreadComments[threadId]
}

// FormatGeneralThreads converts the field GeneralThreads into a 
// map[string]*pb.IdList to be used in a request to recycle general threads.
func (d *DiscardIds) FormatGeneralThreads() map[string]*pb.IdList {
	result := make(map[string]*pb.IdList)
	for section, threadIds := range d.GeneralThreads {
		result[section] = &pb.IdList{
			Ids: threadIds,
		}
	}
	return result
}

// FormatSavedThreads converts the field SavedThreads into a map[string]*pb.IdList
// to be used in a request to recycle saved threads of a user.
func (d *DiscardIds) FormatSavedThreads() map[string]*pb.IdList {
	result := make(map[string]*pb.IdList)
	for section, threadIds := range d.SavedThreads {
		result[section] = &pb.IdList{
			Ids: threadIds,
		}
	}
	return result
}

// FormatUserActivity converts the field UserActivity into a 
// map[string]*pb.Activity to be used in a request to recycle activity, formatting
// the threads created, comments and subcomments in the given Activity object of 
// the given key in UserActivity into a *pb.Activity.
// It uses the given userId as the key to the activity of the given user.
func (d *DiscardIds) FormatUserActivity(userId string) map[string]*pb.Activity {
	pbActivity := make(map[string]*pb.Activity)
	pbActivity[userId] = formatActivity(d.UserActivity[userId])
	return pbActivity
}

// FormatFeedActivity converts the field FeedActivity into a 
// map[string]*pb.Activity to be used in a request to recycle activity, formatting
// the threads created, comments and subcomments in the given Activity object of 
// each key in FeedActivity into a *pb.Activity.
// It uses the given userIds as the keys to the activity of the given users.
func (d *DiscardIds) FormatFeedActivity(userIds []string) map[string]*pb.Activity {
	pbActivity := make(map[string]*pb.Activity)
	for _, userId := range userIds {
		pbActivity[userId] = formatActivity(d.FeedActivity[userId])
	}
	return pbActivity
}

// formatActivity formats the threads created, comments and subcomments in the
// given Activity object into a *pb.Activity
func formatActivity(activity Activity) *pb.Activity {
	var pbActivity *pb.Activity
	// Set threads
	for _, t := range activity.ThreadsCreated {
		pbThread := &pb.Context_Thread{
			Id:         t.Id,
			SectionCtx: &pb.Context_section{
				Name: t.SectionName,
			},
		}
		pbActivity.ThreadsCreated = append(pbActivity.ThreadsCreated, pbThread)
	}
	// Set comments
	for _, c := range activity.Comments {
		pbComment := &pb.Context_Comment{
			Id:        c.Id,
			ThreadCtx: &pb.Context_Thread{
				Id:         c.Thread.Id,
				SectionCtx: &pb.Context_section{
					Name: c.Thread.SectionName,
				},
			},
		}
		pbActivity.Comments = append(pbActivity.Comments, pbComment)
	}
	// Set subcomments
	for _, sc := range activity.Subcomments {
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
		pbActivity.Subcomments = append(pbActivity.Subcomments, pbSubcomment)
	}
	return pbActivity
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
