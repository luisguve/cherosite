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

// GetPaginationActivity converts an ActivityFeed object into a pagination.Activity 
// object
func (cf ContentsFeed) GetPaginationActivity() p.Activity {
	var pActivity p.Activity

	for activity := range cf.Contents {
		switch content := activity.ContentContext.(type) {
		case *pb.ContentRule_ThreadCtx:
			// content type: THREAD
			ctx := content.ThreadCtx
			thread := p.Thread{
				Id:          ctx.Id,
				SectionName: ctx.SectionCtx.Name,
			}
			pActivity.ThreadsCreated = append(pActivity.ThreadsCreated, thread)
		case *pb.ContentRule_CommentCtx:
			// content type: COMMENT
			ctx := content.CommentCtx
			comment := p.Comment{
				Id:     ctx.Id,
				Thread: p.Thread{
					SectionName: ctx.TheadCtx.SectionCtx.Name,
					Id:          ctx.TheadCtx.Id,
				},
			}
			pActivity.Comments = append(pActivity.Comments, comment)
		case *pb.ContentRule_SubcommentCtx:
			// content type: SUBCOMMENT
			ctx := content.SubcommentCtx
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
			pActivity.Subcomments = append(pActivity.Subcomments, subcomment)
		}
	}
}

// GetPaginationThreads returns thread ids mapped to their section names
func (cf ContentsFeed) GetPaginationThreads() map[string][]string {
	var result map[string][]string

	for content := range cf.Contents {
		metadata := content.Data.Metadata
		section := metadata.Section
		id := metadata.Id
		result[section] = append(result[section], id)
	}
	return result
}

// GetPaginationComments returns comment ids mapped to their thread ids
func (cf ContentsFeed) GetPaginationComments() map[string][]string {
	var result map[string][]string

	for content := range cf.Contents {
		metadata := content.Data.Metadata
		threadId := metadata.Id

		ctx, ok := content.ContentContext.(*pb.ContentRule_CommentCtx)
		if !ok {
			return result
		}
		commentId := ctx.Id
		result[threadId] = append(result[threadId], commentId)
	}
	return result
}