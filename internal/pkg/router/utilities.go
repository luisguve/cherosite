package router

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/sessions"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbContext "github.com/luisguve/cheroproto-go/context"
	pbDataFormat "github.com/luisguve/cheroproto-go/dataformat"
	"github.com/luisguve/cherosite/internal/pkg/pagination"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxUploadSize = 64 << 20 // 64 mb
	uploadPath    = "tmp"
)

var (
	errMissingFile      = errors.New("MISSING_ft_file_INPUT")
	errInternalFailure  = errors.New("INTERNAL_FAILURE")
	errFileTooBig       = errors.New("FILE_TOO_BIG")
	errInvalidFile      = errors.New("INVALID_FILE")
	errInvalidFileType  = errors.New("INVALID_FILE_TYPE")
	errCantReadFileType = errors.New("CANT_READ_FILE_TYPE")
	errCantWriteFile    = errors.New("CANT_WRITE_FILE")
	errUnregistered     = errors.New("USER_UNREGISTERED")
)

// wrapper interface to be used instead of the generated interfaces in pbApi, for streams
// that return a NotifyUser object
type streamNotifs interface {
	Recv() (*pbApi.NotifyUser, error)
}

// wrapper interface to be used instead of the generated interfaces in pbApi, for streams
// that return a ContentRule object
type streamFeed interface {
	Recv() (*pbApi.ContentRule, error)
}

// getFeed continuously receive content rules from the given stream and returns a
// templates.ContentsFeed and any error encountered.
func getFeed(stream streamFeed) (templates.ContentsFeed, error) {
	var feed templates.ContentsFeed
	var err error
	// Continuously receive responses
	for {
		contentRule, err := stream.Recv()
		if err == io.EOF {
			// Reset err value
			err = nil
			break
		}
		if err != nil {
			log.Printf("Error receiving response from stream: %v\n", err)
			break
		}
		feed.Contents = append(feed.Contents, contentRule)
	}
	return feed, err
}

// getDiscardIds returns the id of contents to be discarded from loads of new feeds
func getDiscardIds(sess *sessions.Session) (discard *pagination.DiscardIds) {
	discardIds := sess.Values["discard_ids"]
	var ok bool
	if discard, ok = discardIds.(*pagination.DiscardIds); !ok {
		// This session value has not been set before.
		discard = &pagination.DiscardIds{
			UserActivity:   make(map[string]pagination.Activity),
			FeedActivity:   make(map[string]pagination.Activity),
			SavedThreads:   make(map[string][]string),
			SectionThreads: make(map[string][]string),
			ThreadComments: make(map[string][]string),
			GeneralThreads: make(map[string][]string),
		}
	}
	return discard
}

// updateDiscardIdsSession replaces ids of contents already set in the session
// with the provided templates.ContentsFeed and saves the cookie.
func (r *Router) updateDiscardIdsSession(req *http.Request, w http.ResponseWriter,
	setDiscardIds func(*pagination.DiscardIds)) {
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)
	// Replace content already seen by the user with the new feed
	setDiscardIds(discard)
	session.Values["discard_ids"] = discard
	if err := session.Save(req, w); err != nil {
		log.Printf("Could not save session because... %v\n", err)
	}
}

// getAndSaveFile gets the file identified by formName coming in the request,
// verifies that it does not exceeds the file size limit, and saves it to the
// disk assigning to it a unique, random name.
// On success, it should return the filepath under which it was stored. If there
// are any errors, it will return an empty string, the error message and the
// http status code, which can be StatusBadRequest, StatusInternalServerError or
// StatusOK.
func getAndSaveFile(req *http.Request, formName string) (string, error, int) {
	file, fileHeader, err := req.FormFile(formName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", errMissingFile, http.StatusBadRequest
		}
		log.Printf("Could not read file because... %v\n", err)
		return "", errInternalFailure, http.StatusInternalServerError
	}
	defer file.Close()
	// Get and print out file size
	fileSize := fileHeader.Size
	log.Printf("File size (bytes): %v\n", fileSize)
	// Validate file size
	if fileSize > maxUploadSize {
		return "", errFileTooBig, http.StatusBadRequest
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Could not read all file: %s\n", err)
		return "", errInvalidFile, http.StatusBadRequest
	}

	// Check file type, DetectContentType only needs the first 512 bytes
	detectedFileType := http.DetectContentType(fileBytes)
	switch detectedFileType {
	case "image/jpeg", "image/jpg":
	case "image/gif", "image/png":
	case "application/pdf":
		break
	default:
		return "", errInvalidFileType, http.StatusBadRequest
	}
	fileName := randToken(12)
	fileEndings, err := mime.ExtensionsByType(detectedFileType)
	if err != nil {
		log.Printf("Can't read filetype: %v\n", err)
		return "", errCantReadFileType, http.StatusInternalServerError
	}
	newPath := filepath.Join(uploadPath, fileName+fileEndings[0])

	// Write file to disk
	newFile, err := os.Create(newPath)
	if err != nil {
		log.Printf("Could not create file: %s\n", err)
		return "", errCantWriteFile, http.StatusInternalServerError
	}
	defer newFile.Close() // idempotent, okay to call twice
	if _, err = newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
		return "", errCantWriteFile, http.StatusInternalServerError
	}
	return newPath, nil, http.StatusOK
}

// getUserHeaderData returns username, alias, both read and unread notifs of the given
// user. It sets the corresponding error header given any error while getting user
// header data.
func (r *Router) getUserHeaderData(w http.ResponseWriter, userId string) *pbApi.UserHeaderData {
	userData, err := r.crudClient.GetUserHeaderData(context.Background(),
		&pbApi.GetBasicUserDataRequest{UserId: userId})
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				w.WriteHeader(http.StatusUnauthorized)
			case codes.Internal:
				log.Printf("Internal error: %v\n", resErr.Message())
				w.WriteHeader(http.StatusInternalServerError)
			default:
				log.Printf("Unknown error code %v: %v\n", resErr.Code(),
					resErr.Message())
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			log.Printf("Could not send request: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	return userData
}

// getBasicUserData returns a user's basic data: alias, username, pic_url and
// description, along with a status code and any error encountered.
func (r *Router) getBasicUserData(userId string) (*pbDataFormat.BasicUserData, int, error) {
	request := &pbApi.GetBasicUserDataRequest{
		UserId: userId,
	}
	userData, err := r.crudClient.GetBasicUserData(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %v unregistered\n", userId)
				return nil, http.StatusNotFound, errUnregistered
			default:
				log.Printf("Unknown code %v: %v\n", resErr.Code(), resErr.Message())
				return nil, http.StatusInternalServerError, errInternalFailure
			}
		}
		log.Printf("Could not send request: %v\n", err)
		return nil, http.StatusInternalServerError, errInternalFailure
	}
	return userData, http.StatusOK, nil
}

// handleUpvote is an utility method to help reduce the repetition of similar code in
// other handlers that perform the same operation, in this case, an upvote,  since
// all the handlers that are called in an upvote event share the same upvote request
// object. The duties of returning a response to the client are also delegated to
// postUpvote, which returns OK on success or an error in case of the following:
// - invalid section name, thread id or comment -> 404 NOT_FOUND
// - section, thread or comment are unavailable -> SECTION_UNAVAILABLE
// - network failures ---------------------------> INTERNAL_FAILURE
func (r *Router) handleUpvote(w http.ResponseWriter, req *http.Request,
	upvoteRequest *pbApi.UpvoteRequest) {
	stream, err := r.crudClient.Upvote(context.Background(), upvoteRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// section or thread not found
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
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	// Call broadcastNotifs in a separate goroutine to collect the garbage in this
	// handler
	go r.broadcastNotifs(stream)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleComment is an utility method to help reduce the repetition of similar code in
// other handlers that perform the same operation, in this case, a comment post, since
// all the handlers that are called in a comment event share the same comment request
// object. The duties of returning a response to the client are also delegated to
// postComment, which returns OK on success or an error in case of the following:
// - invalid section name, thread id or comment -> 404 NOT_FOUND
// - network failures ---------------------------> INTERNAL_FAILURE
func (r *Router) handleComment(w http.ResponseWriter, req *http.Request,
	commentRequest *pbApi.CommentRequest) {
	stream, err := r.crudClient.Comment(context.Background(), commentRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// section, thread or comment not found
				log.Printf("Could not find content: %v", resErr.Message())
				http.NotFound(w, req)
				return
			default:
				log.Printf("Unknown code %v: %s\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	// Call broadcastNotifs in a separate goroutine to collect the garbage in this
	// handler
	go r.broadcastNotifs(stream)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (r *Router) broadcastNotifs(stream streamNotifs) {
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
		userId := notifyUser.UserId
		notification := notifyUser.Notification
		// send notification
		go r.hub.Broadcast(userId, notification)
	}
}

// handleDelete is an utility method to help reduce the repetition of similar code in
// other handlers that perform the same operation, in this case, a content deletion, since
// all the handlers that are called in a delete event share the same delete request
// object. The duties of returning a response to the client are also delegated to
// handleDelete, which returns OK on success or an error in case of the following:
// - invalid section name or thread id ---> 404 NOT_FOUND
// - user id and author id are not equal -> UNAUTHORIZED
// - network failures --------------------> INTERNAL_FAILURE
func (r *Router) handleDelete(w http.ResponseWriter, req *http.Request,
	deleteRequest *pbApi.DeleteContentRequest) {
	_, err := r.crudClient.DeleteContent(context.Background(), deleteRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Could not find resource: %v\n", resErr.Message())
				http.NotFound(w, req)
				return
			case codes.Unauthenticated:
				log.Println(resErr.Message())
				http.Error(w, "USER_UNAUTHORIZED", http.StatusUnauthorized)
				return
			default:
				log.Printf("Unknown error code %v: %v", resErr.Code(),
					resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleUndoUpvote is an utility method to help reduce the repetition of similar code in
// other handlers that perform the same operation, in this case, a content upvote undoing,
// since all the handlers that are called in an unupvote event share the same unupvote
// request object. The duties of returning a response to the client are also delegated to
// handleUnupvote, which returns OK on success or an error in case of the following:
// - invalid section name or thread id ------> 404 NOT_FOUND
// - user did not upvote the content before -> NOT_UPVOTED
// - network failures -----------------------> INTERNAL_FAILURE
func (r *Router) handleUndoUpvote(w http.ResponseWriter, req *http.Request,
	undoUpvoteRequest *pbApi.UndoUpvoteRequest) {
	_, err := r.crudClient.UndoUpvote(context.Background(), undoUpvoteRequest)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Could not find resource: %v\n", resErr.Message())
				http.NotFound(w, req)
				return
			case codes.FailedPrecondition:
				log.Println(resErr.Message())
				http.Error(w, "NOT_UPVOTED", http.StatusBadRequest)
				return
			default:
				log.Printf("Unknown error code %v: %v", resErr.Code(),
					resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// currentUser returns a string containing the current user id or an empty
// string if the user is not logged in.
func (r *Router) currentUser(req *http.Request) string {
	session, _ := r.store.Get(req, "session")
	userId, ok := session.Values["user_id"].(string)
	if !ok {
		// User not logged in
		return ""
	}
	return userId
}

// onlyUsers middleware displays the login page if the user has not logged in yet,
// otherwise it executes the next handler passing it the current user id, the
// ResponseWriter and the Request.
func (r *Router) onlyUsers(next func(string, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		userId := r.currentUser(req)
		if userId == "" {
			// user has not logged in.
			if err := r.templates.ExecuteTemplate(w, "login.html", nil); err != nil {
				log.Printf("Could not execute template login.html: %v\n", err)
				http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
			}
			return
		}
		next(userId, w, req)
	}
}

// randToken generates a random, unique string with a length equal to len.
func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// formatContextSection, formatContextThread, formatContextComment and
// formatContextSubcomment are utility functions that return different
// pbContext objects.
func formatContextSection(id string) *pbContext.Section {
	return &pbContext.Section{
		Id: id,
	}
}

func formatContextThread(section, id string) *pbContext.Thread {
	sectionCtx := formatContextSection(section)
	return &pbContext.Thread{
		Id:         id,
		SectionCtx: sectionCtx,
	}
}

func formatContextComment(section, thread, id string) *pbContext.Comment {
	threadCtx := formatContextThread(section, thread)
	return &pbContext.Comment{
		Id:        id,
		ThreadCtx: threadCtx,
	}
}

func formatContextSubcomment(section, thread, comment, id string) *pbContext.Subcomment {
	commentCtx := formatContextComment(section, thread, comment)
	return &pbContext.Subcomment{
		Id:         id,
		CommentCtx: commentCtx,
	}
}
