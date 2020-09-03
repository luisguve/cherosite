package router

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbUsers "github.com/luisguve/cheroproto-go/userapi"
	"github.com/luisguve/cherosite/internal/pkg/pagination"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Thread "/{section}/{thread}" handler. It looks for a thread using its identifier
// under the given section name, and displays a layout showing buttons for
// viewing profile, creating a thread and submitting a comment on the current thread.
// That's the only difference between the logged in user and the non-logged in user
// views. It may return an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - template rendering failures -------> TEMPLATE_ERROR
// - netwotk failures ------------------> INTERNAL_FAILURE
func (r *Router) handleViewThread(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	request := &pbApi.GetThreadRequest{
		Thread: threadCtx,
	}
	// Load thread
	content, err := section.Client.GetThread(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// Section name or thread id are probably wrong.
				// Log for debugging.
				log.Printf("Could not find thread (id: %s) on section %s\n",
					thread, sectionId)
				http.NotFound(w, req)
				return
			case codes.Unavailable:
				// Section unavailable
				log.Printf("Section %s unavailable\n", sectionId)
				http.Error(w, "SECTION_UNAVAILABLE", http.StatusNoContent)
				return
			default:
				log.Printf("Unknown error code %v: %v\n", resErr.Code(),
					resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	var feed templates.ContentsFeed
	// Load comments only if there are comments on this thread
	if content.Metadata.Replies > 0 {
		// Request to load comments
		contentPattern := &pbApi.ContentPattern{
			Pattern:        templates.CommentPattern,
			ContentContext: &pbApi.ContentPattern_ThreadCtx{threadCtx},
			// ignore DiscardIds; do not discard any comment
		}
		stream, err := section.Client.RecycleContent(context.Background(), contentPattern)
		if err != nil {
			log.Printf("Could not send request: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		} else {
			feed, err = getFeed(stream)
			if err != nil {
				log.Printf("An error occurred while getting feed: %v\n", err)
				w.WriteHeader(http.StatusPartialContent)
			}
		}
	}
	// Update session only if there are comments.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pComments := feed.GetPaginationComments()

			d.ThreadComments[thread] = pComments
		})
	}

	// get current user data for header section
	userId := r.currentUser(req)
	var userHeader *pbUsers.UserHeaderData
	if userId != "" {
		// A user is logged in. Get its data.
		userHeader = r.getUserHeaderData(w, userId)
	}

	threadView := templates.DataToThreadView(content, feed.Contents, userHeader, userId, sectionId)

	if err := r.templates.ExecuteTemplate(w, "thread.html", threadView); err != nil {
		log.Printf("Could not execute template thread.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle thread comments "/{section}/{thread}/recycle" handler.
// It returns a new feed of comments for the thread in JSON format.
// It may return an error in the following cases:
// - invalid section name or thread id -> 404 NOT_FOUND
// - no more comments are available ----> OUT_OF_RANGE
// - network or encoding failures ------> INTERNAL_FAILURE
func (r *Router) handleRecycleComments(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	discardIds := getDiscardIds(session)

	contentPattern := &pbApi.ContentPattern{
		Pattern:        templates.CommentPattern,
		ContentContext: &pbApi.ContentPattern_ThreadCtx{threadCtx},
		DiscardIds:     discardIds.FormatThreadComments(thread),
	}
	var feed templates.ContentsFeed

	stream, err := section.Client.RecycleContent(context.Background(), contentPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Invalid section id %s or thread id %s\n", sectionId, thread)
				http.NotFound(w, req)
				return
			case codes.OutOfRange:
				log.Println("OOR: no more comments on this thread are available")
				http.Error(w, "OUT_OF_RANGE", http.StatusNoContent)
				return
			default:
				log.Printf("Unknown code %v: %v\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Could not send request: %v\n", err)
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
		}
	} else {
		var err error
		feed, err = getFeed(stream)
		if err != nil {
			log.Printf("An error occurred while getting feed: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	// update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pComments := feed.GetPaginationComments()

			d.ThreadComments[thread] = append(d.ThreadComments[thread], pComments...)
		})
	}
	// Get current user id.
	userId := r.currentUser(req)
	res := templates.FeedToBytes(feed.Contents, userId, false)
	contentLength := strconv.Itoa(len(res))
	w.Header().Set("Content-Length", contentLength)
	w.Header().Set("Content-Type", "text/html")
	if _, err = w.Write(res); err != nil {
		log.Println("Recycle comments: could not send response:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Save thread "/{section}/{thread}/save" handler. It adds the thread id
// to the list of saved threads of the given user, whose id is provided.
// It returns OK on success or an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleSave(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	request := &pbApi.SaveThreadRequest{
		UserId: userId,
		Thread: threadCtx,
	}
	_, err := section.Client.SaveThread(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Invalid section id %s or thread id %s\n", sectionId, thread)
				http.NotFound(w, req)
				return
			case codes.Unavailable:
				http.Error(w, "SECTION_UNAVAILABLE", http.StatusNoContent)
				return
			default:
				log.Printf("Unknown code %v: %v\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Could not send request: %v\n", err)
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Undo save thread "/{section}/{thread}/undosave" handler. It removes the thread
// id from the list of saved threads of the given user, whose id is provided.
// It returns OK on success or an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleUndoSave(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	undoSaveRequest := &pbApi.UndoSaveThreadRequest{
		UserId: userId,
		Thread: threadCtx,
	}
	_, err := section.Client.UndoSaveThread(context.Background(), undoSaveRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Invalid section id %s or thread id %s\n", sectionId, thread)
				http.NotFound(w, req)
				return
			default:
				log.Printf("Unknown code %v: %v\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Could not send request: %v\n", err)
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Delete Thread "/{section}/{thread}/delete/" handler. It deletes the thread
// and all the content related to it (i.e. comments, subcomments and any link
// pointing to the thread) from the database and returns OK on success or an
// error in case of the following:
// - invalid section name or thread id ---> 404 NOT_FOUND
// - user id and author id are not equal -> UNAUTHORIZED
// - network failures --------------------> INTERNAL_FAILURE
func (r *Router) handleDeleteThread(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	deleteRequest := &pbApi.DeleteContentRequest{
		UserId:         userId,
		ContentContext: &pbApi.DeleteContentRequest_ThreadCtx{threadCtx},
	}
	r.handleDelete(w, req, deleteRequest, section.Client)
}

// Post Upvote "/{section}/{thread}/upvote/" handler. It leverages the operation of
// submitting the upvote to the method handleUpvote, which returns OK on success or
// an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleUpvoteThread(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	upvoteRequest := &pbApi.UpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UpvoteRequest_ThreadCtx{threadCtx},
	}

	r.handleUpvote(w, req, upvoteRequest, section.Client)
}

// Post upvote undoing "/{section}/{thread}/undoupvote/" handler. It leverages
// the operation of submitting the un-upvote to the method handleUpvote, which
// returns OK on success or an error in case of the following:
// - invalid section name or thread id ------> 404 NOT_FOUND
// - user did not upvote the content before -> NOT_UPVOTED
// - network failures -----------------------> INTERNAL_FAILURE
func (r *Router) handleUndoUpvoteThread(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s not found\n", sectionId)
		http.NotFound(w, req)
		return
	}

	threadCtx := formatContextThread(sectionId, thread)

	undoUpvoteRequest := &pbApi.UndoUpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UndoUpvoteRequest_ThreadCtx{threadCtx},
	}

	r.handleUndoUpvote(w, req, undoUpvoteRequest, section.Client)
}
