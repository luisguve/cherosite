package router

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbUsers "github.com/luisguve/cheroproto-go/userapi"
	"github.com/luisguve/cherosite/internal/pkg/pagination"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Read Notifications "/readnotifs" handler. It moves the unread notifications of
// the current user to the read notifications and returns OK on success or an error
// in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
func (r *Router) handleReadNotifs(userId string, w http.ResponseWriter,
	req *http.Request) {
	request := &pbUsers.ReadNotifsRequest{
		UserId: userId,
	}
	_, err := r.usersClient.MarkAllAsRead(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Println(resErr.Message())
				http.Error(w, "USER_UNREGISTERED", http.StatusNotFound)
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
	request := &pbUsers.ClearNotifsRequest{
		UserId: userId,
	}
	_, err := r.usersClient.ClearNotifs(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Println(resErr.Message())
				http.Error(w, "USER_UNREGISTERED", http.StatusNotFound)
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
// - username not found ----> 404 NOT_FOUND
// - user following itself -> SELF_FOLLOW
// - user is unregistered --> USER_UNREGISTERED
// - network failures ------> INTERNAL_FAILURE
func (r *Router) handleFollow(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["username"]
	request := &pbUsers.FollowUserRequest{
		UserId:       userId,
		UserToFollow: username,
	}
	_, err := r.usersClient.FollowUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				http.NotFound(w, req)
				return
			case codes.InvalidArgument:
				http.Error(w, "SELF_FOLLOW", http.StatusBadRequest)
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
// - username not found ------> 404 NOT_FOUND
// - user unfollowing itself -> SELF_UNFOLLOW
// - user is unregistered ----> USER_UNREGISTERED
// - network failures --------> INTERNAL_FAILURE
func (r *Router) handleUnfollow(userId string, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["username"]
	request := &pbUsers.UnfollowUserRequest{
		UserId:         userId,
		UserToUnfollow: username,
	}
	_, err := r.usersClient.UnfollowUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				http.NotFound(w, req)
				return
			case codes.InvalidArgument:
				http.Error(w, "SELF_UNFOLLOW", http.StatusBadRequest)
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
// - context other than "followers" or "following" ---> INVALID_CONTEXT
// - negative or non-number offset query parameter ---> INVALID_OFFSET
// - offset is out of range; there are no more users -> OFFSET_OOR
// - network or encoding failures --------------------> INTERNAL_FAILURE
func (r *Router) handleViewUsers(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	ctx := strings.ToLower(vars["context"])
	userId := vars["userid"]

	offset, err := strconv.Atoi(vars["offset"])
	if err != nil || offset < 0 {
		log.Printf("offset (%v) is not valid\n", offset)
		http.Error(w, "INVALID_OFFSET", http.StatusBadRequest)
		return
	}
	// ctx should be either "following" or "followers"
	switch ctx {
	case "followers":
	case "following":
	default:
		http.Error(w, "INVALID_CONTEXT", http.StatusBadRequest)
		return
	}
	request := &pbUsers.ViewUsersRequest{
		UserId:  userId,
		Context: ctx,
		Offset:  uint32(offset),
	}
	users, err := r.usersClient.ViewUsers(context.Background(), request)
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
	userData, s, err := r.getBasicUserData(userId)
	if err != nil {
		http.Error(w, err.Error(), s)
		return
	}
	userHeader := r.getUserHeaderData(w, userId)

	profileView := templates.DataToMyProfileView(userData, userHeader)

	if err := r.templates.ExecuteTemplate(w, "myprofile.html", profileView); err != nil {
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
	newPicUrl, err, s := getAndSaveFile(req, "pic_url")
	if err != nil {
		// It's ok to get an errMissingFile, but if it's not such an error, it is
		// an internal failure.
		if !errors.Is(err, errMissingFile) {
			http.Error(w, err.Error(), s)
			return
		}
	}
	request := &pbUsers.UpdateBasicUserDataRequest{
		UserId:      userId,
		Alias:       alias,
		Username:    username,
		Description: description,
		PicUrl:      newPicUrl,
	}
	_, err = r.usersClient.UpdateBasicUserData(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
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
// a user's basic data, followers, following and its recent activity. It may return
// an error in case of the following:
// - username not found -> 404 NOT_FOUND
// - network failure ----> INTERNAL_FAILURE
// - template rendering -> TEMPLATE_ERROR
func (r *Router) handleViewUserProfile(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["username"]
	request := &pbUsers.ViewUserByUsernameRequest{
		Username: username,
	}
	userData, err := r.usersClient.ViewUserByUsername(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
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

	// get user activity
	activityPattern := &pbApi.ActivityPattern{
		Pattern: templates.CompactPattern,
		Users: []string{userData.UserId},
		// ignore DiscardIds; do not discard any activity
	}
	var feed templates.ContentsFeed

	stream, err := r.generalClient.RecycleActivity(context.Background(), activityPattern)
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

	// Get current user data for header section.
	userId := r.currentUser(req)
	var userHeader *pbUsers.UserHeaderData
	if userId != "" {
		// A user is logged in. Get its data.
		userHeader = r.getUserHeaderData(w, userId)
	}
	// update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pActivity := feed.GetUserPaginationActivity()
			id := userData.UserId
			a := d.UserActivity[id]
			a.ThreadsCreated = pActivity.ThreadsCreated
			a.Comments = pActivity.Comments
			a.Subcomments = pActivity.Subcomments
			d.UserActivity[id] = a
		})
	}
	profileView := templates.DataToProfileView(userData, userHeader, feed.Contents, userId)

	err = r.templates.ExecuteTemplate(w, "viewuserprofile.html", profileView)
	if err != nil {
		log.Printf("Could not execute template viewuserprofile.html: %v", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle user activity "/profile/recycle?userid={userid}" handler. It returns a
// new feed of recent activity for the user in JSON format. It may return an error
// in case of the following:
// - user not found ---------------> 404 NOT_FOUND
// - network or encoding failures -> INTERNAL_FAILURE
func (r *Router) handleRecycleUserActivity(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userId := vars["userid"]

	session, _ := r.store.Get(req, "session")
	discardIds := getDiscardIds(session)

	activityPattern := &pbApi.ActivityPattern{
		DiscardIds: discardIds.FormatUserActivity(userId),
		Pattern:    templates.CompactPattern,
		Users:      []string{userId},
	}
	var feed templates.ContentsFeed

	// get user activity
	stream, err := r.generalClient.RecycleActivity(context.Background(), activityPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
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
	} else {
		feed, err = getFeed(stream)
		if err != nil {
			log.Printf("An error occurred while getting feed: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	// update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pActivity := feed.GetUserPaginationActivity()
			a := d.UserActivity[userId]
			a.ThreadsCreated = append(a.ThreadsCreated, pActivity.ThreadsCreated...)
			a.Comments = append(a.Comments, pActivity.Comments...)
			a.Subcomments = append(a.Subcomments, pActivity.Subcomments...)
			d.UserActivity[userId] = a
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
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
	request := &pbUsers.LoginRequest{
		Username: username,
		Password: password,
	}
	res, err := r.usersClient.Login(context.Background(), request)
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
	picUrl, err, s := getAndSaveFile(req, "pic_url")
	if err != nil {
		// It's ok to get an errMissingFile, but if it's not such an error,
		// it is an internal failure.
		if !errors.Is(err, errMissingFile) {
			http.Error(w, err.Error(), s)
			return
		}
		idx := rand.Intn(len(defaultPics))
		pic := defaultPics[idx]
		picUrl = path.Join("static", "pics", pic)
	}
	if alias == "" {
		alias = name
	}
	request := &pbUsers.RegisterUserRequest{
		Email:    email,
		Name:     name,
		PicUrl:   picUrl,
		Username: username,
		Alias:    alias,
		About:    about,
		Password: password,
	}
	res, err := r.usersClient.RegisterUser(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.AlreadyExists:
				// could be email or username already in use
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
	w.Write([]byte("OK"))
}

// Logout "/logout" handler. It removes the session. It doesn't use the userId
// provided by r.onlyUsers middleware. It returns OK on success or an error in
// case of the following:
// - unable to set cookie -> COOKIE_ERROR
func (r *Router) handleLogout(_ string, w http.ResponseWriter, req *http.Request) {
	if err := r.deleteSession(req, w); err != nil {
		log.Printf("Could not save session because... %v\n", err)
		http.Error(w, "COOKIE_ERROR", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Delete cookie and update session.
func (r *Router) deleteSession(req *http.Request, w http.ResponseWriter) error {
	session, _ := r.store.Get(req, "session")
	session.Options = &sessions.Options{
		// MaxAge < 0 means delete cookie immediately
		MaxAge: -1,
	}
	return session.Save(req, w)
}
