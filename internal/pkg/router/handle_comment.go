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
	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
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
	commentId := vars["c_id"]
	thread := vars["thread"]
	section := vard["section"]

	subcommentsReq := &pb.GetSubcommentsRequest{
		Offset: uint32(offset),
		CommentCtx: &pb.Context_Comment{
			CommentId: commentId, 
			ThreadCtx: &pb.Context_Thread{
				ThreadId: thread,
				SectionCtx: &pb.Context_Section{
					SectionName: section,
				},
			},
		},
	}
	// Send request
	stream, err := r.crudClient.GetComments(context.Background(), subcommentsReq)
	if err != nil {
		log.Printf("Failed to communicate to server: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}

	var feed templates.FeedSubcomments

	// Continuously receive responses
	for {
		subcomment, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// check for server failures or bad request
			if resErr, ok := status.FromError(err); ok {
				switch resErr.Code() {
				case codes.NotFound:
					log.Printf("Could not find resource: %v", resErr.Message())
					http.NotFound(w, r)
					return
				case codes.OutOfRange:
					log.Printf("Offset is out of range: %v\n", resErr.Message())
					http.Error(w, "OFFSET_OOR", http.StatusBadRequest)
					return
				case codes.Internal:
					log.Printf("Internal server failure: %v", resErr.Message())
				default:
					log.Printf("Unknown error code %v: %v\n", resErr.Code(), 
					resErr.Message())
				}
			} else {
				errMsg := fmt.Sprintf("Error receiving response from stream: %v\n", err)
				log.Printf("%v", errMsg)
				feed.ErrorMsg = errMsg
			}
			w.WriteHeader(http.StatusInternalServerError)
			break
		}
		feed.Subcomments = append(feed.Subcomments, subcomment)
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Post Comment "/{section}/{thread}/comment/" handler. It handles the creation
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
// - user unathenticated ----------------> USER_UNREGISTERED
// - network failures -------------------> INTERNAL_FAILURE
func (r *Router) handlePostComment(userId string, w http.ResponseWriter, 
	req *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]
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
	postCommentRequest := &pb.CommentRequest{
		Data:       &pb.BasicContentData{
			PublishDate: time.Now().Unix(),
			Content:     content,
			FtFile:      filePath,
			AuthorId:    userId,
		},
		ContentContext: &pb.Context_Thread{
			ThreadId: thread,
			SectionCtx: &pb.Context_Section{
				SectionName: section,
			},
		},
	}
	err = r.postComment(postCommentRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// section or thread not found
				http.NotFound(w, r)
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
		log.Printf("Could not send request to Comment: %v\n", err)
		http.Error(w, errInternalFailure.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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
// - user unathenticated ----------------> USER_UNREGISTERED
// - network failures -------------------> INTERNAL_FAILURE
func (r *Router) handlePostSubcomment(userId string, w http.ResponseWriter, 
	req *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]
	comment := vars["c_id"]
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
	postCommentRequest := &pb.CommentRequest{
		Data:           &pb.BasicContentData{
			PublishDate: time.Now().Unix(),
			Content:     content,
			FtFile:      filePath,
			AuthorId:    userId,
		},
		ContentContext: &pb.Context_Comment{
			CommentId: comment,
			ThreadCtx: &pb.Context_Thread{
				ThreadId: thread,
				SectionCtx: &pb.Context_Section{
					SectionName: section,
				},
			},
		},
	}
	err = r.postComment(postCommentRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// section, thread or comment not found
				http.NotFound(w, r)
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
		log.Printf("Could not send request to Comment: %v\n", err)
		http.Error(w, errInternalFailure.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (r *Router) postComment (postCommentRequest *pb.CommentRequest) error {
	stream, err := r.crudClient.Comment(context.Background(), postCommentRequest)
	if err != nil {
		return err
	}
	// Continuously receive notifications and the user ids they are for.
	for {
		notifyUser, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving response from stream: %v\n", err)
			break
		}
		userId := notifyUser.userId
		notification := notifyUser.Notification
		// send notification
		go r.hub.Broadcast(userId, notification)
	}
	return nil
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
	comment := vars["c_id"]

	request := &pb.UpvoteRequest{
		UserId: userId,
		ContentContext: &pb.Context_Comment{
			CommentId: comment,
			ThreadCtx: &pb.Context_Thread{
				ThreadId: thread,
				SectionCtx: &pb.Context_Section{
					SectionName: section,
				},
			},
		},
	}

	r.handleUpvote(w, r, request)
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
	comment := vars["c_id"]
	subcomment := vars["sc_id"]

	request := &pb.Request{
		UserId: userId,
		ContentContext: &pb.Context_Subcomment{
			SubcommentId: subcomment,
			CommentCtx: &pb.Context_Comment{
				CommentId: comment,
				ThreadCtx: &pb.Context_Thread{
					ThreadId: thread,
					SectionCtx: &pb.Context_Section{
						SectionName: section,
					},
				},
			},
		},
	}

	r.handleUpvote(w, r, request)
}