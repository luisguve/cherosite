// package templates defines the set of types to hold the data that will be
// consumed by HTML templates, along with other types and helper functions to
// bind objects from protobuf messages and types for template rendering, and
// for holding page feeds and format them into values useful for pagination.
//
// It also defines the patterns of specified quality expected to get from the
// server into feeds.
//
// The template engine used to render the templates is html/templates.
//
// The templates definitions are located in /web/.

package templates

import (
	"html/template"

	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pag "github.com/luisguve/cherosite/internal/pkg/pagination"
)

var tpl *template.Template

func Setup() *template.Template {
	tpl = template.Must(template.ParseGlob("web/internal/templates/*.html"))
	return template.Must(template.ParseGlob("web/templates/*.html"))
}

// ContentsFeed holds a list of *pbApi.ContentRule, representing a page feed of
// some kind.
type ContentsFeed struct {
	Contents []*pbApi.ContentRule
}

// GetUserPaginationActivity formats a ContentsFeed object holding contents
// from a single user to a pagination.Activity object.
func (cf ContentsFeed) GetUserPaginationActivity() pag.Activity {
	var pActivity pag.Activity

	for _, activity := range cf.Contents {
		switch ctx := activity.ContentContext.(type) {
		case *pbApi.ContentRule_ThreadCtx:
			// content type: THREAD
			thread := pag.FormatThread(ctx)
			pActivity.ThreadsCreated = append(pActivity.ThreadsCreated, thread)
		case *pbApi.ContentRule_CommentCtx:
			// content type: COMMENT
			comment := pag.FormatComment(ctx)
			pActivity.Comments = append(pActivity.Comments, comment)
		case *pbApi.ContentRule_SubcommentCtx:
			// content type: SUBCOMMENT
			sc := pag.FormatSubcomment(ctx)
			pActivity.Subcomments = append(pActivity.Subcomments, sc)
		}
	}
	return pActivity
}

// GetPaginationActivity formats a ContentsFeed object holding contents from
// different users into a map of UserIds, i.e. the authors of each content to
// pagination.Activity, i.e. their contents.
func (cf ContentsFeed) GetPaginationActivity() map[string]pag.Activity {
	pActivity := make(map[string]pag.Activity)

	for _, activity := range cf.Contents {
		userId := activity.Data.Author.Id

		switch ctx := activity.ContentContext.(type) {
		case *pbApi.ContentRule_ThreadCtx:
			// content type: THREAD
			thread := pag.FormatThread(ctx)
			a := pActivity[userId]
			a.ThreadsCreated = append(a.ThreadsCreated, thread)
			pActivity[userId] = a
		case *pbApi.ContentRule_CommentCtx:
			// content type: COMMENT
			comment := pag.FormatComment(ctx)
			a := pActivity[userId]
			a.Comments = append(a.Comments, comment)
			pActivity[userId] = a
		case *pbApi.ContentRule_SubcommentCtx:
			// content type: SUBCOMMENT
			subcom := pag.FormatSubcomment(ctx)
			a := pActivity[userId]
			a.Subcomments = append(a.Subcomments, subcom)
			pActivity[userId] = a
		}
	}
	return pActivity
}

// GetSectionPaginationThreads formats a ContentsFeed object holding threads
// from a single section into a slice of thread ids.
func (cf ContentsFeed) GetSectionPaginationThreads() []string {
	var threadIds []string

	for _, content := range cf.Contents {
		metadata := content.Data.Metadata
		threadIds = append(threadIds, metadata.Id)
	}
	return threadIds
}

// GetPaginationThreads formats a ContentsFeed object holding threads from
// different sections into a map of section names to thread ids.
func (cf ContentsFeed) GetPaginationThreads() map[string][]string {
	ids := make(map[string][]string)

	for _, content := range cf.Contents {
		metadata := content.Data.Metadata
		section := metadata.Section
		id := metadata.Id
		ids[section] = append(ids[section], id)
	}
	return ids
}

// GetPaginationComments formats a ContentsFeed object holding comments from
// a single thread into a slice of comment ids.
func (cf ContentsFeed) GetPaginationComments() []string {
	var commentIds []string

	for _, content := range cf.Contents {
		ctx, ok := content.ContentContext.(*pbApi.ContentRule_CommentCtx)
		if !ok {
			continue
		}
		commentId := ctx.CommentCtx.Id
		commentIds = append(commentIds, commentId)
	}
	return commentIds
}
