package router

import(
	"log"
	"errors"
	"net/http"
	"strconv"
	"context"
	"encoding/json"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/gorilla/mux"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	"github.com/luisguve/cheropatilla/internal/pkg/templates"
)

// Subcomments "/{section}/{thread}/comment/?c_id={c_id}&offset={offset}" handler.
// It returns 10 subcomments on a given comment (c_id) on a given thread, on a given 
// section.
// The offset query parameter indicates how many subcomments to skip, since these data 
// is stored and returned in sequential order. It may return an error in case of the 
// following: 
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
	section := vard["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	commentCtx := formatContextComment(section, thread, commentId)

	request := &pbApi.GetSubcommentsRequest{
		Offset:     uint32(offset),
		CommentCtx: commentCtx,
	}

	stream, err := r.crudClient.GetComments(context.Background(), request)
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

	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
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
	section := vars["section"]
	threadId := vars["thread"]
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
	thread := formatContextThread(section, threadId)
	postCommentRequest := &pbApi.CommentRequest{
		Content:        content,
		FtFile:         filePath,
		UserId:         userId,
		PublishDate:    time.Now().Unix(),
		ContentContext: &pbApi.CommentRequest_ThreadCtx{thread},
	}
	r.handleComment(w, req, postCommentRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]

	comment := formatContextComment(section, thread, commentId)
	deleteContentRequest := &pbApi.DeleteContentRequest{
		UserId:         userId,
		ContentContext: &pbApi.DeleteContentRequest_CommentCtx{comment},
	}
	r.handleDelete(w, req, deleteContentRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
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
	comment := formatContextComment(section, thread, commentId)
	postCommentRequest := &pbApi.CommentRequest{
		Content:        content,
		FtFile:         filePath,
		PublishDate:    time.Now().Unix(),
		UserId:         userId,
		ContentContext: &pbApi.CommentRequest_CommentCtx{comment},
	}
	r.handleComment(w, req, postCommentRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	comment := vars["c_id"]
	subcommentId := vars["sc_id"]

	subcomment := formatContextSubcomment(section, thread, comment, subcommentId)
	deleteRequest := &pbApi.DeleteContentRequest{
		UserId:         userId,
		ContentContext: &pbApi.DeleteContentRequest_SubcommentCtx{subcomment},
	}
	r.handleDelete(w, req, deleteRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]

	comment := formatContextComment(section, thread, commentId)
	upvoteRequest := &pbApi.UpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UpvoteRequest_CommentCtx{comment},
	}
	r.handleUpvote(w, req, upvoteRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]
	subcommentId := vars["sc_id"]

	subcomment := formatContextSubcomment(section, thread, commentId, subcommentId)
	upvoteRequest := &pbApi.UpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UpvoteRequest_SubcommentCtx{subcomment},
	}
	r.handleUpvote(w, req, upvoteRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	commentId := vars["c_id"]

	comment := formatContextComment(section, thread, commentId)
	undoUpvoteRequest := &pbApi.UndoUpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UndoUpvoteRequest_CommentCtx{comment},
	}
	r.handleUndoUpvote(w, req, undoUpvoteRequest)
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
	section := vars["section"]
	thread := vars["thread"]
	comment := vars["c_id"]
	subcommentId := vars["sc_id"]

	subcomment := formatContextSubcomment(section, thread, commentId, subcommentId)
	undoUpvoteRequest := &pbApi.UndoUpvoteRequest{
		UserId:         userId,
		ContentContext: &pbApi.UndoUpvoteRequest_SubcommentCtx{subcomment},
	}
	r.handleUndoUpvote(w, req, undoUpvoteRequest)
}
