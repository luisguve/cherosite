package router

import(
	"net/http"
	"context"
	"log"
	"strings"
	"strconv"
	"encoding/json"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/gorilla/mux"
	pb "github.com/luisguve/cheropatilla/internal/protogen/cheropatillapb"
)

// Read Notifications "/readnotifs" handler. It returns OK on success or an error
// in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleReadNotifs(userId string, w http.ResponseWriter, 
	req *http.Request) {
	request := &pb.ReadNotifsRequest {
		UserId: userId,
	}
	_, err := r.crudClient.MarkAllAsRead(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.Unauthenticated:
				log.Println(resErr.Message())
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			default:
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

// Clear Notifications "/clearnotifs" handler. It returns OK on success or an error
// in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleClearNotifs(userId string, w http.ResponseWriter, 
	req *http.Request) {
	request := &pb.ClearNotifsRequest {
		UserId: userId,
	}
	_, err := r.crudClient.ClearNotifs(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.Unauthenticated:
				log.Println(resErr.Message())
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			default:
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

// Follow User "/follow?username={username}" handler. It returns OK on success or an
// error in case of the following:
// - username not found ---> 404 NOT_FOUND
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleFollow(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username = vars["username"]
	request := &pb.FollowUserRequest{
		UserId: userId,
	}
	_, err := r.crudClient.FollowUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				http.NotFound(w, r)
				return
			case codes.Unauthenticated:
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
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
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Unfollow User "/unfollow?username={username}" handler. It returns OK on success or 
// an error in case of the following:
// - username not found ---> 404 NOT_FOUND
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleUnfollow(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username = vars["username"]
	request := &pb.UnfollowUserRequest{
		UserId: userId,
	}
	_, err := r.crudClient.UnfollowUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				http.NotFound(w, r)
				return
			case codes.Unauthenticated:
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
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
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// View Users "/viewusers" handler. It returns a list of user data containing basic
// info in JSON format. It may return an error in case of the following:
// - context other than "followers" or "following" ----> INVALID CONTEXT
// - negative or non-number offset query parameter ----> INVALID_OFFSET
// - offset is out of range; there are not more users -> OFFSET_OOR
// - network or encoding failures ---------------------> INTERNAL_FAILURE
func (r *Router) handleViewUsers(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	context := strings.ToLower(vars["context"])
	userid := vars["userid"]

	offset, err := strconv.Atoi(vars["offset"])
	if err != nil || offset < 0 {
		log.Printf("offset (%v) is not valid\n", offset)
		http.Error(w, "INVALID_OFFSET", http.StatusBadRequest)
		return
	}
	// context should be either "following" or "followers"
	switch context {
	case "followers":
	case "following":
	default:
		http.Error(w, "INVALID_CONTEXT", http.StatusBadRequest)
		return
	}
	request := &pb.ViewUsersRequest {
		UserId:  userId,
		Context: context,
		Offset:  offset,
	}
	users, err := r.crudClient(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.OutOfRange:
				http.Error(w, "OFFSET_OOR", http.StatusBadRequest)
				return
			case codes.NotFound:
				// user not found
				http.NotFound(w, r)
				return
			default:
				log.Printf("Unknown code %v: %v", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Printf("Could not encode users: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}
