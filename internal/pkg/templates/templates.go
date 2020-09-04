// package templates defines the set of types to hold the data that will be
// consumed by HTML templates, along with other types and helper functions to
// bind objects from protobuf messages and types for template rendering, and
// for holding page feeds and format them into values useful for pagination.
//
// It also defines the patterns of specified quality expected to get from the
// server into feeds.
//
// The template engine used to render the templates is html/template.
//
// The templates definitions are located in /web/.

package templates

import (
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pag "github.com/luisguve/cherosite/internal/pkg/pagination"
)

var baseURL *url.URL
var tpl *template.Template

func mustParseTemplates(dir string) *template.Template {
	templ := template.New("")
	filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if strings.Contains(path, ".html") {
			_, err = templ.ParseFiles(path)
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
	return templ
}

// AbsURL creates an absolute URL from the relative path given and the baseURL
// defined in the global variable.
func absURL(in string) string {
	url, err := url.Parse(in)
	if err != nil {
		log.Printf("Could not parse %s: %v\n", in, err)
		return in
	}
	if url.IsAbs() || strings.HasPrefix(in, "//") {
		return in
	}

	var stringBaseURL = baseURL.String()
	if strings.HasPrefix(in, "/") {
		// Strip trailing / from baseURL.
		last := len(stringBaseURL)
		stringBaseURL = stringBaseURL[:last]
	}
	return makePermalink(stringBaseURL, in).String()
}

// MakePermalink combines base URL with content path to create full URL paths.
// Example
//    base:   http://spf13.com/
//    path:   post/how-i-blog
//    result: http://spf13.com/post/how-i-blog
// *Borrowed from gohugo.io:
// https://github.com/gohugoio/hugo/blob/master/helpers/url.go
func makePermalink(host, plink string) *url.URL {

	base, err := url.Parse(host)
	if err != nil {
		panic(err)
	}

	p, err := url.Parse(plink)
	if err != nil {
		panic(err)
	}

	if p.Host != "" {
		panic(fmt.Errorf("can't make permalink from absolute link %q", plink))
	}

	base.Path = path.Join(base.Path, p.Path)

	// path.Join will strip off the last /, so put it back if it was there.
	hadTrailingSlash := (plink == "" && strings.HasSuffix(host, "/")) || strings.HasSuffix(p.Path, "/")
	if hadTrailingSlash && !strings.HasSuffix(base.Path, "/") {
		base.Path = base.Path + "/"
	}

	return base
}

func Setup(env, port, internalTplDir, publicTplDir string) *template.Template {
	if env == "local" {
		stringBaseURL := "http://localhost" + port + "/"
		var err error
		baseURL, err = url.Parse(stringBaseURL)
		if err != nil {
			log.Printf("Could not parse baseURL (%s): %v\n", stringBaseURL, err)
		}
	}
	tpl = mustParseTemplates(internalTplDir).Funcs(template.FuncMap{"absURL": absURL})
	publicTpl := mustParseTemplates(publicTplDir).Funcs(template.FuncMap{"absURL": absURL})
	return publicTpl
}

// ContentsFeed holds a list of *pbApi.ContentRule, representing a page feed.
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
		section := metadata.SectionId
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
