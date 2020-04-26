package router

import(
	"net/http"
	"context"
	"log"
	"strings"
	"errors"
	"strconv"
	"encoding/json"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/gorilla/mux"
	pb "github.com/luisguve/cheropatilla/internal/protogen/cheropatillapb"
)

// Read Notifications "/readnotifs" handler. It moves the unread notifications of 
// the current user to the read notifications and returns OK on success or an error 
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

// Clear Notifications "/clearnotifs" handler. It deletes both read and unread 
// notifications of the current user and returns OK on success or an error in 
// case of the following:
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

// Follow User "/follow?username={username}" handler. It updates the current user
// to follow the user with the given username and returns OK on success or an
// error in case of the following:
// - username not found ---> 404 NOT_FOUND
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleFollow(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username = vars["username"]
	request := &pb.FollowUserRequest{
		UserId:       userId,
		UserToFollow: username,
	}
	_, err := r.crudClient.FollowUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				http.NotFound(w, req)
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

// Unfollow User "/unfollow?username={username}" handler. It updates the current user
// to unfollow the user with the given username and returns OK on success or an
// error in case of the following:
// - username not found ---> 404 NOT_FOUND
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleUnfollow(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username = vars["username"]
	request := &pb.UnfollowUserRequest{
		UserId:         userId,
		UserToUnfollow: username,
	}
	_, err := r.crudClient.UnfollowUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				http.NotFound(w, req)
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
// - context other than "followers" or "following" ---> INVALID CONTEXT
// - negative or non-number offset query parameter ---> INVALID_OFFSET
// - offset is out of range; there are no more users -> OFFSET_OOR
// - network or encoding failures --------------------> INTERNAL_FAILURE
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
	users, err := r.crudClient.ViewUsers(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.OutOfRange:
				http.Error(w, "OFFSET_OOR", http.StatusBadRequest)
				return
			case codes.NotFound:
				// user not found
				http.NotFound(w, req)
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

// View My Profile "/myprofile" handler. It returns a page containing the current
// user's personal information. It may return an error in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
// - template rendering ---> TEMPLATE_ERROR
func (r *Router) handleMyProfile(userId string, w http.ResponseWriter, 
	req *http.Request) {
	request := &pb.GetBasicUserDataRequest{
		UserId: userId,
	}
	userData, err := r.crudClient.GetBasicUserData(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
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
	if err = r.templates.ExecuteTemplate(w, "myprofile.html", userData); err != nil {
		log.Printf("Could not execute template myprofile.html: %v", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Update My Profile "/myprofile/update" handler. It updates all of the fields
// which may be updated in the /myprofile page. It returns OK on success or an error
// in case of the following:
// username already in use -> USERNAME_UNAVAILABLE
// not valid username ------> INVALID_USERNAME
// network failures --------> INTERNAL_FAILURE
func (r *Router) handleUpdateMyProfile(userId string, w http.ResponseWriter, 
	req *http.Request) {
	alias := req.FormValue("alias")
	username := req.FormValue("username")
	description := req.FormValue("description")
	newPicUrl, err, status := getAndSaveFile(req, "pic_url")
	if err != nil {
		// It's ok to get an errMissingFile, but if it's not such an error, it is
		// an internal failure.
		if !errors.Is(err, errMissingFile) {
			http.Error(w, err.Error(), status)
			return
		}
	}
	request := &pb.UpdateBasicUserDataRequest{
		UserId:      userId,
		Alias:       alias,
		Username:    username,
		Description: description,
		PicUrl:      newPicUrl,
	}
	_, err := r.crudClient.UpdateBasicUserData(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code(){
			case codes.AlreadyExists:
				http.Error(w, "USERNAME_UNAVAILABLE", http.StatusOK)
				return
			case codes.InvalidArgument:
				http.Error(w, "INVALID_USERNAME", http.StatusBadRequest)
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

// View User Profile "/profile/{username}" handler. It returns a page containing
// a user's basic data, followers, following and threads created. It may return an
// error in case of the following:
// - username not found -> 404 NOT_FOUND
// - network failure ----> INTERNAL_FAILURE
// - template rendering -> TEMPLATE_ERROR
func (r *Router) handleViewUserProfile(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["username"]
	request := &pb.ViewUserByUsernameRequest{
		Username: username,
	}
	userData, err := r.crudClient.ViewUserByUsername(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code(){
			case codes.NotFound:
				http.NotFound(w, req)
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

	var threadsCreated *pb.GetThreadsResponse
	// Load threads created only if this user has created some threads.
	if len(userData.ThreadsCreated) > 0 {
		threadsRequest := &pb.GetThreadsRequest{Threads: userData.ThreadsCreated}
		threadsCreated, err = r.crudClient.GetThreads(context.Background(), 
		threadsRequest)
		if err != nil {
			log.Printf("Could not get threads created: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}

	followers := len(userData.FollowersIds)
	following := len(userData.FollowingIds)

	data := &templates.ProfileView{
		BasicData:      templates.UserInfo{
			Alias:           userData.Alias,
			Username:        userData.Username,
			PicUrl:          userData.PicUrl,
			About:           userData.About,
			LastTimeCreated: userData.LastTimeCreated,
		},
		ThreadsCreated: threadsCreated.Threads,
		Following:      following,
		Followers:      followers,
	}

	if err = r.templates.ExecuteTemplate(w, "viewuserprofile.html", data); err != nil {
		log.Printf("Could not execute template viewuserprofile.html: %v", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Login "/login" handler. It returns OK on successful login or an error in case of the 
// following:
// - invalid username or password -> 401 UNAUTHORIZED
// - network failure --------------> INTERNAL_FAILURE
// - unable to set cookie ---------> COOKIE_ERROR
func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	username := req.FormValue("username")
	password := req.FormValue("password")
	request := &pb.LoginRequest{
		Username: username,
		Password: password,
	}
	res, err := r.crudClient.Login(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.PermissionDenied:
				http.Error(w, "INVALID_CREDENTIALS", http.StatusUnauthorized)
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
	// Set session cookie
	session, _ := r.store.Get(req, "session")
	session.Values["user_id"] = res.UserId
	if err := session.Save(req, w); err != nil {
		log.Printf("Could not save session because... %v\n", err)
		http.Error(w, "COOKIE_ERROR", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Sign in "/signin" handler. It returns OK on successful sign in or an error in case
// of the following:
// - email already in use ----> EMAIL_ALREADY_EXISTS
// - username already in use -> USERNAME_ALREADY_EXISTS
// - username not valid ------> INVALID_USERNAME
// - network failure ---------> INTERNAL_FAILURE
// - unable to set cookie ----> COOKIE_ERROR
func (r *Router) handleSignin(w http.ResponseWriter, req *http.Request) {
	email := req.FormValue("email")
	name := req.FormValue("name")
	alias := req.FormValue("alias")
	about := req.FormValue("about")
	username := req.FormValue("username")
	password := req.FormValue("password")
	picUrl, err, status := getAndSaveFile(req, "pic_url")
	if err != nil {
		// It's ok to get an errMissingFile, but if it's not such an error, it is
		// an internal failure.
		if !errors.Is(err, errMissingFile) {
			http.Error(w, err.Error(), status)
			return
		}
		picUrl = "/tmp/default.jpg"
	}
	userData := &pb.BasicUserData{
		Email:    email,
		Name:     name,
		PicUrl:   picUrl,
		Username: username,
		Alias:    alias,
		About:    about,
	}
	res, err := r.crudClient.RegisterUser(userData)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.AlreadyExists:
				// could be email or username already in use
				log.Printf(resErr.Message())
				http.Error(w, resErr.Message(), http.StatusConflict)
				return
			case codes.InvalidArgument:
				http.Error(w, "INVALID_USERNAME", http.StatusBadRequest)
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
	// Set session cookie
	session, _ := r.store.Get(req, "session")
	session.Values["user_id"] = res.UserId
	if err = session.Save(req, w); err != nil {
		log.Printf("Could not save session because... %v\n", err)
		http.Error(w, "COOKIE_ERROR", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write("OK")
}

// Logout "/logout" handler. It removes the session. It doesn't use the userId 
// provided by r.onlyUsers middleware. It returns OK on success or an error in 
// case of the following:
// - unable to set cookie -> COOKIE_ERROR
func (r *Router) handleLogout(_ string, w http.ResponseWriter, req *http.Request) {
	session, _ := r.store.Get(req, "session")
	session.Options = &sessions.Options{
		// MaxAge < 0 means delete cookie immediately
		MaxAge: -1,
	}
	if err := session.Save(req, w); err != nil {
		log.Printf("Could not save session because... %v\n", err)
		http.Error(w, "COOKIE_ERROR", htp.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
