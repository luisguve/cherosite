package router

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	pbTime "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/gorilla/mux"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Subcomments "/{section}/{thread}/comment/?c_id={c_id}&offset={offset}" handler.
// It returns 10 subcomments on a given comment (c_id) on a given thread, on a given
// section in HTML format.
// The offset query parameter indicates how many subcomments to skip, since these data
// is stored and returned in sequential order. It may return an error in case of the
// following:
// - invalid section id ---------------------------------------> 404 NOT FOUND
// - negative or non-number offset query parameter ------------> INVALID_OFFSET
// - offset is out of range; there are not that much comments -> OFFSET_OOR
// - network or encoding failures -----------------------------> INTERNAL_FAILURE
func (r *Router) handleGetSubcomments(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	offset, err := strconv.Atoi(vars["offset"])
	if err != nil || offset < 0 {
		log.Printf("offset (%v) is not valid\n", offset)
		http.Error(w, "INVALID_OFFSET", http.StatusBadRequest)
		return
	}
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}
	commentCtx := formatContextComment(sectionId, thread, commentId)

	request := &pbApi.GetSubcommentsRequest{
		Offset:     uint32(offset),
		CommentCtx: commentCtx,
	}

	stream, err := section.Client.GetSubcomments(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("Could not find content: %v", resErr.Message())
				http.NotFound(w, req)
				return
			case codes.OutOfRange:
				log.Printf("Offset is out of range: %v\n", resErr.Message())
				http.Error(w, "OFFSET_OOR", http.StatusBadRequest)
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
	feed, err := getFeed(stream)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
	}

	// Get current user id.
	userId := r.currentUser(req)

	res := templates.SubcommentsToBytes(feed.Contents, userId)
	contentLength := strconv.Itoa(len(res))
	w.Header().Set("Content-Length", contentLength)
	w.Header().Set("Content-Type", "text/html")

	if _, err = w.Write(res); err != nil {
		log.Println("Get subcomments: could not send response:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Post Comment "/{section}/{thread}/comment/" handler. It handles the posting
// of a comment in a given thread in a given section through POSTing a form.
// As opposed to creating a thread, when posting a comment it is optional to submit
// a ft_file, and a title isn't submitted. Also note that a user is allowed to create
// one single thread per day, but can comment multiple times on different threads.
// It returns "OK" on success, or an error in case of the following:
// - invalid section or thread ----------> 404 NOT_FOUND
// - file greater than 64mb -------------> FILE_TOO_BIG
// - corrupted file ---------------------> INVALID_FILE
// - file type other than image and gif -> INVALID_FILE_TYPE
// - file creation/write failure --------> CANT_WRITE_FILE
// - missing content (empty input) ------> NO_CONTENT
// - network failures -------------------> INTERNAL_FAILURE
func (r *Router) handlePostComment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	threadId := vars["thread"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}
	// Get ft_file and save it to the disk with a unique, random name.
	filePath, err, status := getAndSaveFile(req, "ft_file")
	if err != nil {
		// It's ok to get an errMissingFile, but if it's not such an error, it is
		// an internal failure.
		if !errors.Is(err, errMissingFile) {
			http.Error(w, err.Error(), status)
			return
		}
	}
	// Get the rest of the content parts
	content := req.FormValue("content")
	if content == "" {
		http.Error(w, "NO_CONTENT", http.StatusBadRequest)
		return
	}
	thread := formatContextThread(sectionId, threadId)
	postCommentRequest := &pbApi.CommentRequest{
		Content: content,
		FtFile:  filePath,
		UserId:  userId,
		PublishDate: &pbTime.Timestamp{
			Seconds: time.Now().Unix(),
		},
		ContentContext: &pbApi.CommentRequest_ThreadCtx{thread},
	}
	r.handleComment(w, req, postCommentRequest, section.Client)
}

// Delete Comment "/{section}/{thread}/comment/delete/?c_id={c_id}" handler.
// It deletes the comment and all the content associated to it (i.e.
// replies and any link pointing to the comment) from the database and
// returns OK on success or an error in case of the following:
// - invalid section name or thread id ---> 404 NOT_FOUND
// - user id and author id are not equal -> UNAUTHORIZED
// - network failures --------------------> INTERNAL_FAILURE
func (r *Router) handleDeleteComment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	comment := formatContextComment(sectionId, thread, commentId)
	deleteContentRequest := &pbApi.DeleteContentRequest{
		UserId:         userId,
		ContentContext: &pbApi.DeleteContentRequest_CommentCtx{comment},
	}
	r.handleDelete(w, req, deleteContentRequest, section.Client)
}

// Post Subcomment "/{section}/{thread}/comment/?c_id={c_id}" handler. It handles the
// submit of a subcomment on a given comment on a given thread on a given section
// through POSTing a form.
// As opposed to creating a thread, when posting a subcomment it is optional to submit
// a ft_file, and a title isn't submitted. Also note that a user is allowed to create
// one single thread per day, but can comment multiple times on different comments.
// It returns "OK" on success, or an error in case of the following:
// - invalid section, thread or comment -> 404 NOT_FOUND
// - file greater than 64mb -------------> FILE_TOO_BIG
// - corrupted file ---------------------> INVALID_FILE
// - file type other than image and gif -> INVALID_FILE_TYPE
// - file creation/write failure --------> CANT_WRITE_FILE
// - missing content (empty input) ------> NO_CONTENT
// - network failures -------------------> INTERNAL_FAILURE
func (r *Router) handlePostSubcomment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}
	// Get ft_file and save it to the disk with a unique, random name.
	filePath, err, status := getAndSaveFile(req, "ft_file")
	if err != nil {
		// It's ok to get an errMissingFile, but if it's not such an error, it is
		// an internal failure.
		if !errors.Is(err, errMissingFile) {
			http.Error(w, err.Error(), status)
			return
		}
	}
	// Get the rest of the content parts
	content := req.FormValue("content")
	if content == "" {
		http.Error(w, "NO_CONTENT", http.StatusBadRequest)
		return
	}
	comment := formatContextComment(sectionId, thread, commentId)
	postCommentRequest := &pbApi.CommentRequest{
		Content: content,
		FtFile:  filePath,
		PublishDate: &pbTime.Timestamp{
			Seconds: time.Now().Unix(),
		},
		UserId:         userId,
		ContentContext: &pbApi.CommentRequest_CommentCtx{comment},
	}
	r.handleComment(w, req, postCommentRequest, section.Client)
}

// Delete Subcomment
// "/{section}/{thread}/comment/delete/?c_id={c_id}&sc_id={sc_id}" handler.
// It deletes the subcomment and all the content associated to it (i.e. any
// link pointing to the subcomment) from the database and returns OK on
// success or an error in case of the following:
// - invalid section name or thread id ---> 404 NOT_FOUND
// - user id and author id are not equal -> UNAUTHORIZED
// - network failures --------------------> INTERNAL_FAILURE
func (r *Router) handleDeleteSubcomment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	comment := vars["c_id"]
	subcommentId := vars["sc_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	subcomment := formatContextSubcomment(sectionId, thread, comment, subcommentId)
	deleteRequest := &pbApi.DeleteContentRequest{
		UserId:         userId,
		ContentContext: &pbApi.DeleteContentRequest_SubcommentCtx{subcomment},
	}
	r.handleDelete(w, req, deleteRequest, section.Client)
}

// Post Upvote "/{section}/{thread}/upvote/?c_id={c_id}" handler.
// It leverages the operation of submitting the upvote to the method handleUpvote
// which returns OK on success or an error in case of the following:
// - invalid section name, thread id or comment -> 404 NOT_FOUND
// - section, thread or comment are unavailable -> SECTION_UNAVAILABLE
// - network failures ---------------------------> INTERNAL_FAILURE
func (r *Router) handleUpvoteComment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	comment := formatContextComment(sectionId, thread, commentId)
	upvoteRequest := &pbApi.UpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UpvoteRequest_CommentCtx{comment},
	}
	r.handleUpvote(w, req, upvoteRequest, section.Client)
}

// Post Upvote "/{section}/{thread}/upvote/?c_id={c_id}&sc_id={sc_id}" handler.
// It leverages the operation of submitting the upvote to the method handleUpvote,
// which returns OK on success or an error in case of the following:
// - invalid section name, thread id or comment -> 404 NOT_FOUND
// - section, thread or comment are unavailable -> SECTION_UNAVAILABLE
// - network failures ---------------------------> INTERNAL_FAILURE
func (r *Router) handleUpvoteSubcomment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	subcommentId := vars["sc_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	subcomment := formatContextSubcomment(sectionId, thread, commentId, subcommentId)
	upvoteRequest := &pbApi.UpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UpvoteRequest_SubcommentCtx{subcomment},
	}
	r.handleUpvote(w, req, upvoteRequest, section.Client)
}

// Post upvote undoing "/{section}/{thread}/unupvote/?c_id={c_id}" handler.
// It leverages the operation of submitting the un-upvote to the method
// handleUpvote, which returns OK on success or an error in case of the
// following:
// - invalid section name or thread id ------> 404 NOT_FOUND
// - user did not upvote the content before -> NOT_UPVOTED
// - network failures -----------------------> INTERNAL_FAILURE
func (r *Router) handleUndoUpvoteComment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	comment := formatContextComment(sectionId, thread, commentId)
	undoUpvoteRequest := &pbApi.UndoUpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UndoUpvoteRequest_CommentCtx{comment},
	}
	r.handleUndoUpvote(w, req, undoUpvoteRequest, section.Client)
}

// Post upvote undoing "/{section}/{thread}/unupvote/?c_id={c_id}&sc_id={sc_id}"
// handler. It leverages the operation of submitting the un-upvote to the
// method handleUpvote, which returns OK on success or an error in case of
// the following:
// - invalid section name or thread id ------> 404 NOT_FOUND
// - user did not upvote the content before -> NOT_UPVOTED
// - network failures -----------------------> INTERNAL_FAILURE
func (r *Router) handleUndoUpvoteSubcomment(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	sectionId := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	subcommentId := vars["sc_id"]
	// Get section client.
	section, ok := r.sections[sectionId]
	if !ok {
		log.Printf("Section %s is not in Router's sections map.\n", sectionId)
		http.NotFound(w, req)
		return
	}

	subcomment := formatContextSubcomment(sectionId, thread, commentId, subcommentId)
	undoUpvoteRequest := &pbApi.UndoUpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UndoUpvoteRequest_SubcommentCtx{subcomment},
	}
	r.handleUndoUpvote(w, req, undoUpvoteRequest, section.Client)
}
