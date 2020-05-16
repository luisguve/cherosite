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

// GetPaginationActivity formats a ContentsFeed object into a map of UserIds,
// i.e. the authors of each content to pagination.Activity, i.e. their contents.
func (cf ContentsFeed) GetPaginationActivity() map[string]p.Activity {
	pActivity := make(map[string]p.Activity)

	for activity := range cf.Contents {
		userId := activity.Data.Author.Id

		switch ctx := activity.ContentContext.(type) {
		case *pb.ContentRule_ThreadCtx:
			// content type: THREAD
			thread := p.Thread{
				Id:          ctx.Id,
				SectionName: ctx.SectionCtx.Name,
			}
			pActivity[userId].ThreadsCreated = append(pActivity[userId].ThreadsCreated, thread)
		case *pb.ContentRule_CommentCtx:
			// content type: COMMENT
			comment := p.Comment{
				Id:     ctx.Id,
				Thread: p.Thread{
					SectionName: ctx.TheadCtx.SectionCtx.Name,
					Id:          ctx.TheadCtx.Id,
				},
			}
			pActivity[userId].Comments = append(pActivity[userId].Comments, comment)
		case *pb.ContentRule_SubcommentCtx:
			// content type: SUBCOMMENT
			subcomment := p.Subcomment{
				Id:      ctx.Id,
				Comment: p.Comment{
					Id:     ctx.CommentCtx.Id,
					Thread: p.Thread{
						SectionName: ctx.CommentCtx.ThreadCtx.SectionCtx.Name,
						Id:          ctx.CommentCtx.ThreadCtx.Id,
					},
				},
			}
			pActivity[userId].Subcomments = append(pActivity[userId].Subcomments, subcomment)
		}
	}
	return pActivity
}

// GetPaginationThreads returns a map of section names to thread ids
func (cf ContentsFeed) GetPaginationThreads() map[string][]string {
	result := make(map[string][]string)

	for content := range cf.Contents {
		metadata := content.Data.Metadata
		section := metadata.Section
		id := metadata.Id
		result[section] = append(result[section], id)
	}
	return result
}

// GetPaginationComments returns a map of thread ids to comment ids
func (cf ContentsFeed) GetPaginationComments() map[string][]string {
	result := make(map[string][]string)

	for content := range cf.Contents {
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
