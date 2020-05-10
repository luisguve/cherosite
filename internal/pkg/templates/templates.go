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

type ActivityFeed struct {
	Activity []*pb.ActivityRule
}

// GetPaginationActivity converts an ActivityFeed object into a pagination.Activity 
// object
func (af ActivityFeed) GetPaginationActivity() p.Activity {
	var pActivity p.Activity

	for activity := range af.Activity {
		switch content := activity.ContentContext.(type) {
		case *pb.ActivityRule_ThreadCtx:
			// content type: THREAD
			ctx := content.ThreadCtx
			thread := p.Thread{
				Id:          ctx.Id,
				SectionName: ctx.SectionCtx.Name,
			}
			pActivity.ThreadsCreated = append(pActivity.ThreadsCreated, thread)
		case *pb.ActivityRule_CommentCtx:
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
		case *pb.ActivityRule_SubcommentCtx:
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
