package router

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	pbTime "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/gorilla/mux"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbUsers "github.com/luisguve/cheroproto-go/userapi"
	"github.com/luisguve/cherosite/internal/pkg/pagination"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Section "/{section}" handler. It requests a set of threads using the identifier
// of the given section name, and displays a layout showing buttons for viewing profile
// and for creating a thread under the current section.
// That's the only difference between the logged in user and the non-logged in user
// views. It may return an error in case of the following:
// - wrong section name ------------------> 404 NOT FOUND
// - valid section name, but unavailable -> SECTION_UNAVAILABLE
// - network failures --------------------> INTERNAL_FAILURE
func (r *Router) handleViewSection(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]

	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	sectionCtx := formatContextSection(sectionId)

	contentPattern := &pbApi.ContentPattern{
		Pattern:        templates.FeedPattern,
		ContentContext: &pbApi.ContentPattern_SectionCtx{sectionCtx},
		// ignore DiscardIds, do not discard any thread
	}

	stream, err := section.Client.RecycleContent(context.Background(), contentPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// The section name is probably wrong.
				// log for debugging.
				log.Printf("Section %s not found\n", sectionId)
				http.NotFound(w, req)
				return
			case codes.Unavailable:
				log.Printf("Section %s temporarily unavailable\n", sectionId)
				http.Error(w, "SECTION_UNAVAILABLE", http.StatusNoContent)
				return
			default:
				log.Printf("Unknown code: %v - %s\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}

	feed, err := getFeed(stream)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
	}

	var userHeader *pbUsers.UserHeaderData
	userId := r.currentUser(req)
	if userId != "" {
		// A user is logged in. Get its data.
		userHeader = r.getUserHeaderData(w, userId)
	}
	sectionView := templates.DataToSectionView(feed.Contents, userHeader, userId, section.Name, sectionId)
	// update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pThreads := feed.GetSectionPaginationThreads()

			d.SectionThreads[sectionId] = pThreads
		})
	}

	if err := r.templates.ExecuteTemplate(w, "section.html", sectionView); err != nil {
		log.Printf("Could not execute template section.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle section "/{section}/recycle" handler. It returns a new feed for the
// section in JSON format. It may return an error in case of the following:
// - wrong section name ------------------> 404 NOT FOUND
// - valid section name, but unavailable -> SECTION_UNAVAILABLE
// - network or encoding failure ---------> INTERNAL_FAILURE
func (r *Router) handleRecycleSection(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]

	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	sectionCtx := formatContextSection(sectionId)

	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)

	contentPattern := &pbApi.ContentPattern{
		Pattern:        templates.FeedPattern,
		ContentContext: &pbApi.ContentPattern_SectionCtx{sectionCtx},
		DiscardIds:     discard.FormatSectionThreads(sectionId),
	}

	stream, err := section.Client.RecycleContent(context.Background(), contentPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// The section name is probably wrong.
				// log for debugging.
				log.Printf("Section %s not found\n", sectionId)
				http.NotFound(w, req)
				return
			case codes.Unavailable:
				log.Printf("Section %s temporarily unavailable\n", sectionId)
				http.Error(w, "SECTION_UNAVAILABLE", http.StatusNoContent)
				return
			default:
				log.Printf("Unknown code: %v - %s\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	feed, err := getFeed(stream)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
	}

	// update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pThreads := feed.GetSectionPaginationThreads()

			d.SectionThreads[sectionId] = append(d.SectionThreads[sectionId], pThreads...)
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Create thread "/{section}/new" handler. It handles the creation of content
// in a section through POSTing a form. It returns the permalink of the newly created
// thread on success, or an error in case of the following:
// - creating a thread in an invalid section -> 404 NOT_FOUND
// - missing ft_file input -------------------> MISSING_ft_file_INPUT
// - file greater than 64mb ------------------> FILE_TOO_BIG
// - corrupted file --------------------------> INVALID_FILE
// - file type other than image and gif ------> INVALID_FILE_TYPE
// - file creation/write failure -------------> CANT_WRITE_FILE
// - missing content (empty input) -----------> NO_CONTENT
// - missing title (empty input) -------------> NO_TITLE
// - user has already posted today -----------> USER_UNABLE_TO_POST
// - user unathenticated ---------------------> USER_UNREGISTERED
// - network failures ------------------------> INTERNAL_FAILURE
func (r *Router) handleNewThread(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]

	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	// Get ft_file and save it to the disk with a unique, random name.
	filePath, err, s := getAndSaveFile(req, "ft_file")
	if err != nil {
		http.Error(w, err.Error(), s)
		return
	}
	// Get the rest of the content parts
	content := req.FormValue("content")
	if content == "" {
		http.Error(w, "NO_CONTENT", http.StatusBadRequest)
		return
	}
	title := req.FormValue("title")
	if title == "" {
		http.Error(w, "NO_TITLE", http.StatusBadRequest)
		return
	}
	sectionCtx := formatContextSection(sectionId)
	createRequest := &pbApi.CreateThreadRequest{
		UserId:  userId,
		Content: &pbApi.Content{
			Title:   title,
			Content: content,
			FtFile:  filePath,
			PublishDate: &pbTime.Timestamp{
				Seconds: time.Now().Unix(),
			},
		},
		SectionCtx: sectionCtx,
	}
	res, err := section.Client.CreateThread(context.Background(), createRequest)
	if err != nil {
		resErr, ok := status.FromError(err)
		if ok {
			switch resErr.Code() {
			case codes.NotFound:
				// the section name is probably wrong; log message.
				log.Printf("Section %s not found\n", sectionId)
				http.NotFound(w, req)
				return
			// A user can create a limited number of threads daily.
			case codes.FailedPrecondition:
				log.Println("This user has already posted a thread today")
				http.Error(w, "USER_UNABLE_TO_POST", http.StatusPreconditionFailed)
				return
			case codes.Unauthenticated:
				log.Println("This user is unregistered")
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			default:
				log.Printf("Unknown code: %v - %s\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(res.Permalink))
}
