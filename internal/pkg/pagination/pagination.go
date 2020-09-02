package pagination

import (
	"encoding/gob"
	"sync"

	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbContext "github.com/luisguve/cheroproto-go/context"
	pbDataFormat "github.com/luisguve/cheroproto-go/dataformat"
)

func init() {
	gob.Register(&DiscardIds{})
}

type DiscardIds struct {
	// UserActivity holds threads, comments and subcomments created by the user that
	// she has already seen in its feed
	UserActivity map[string]Activity
	// FeedActivity maps user ids of authors of threads, comments and subcomments
	// to ids of these kinds of content that compose the feed of the current user
	// that she has already seen
	FeedActivity map[string]Activity
	// SavedThreads maps section names to the threads the user has already seen in its
	// saved area
	SavedThreads map[string][]string
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
func (d *DiscardIds) FormatSectionThreads(sectionName string) []string {
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
// map[string]*pbApi.IdList to be used in a request to recycle general threads.
func (d *DiscardIds) FormatGeneralThreads() map[string]*pbApi.IdList {
	result := make(map[string]*pbApi.IdList)
	for section, threadIds := range d.GeneralThreads {
		result[section] = &pbApi.IdList{
			Ids: threadIds,
		}
	}
	return result
}

// FormatSavedThreads converts the field SavedThreads into a map[string]*pbApi.IdList
// to be used in a request to recycle saved threads of a user.
func (d *DiscardIds) FormatSavedThreads() map[string]*pbApi.IdList {
	result := make(map[string]*pbApi.IdList)
	for section, threadIds := range d.SavedThreads {
		result[section] = &pbApi.IdList{
			Ids: threadIds,
		}
	}
	return result
}

// FormatUserActivity converts the field UserActivity into a
// map[string]*pbDataFormat.Activity to be used in a request to recycle activity,
// formatting the threads created, comments and subcomments in the given Activity
// object of the given key in UserActivity into a *pbDataFormat.Activity.
// It uses the given userId as the key to the activity of the given user.
func (d *DiscardIds) FormatUserActivity(userId string) map[string]*pbDataFormat.Activity {
	pbActivity := make(map[string]*pbDataFormat.Activity)
	pbActivity[userId] = formatActivity(d.UserActivity[userId])
	return pbActivity
}

// FormatFeedActivity converts the field FeedActivity into a
// map[string]*pbDataFormat.Activity to be used in a request to recycle activity,
// formatting the threads created, comments and subcomments in the given Activity
// object of  each key in FeedActivity into a *pbDataFormat.Activity.
// It uses the given userIds as the keys to the activity of the given users.
func (d *DiscardIds) FormatFeedActivity(userIds []string) map[string]*pbDataFormat.Activity {
	pbActivity := make(map[string]*pbDataFormat.Activity)
	for _, userId := range userIds {
		pbActivity[userId] = formatActivity(d.FeedActivity[userId])
	}
	return pbActivity
}

// formatActivity formats the threads created, comments and subcomments in the
// given Activity object into a *pbDataFormat.Activity
func formatActivity(activity Activity) *pbDataFormat.Activity {
	var wg sync.WaitGroup
	pbActivity := &pbDataFormat.Activity{}
	// Set threads
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, t := range activity.ThreadsCreated {
			pbThread := &pbContext.Thread{
				Id: t.Id,
				SectionCtx: &pbContext.Section{
					Id: t.SectionName,
				},
			}
			pbActivity.ThreadsCreated = append(pbActivity.ThreadsCreated, pbThread)
		}
	}()
	// Set comments
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, c := range activity.Comments {
			pbComment := &pbContext.Comment{
				Id: c.Id,
				ThreadCtx: &pbContext.Thread{
					Id: c.Thread.Id,
					SectionCtx: &pbContext.Section{
						Id: c.Thread.SectionName,
					},
				},
			}
			pbActivity.Comments = append(pbActivity.Comments, pbComment)
		}
	}()
	// Set subcomments
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, sc := range activity.Subcomments {
			pbSubcomment := &pbContext.Subcomment{
				Id: sc.Id,
				CommentCtx: &pbContext.Comment{
					Id: sc.Comment.Id,
					ThreadCtx: &pbContext.Thread{
						Id: sc.Comment.Thread.Id,
						SectionCtx: &pbContext.Section{
							Id: sc.Comment.Thread.SectionName,
						},
					},
				},
			}
			pbActivity.Subcomments = append(pbActivity.Subcomments, pbSubcomment)
		}
	}()
	wg.Wait()
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

// FormatComment converts a *pbApi.ContentRule_CommentCtx into a Comment object
// for pagination.
func FormatComment(comCtx *pbApi.ContentRule_CommentCtx) Comment {
	comment := comCtx.CommentCtx
	return Comment{
		Id: comment.Id,
		Thread: Thread{
			SectionName: comment.ThreadCtx.SectionCtx.Id,
			Id:          comment.ThreadCtx.Id,
		},
	}
}

// FormatSubcomment converts a *pbApi.ContentRule_SubcommentCtx into a Subcomment
// object for pagination.
func FormatSubcomment(subcCtx *pbApi.ContentRule_SubcommentCtx) Subcomment {
	sc := subcCtx.SubcommentCtx
	return Subcomment{
		Id: sc.Id,
		Comment: Comment{
			Id: sc.CommentCtx.Id,
			Thread: Thread{
				SectionName: sc.CommentCtx.ThreadCtx.SectionCtx.Id,
				Id:          sc.CommentCtx.ThreadCtx.Id,
			},
		},
	}
}

// FormatThread converts a *pbApi.ContentRule_ThreadCtx into a Thread object
// for pagination.
func FormatThread(threadCtx *pbApi.ContentRule_ThreadCtx) Thread {
	thread := threadCtx.ThreadCtx
	return Thread{
		Id:          thread.Id,
		SectionName: thread.SectionCtx.Id,
	}
}
