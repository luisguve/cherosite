package router

import(
	"crypto/rand"
	"os"
	"path/filepath"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"error"
	"net/http"
	"context"

	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
	"github.com/luisguve/cheropatilla/internal/pkg/livedata"
	"github.com/luisguve/cheropatilla/internal/pkg/templates"
	"github.com/luisguve/cheropatilla/internal/pkg/pagination"
)

const(
	maxUploadSize = 64 << 20 // 64 mb
	uploadPath = "tmp"
)

var(
	errMissingFile      = errors.New("MISSING_ft_file_INPUT")
	errInternalFailure  = errors.New("INTERNAL_FAILURE")
	errFileTooBig       = errors.New("FILE_TOO_BIG")
	errInvalidFile      = errors.New("INVALID_FILE")
	errInvalidFileType  = errors.New("INVALID_FILE_TYPE")
	errCantReadFileType = errors.New("CANT_READ_FILE_TYPE")
	errCantWriteFile    = errors.New("CANT_WRITE_FILE")
)

func (r *Router) recycleContent(contentPattern *pb.ContentPattern) (templates.FeedContent, 
	error) {
	// Send request
	stream, err := r.crudClient.RecycleContent(context.Background(), contentPattern)
	if err != nil {
		log.Printf("Could not send request to RecycleContent: %v\n", err)
		return templates.FeedContent{}, err
	}

	var feed templates.FeedContent
	
	// Continuously receive responses
	for {
		contentRule, err := stream.Recv()
		if err == io.EOF {
			// Reset err value
			err = nil
			break
		}
		if err != nil {
			errMsg := fmt.Sprintf("Error receiving response from stream: %v\n", err)
			log.Printf("%v", errMsg)
			feed.ErrorMsg = errMsg
			break
		}
		feed.ContentPatternResponse = append(feed.ContentPatternResponse, contentRule)
		feed.ContentIds = append(feed.ContentIds, contentRule.Data.Id)
	}
	return feed, err
}

func (r *Router) recycleGeneral(contentPattern *pb.GeneralPattern) (templates.FeedGeneral, 
	error) {
	// Send request
	stream, err := r.crudClient.RecycleGeneral(context.Background(), contentPattern)
	if err != nil {
		log.Printf("Could not send request to RecycleContent: %v\n", err)
		return templates.FeedGeneral{}, err
	}

	var feed templates.FeedGeneral
	
	// Continuously receive responses
	for {
		contentRule, err := stream.Recv()
		if err == io.EOF {
			// Reset err value
			err = nil
			break
		}
		if err != nil {
			errMsg := fmt.Sprintf("Error receiving response from stream: %v\n", err)
			log.Printf("%v", errMsg)
			feed.ErrorMsg = errMsg
			break
		}
		section := feed.ContentPatternResponse.Data.Section
		id := feed.ContentPatternResponse.Data.Id
		feed.ContentPatternResponse = append(feed.ContentPatternResponse, contentRule)
		feed.ContentIds[section] = append(feed.ContentIds[section], id)
	}
	return feed, err
}

// getDiscardIds returns the id of contents to be discarded from loads of new feeds
func getDiscardIds(sess *sessions.Session) (discard *pagination.DiscardIds) {
	discardIds := sess.Values["discard_ids"]
	var ok bool
	if discard, ok = discardIds.(*pagination.DiscardIds); !ok {
		// This session value has not been set before.
		discard = &pagination.DiscardIds{}
	}
	return discard
}

// updateDiscardIdsSession replaces id of contents already set in the session 
// with the provided ids and saves the cookie.
func (r *Router) updateDiscardIdsSession(req *http.Request, w http.ResponseWriter, 
	ids []string, setDiscardIds func(*pagination.DiscardIds, []string)) {
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)
	// Replace content already seen by the user with the new feed
	setDiscardIds(discard, ids)
	session.Values["discard_ids"] = discard
	if err = session.Save(req, w); err != nil {
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
func getAndSaveFile(w http.ResponseWriter, req *http.Request, formName string) (string, 
	error, int) {
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
		return "" errInvalidFileType, http.StatusBadRequest
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

// currentUser returns a string containing the current user id or an empty 
// string if the user is not logged in.
func (r *Router) currentUser(req *http.Request) string {
	session, err := r.store.Get(req, "session")
	if err != nil {
		log.Printf("Could not get session because...%v\n", err)
		return ""
	}
	if userId, ok := session.Values["user_id"].(string); !ok {
		// User not logged in
		return ""
	}
	return userId
}

// onlyUsers middleware displays the login page if the user has not logged in yet,
// otherwise it executes the next handler passing it the current user id, the
// ResponseWriter and the Request.
func (r *Router) onlyUsers(next userContentsHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId := currentUser(r)
		if userId == "" {
			// user has not logged in.
			if err := r.templates.ExecuteTemplate(w, "login.html", nil); err != nil {
				log.Printf("Error: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		next(userId, w, r)
	}
}

type userContentsHandler func(userId string, w http.ResponseWriter, r *http.Request)

// renderError is an helper function to set a given status code header and
// return a given error message to the client.
func renderError(w http.ResponseWriter, message string, statusCode int) {
	r.WriteHeader(statusCode)
	w.Write([]byte(message))
}

// randToken generates a random, unique string with a length equal to len.
func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
