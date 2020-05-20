package templates

import(
	"html/template"

	p "github.com/luisguve/cheropatilla/internal/pkg/pagination"
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

var tpl *template.Template

func Setup() *template.Template {
	tpl = template.Must(template.ParseGlob("/web/internal/templates/*.html"))
	return template.Must(template.ParseGlob("/web/templates/*.html"))
}

type ContentsFeed struct {
	Contents []*pb.ContentRule
}

// (boilerplate) formatComment converts a *pb.ContentRule_CommentCtx into a
// pagination.Comment object.
func formatComment(comment *pb.ContentRule_CommentCtx) p.Comment {
	return p.Comment{
		Id:     comment.Id,
		Thread: p.Thread{
			SectionName: comment.TheadCtx.SectionCtx.Name,
			Id:          comment.TheadCtx.Id,
		},
	}
}

// (boilerplate) formatSubcomment converts a *pb.ContentRule_SubcommentCtx
// into a pagination.Subcomment object.
func formatSubcomment(sc *pb.ContentRule_SubcommentCtx) p.Subcomment {
	return p.Subcomment{
		Id:      sc.Id,
		Comment: p.Comment{
			Id:     sc.CommentCtx.Id,
			Thread: p.Thread{
				SectionName: sc.CommentCtx.ThreadCtx.SectionCtx.Name,
				Id:          sc.CommentCtx.ThreadCtx.Id,
			},
		},
	}
}

// (boilerplate) formatThread converts a *pb.ContentRule_ThreadCtx into a
// pagination.Thread object.
func formatThread(thread *pb.ContentRule_ThreadCtx) p.Thread {
	return p.Thread{
		Id:          thread.Id,
		SectionName: thread.SectionCtx.Name,
	}
}

// GetUserPaginationActivity formats a ContentsFeed object holding contents
// from a single user to a pagination.Activity object.
func (cf ContentsFeed) GetUserPaginationActivity() p.Activity {
	var pActivity p.Activity

	for _, activity := range cf.Contents {
		switch ctx := activity.ContentContext.(type) {
		case *pb.ContentRule_ThreadCtx:
			// content type: THREAD
			thread := formatThread(ctx)
			pActivity.ThreadsCreated = append(pActivity.ThreadsCreated, thread)
		case *pb.ContentRule_CommentCtx:
			// content type: COMMENT
			comment := formatComment(ctx)
			pActivity.Comments = append(pActivity.Comments, comment)
		case *pb.ContentRule_SubcommentCtx:
			// content type: SUBCOMMENT
			sc := formatSubcomment(ctx)
			pActivity.Subcomments = append(pActivity.Subcomments, sc)
		}
	}
	return pActivity
}

// GetPaginationActivity formats a ContentsFeed object holding contents from
// different users into a map of UserIds, i.e. the authors of each content to
// pagination.Activity, i.e. their contents.
func (cf ContentsFeed) GetPaginationActivity() map[string]p.Activity {
	pActivity := make(map[string]p.Activity)

	for _, activity := range cf.Contents {
		userId := activity.Data.Author.Id

		switch ctx := activity.ContentContext.(type) {
		case *pb.ContentRule_ThreadCtx:
			// content type: THREAD
			thread := formatThread(ctx)
			pActivity[userId].ThreadsCreated = append(pActivity[userId].ThreadsCreated, thread)
		case *pb.ContentRule_CommentCtx:
			// content type: COMMENT
			comment := formatComment(ctx)
			pActivity[userId].Comments = append(pActivity[userId].Comments, comment)
		case *pb.ContentRule_SubcommentCtx:
			// content type: SUBCOMMENT
			sc := formatSubcomment(ctx)
			pActivity[userId].Subcomments = append(pActivity[userId].Subcomments, sc)
		}
	}
	return pActivity
}

// GetPaginationThreads formats a ContentsFeed object holding threads from
// different sections into a map of section names to thread ids.
func (cf ContentsFeed) GetPaginationThreads() map[string][]string {
	result := make(map[string][]string)

	for _, content := range cf.Contents {
		metadata := content.Data.Metadata
		section := metadata.Section
		id := metadata.Id
		result[section] = append(result[section], id)
	}
	return result
}

// GetPaginationComments formats a ContentsFeed object holding comments from
// different threads into a map of thread ids to comment ids.
func (cf ContentsFeed) GetPaginationComments() map[string][]string {
	result := make(map[string][]string)

	for _, content := range cf.Contents {
		ctx, ok := content.ContentContext.(*pb.ContentRule_CommentCtx)
		if !ok {
			continue
		}
		threadId := ctx.ThreadCtx.Id
		commentId := ctx.Id
		result[threadId] = append(result[threadId], commentId)
	}
	return result
}
