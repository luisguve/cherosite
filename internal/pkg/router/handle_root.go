package router

import(
	"log"
	"net/http"
	"context"
	"encoding/json"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
	"github.com/luisguve/cheropatilla/internal/pkg/templates"
	"github.com/luisguve/cheropatilla/internal/pkg/pagination"
)

// Root "/" handler. It displays the dashboard of the logged in user that consists 
// of the user notifications, the threads saved, threads created, the number of 
// followers and following. Below that, a list of the active threads created by 
// users that this user is following. 
// It may return an error in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
// - template rendering ---> TEMPLATE_ERROR
func (r *Router) handleRoot(userId string, w http.ResponseWriter, req *http.Request) {
	userData, err := r.crudClient.GetFullUserData(context.Background(), 
	&pb.GetFullUserDataRequest{UserId: userId})
	if err != nil {
		if resErr, ok := status.FromError(); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			case codes.Internal:
				log.Printf("Internal error: %v\n", resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			default:
				log.Printf("Unknown error code %v: %v\n", resErr.Code(), 
				resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not get full user data: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}

	// userData holds only the notifications.
	// The rest of the data (threads saved, threads created, users following 
	// and followers) comes in the form of IDs. We should load these sets of 
	// pieces individually.

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
		
	var threadsSaved *pb.GetThreadsResponse
	// Load threads saved only if this user has saved some threads.
	if len(userData.ThreadsSaved) > 0 {
		threadsRequest = &pb.GetThreadsRequest{Threads: userData.ThreadsSaved}
		threadsSaved, err = r.crudClient.GetThreads(context.Background(), threadsRequest)
		if err != nil {
			log.Printf("Could not get threads saved: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}

	followers := len(userData.FollowersIds)
	following := len(userData.FollowingIds)

	feed := templates.FeedContent{}
	data := &templates.FeedView{
		FullUserData:   userData,
		ThreadsCreated: threadsCreated,
		ThreadsSaved:   threadsSaved,
		Following:      following,
		Followers:      followers,
	}
	// Get dashboard feed only if this user is following other users
	if following > 0 {
		contentPattern := &pb.ContentPattern{
			Pattern:        templates.FeedPattern,
			// Do not discard any thread
			DiscardIds:     []string{},
			ContentContext: &pb.Context_Feed{
				UserIds: userData.FollowingIds,
			},
		}
		data.Feed, err = r.recycleContent(contentPattern)
		if err != nil {
			log.Printf("An error occurred while getting feed: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	// /*FOR DEBUGGING
	if len(data.Feed.ContentIds) == 0 {
		if following == 0 {
			log.Printf("This user isn't following anybody\n")
		} else {
			log.Printf("Could not get any threads created by %v\n", 
			userData.FollowingIds)
		}
	}// FOR DEBUGGING*/

	// update session only if there is feed.
	if len(data.Feed.ContentIds) > 0 {
		// make a map holding all of the content ids on its only key "user_feed"
		feedIds := map[string][]string{
			"user_feed": data.Feed.ContentIds,
		}
		// Update session
		r.updateDiscardIdsSession(req, w, feedIds, 
			func(discard *pagination.DiscardIds, feedIds map[string][]string){
			discard.FeedThreads = feedIds["user_feed"]
		})
	}
	// render dashboard
	if err = r.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("Could not execute template dashboard.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle Feed "/recycle" handler. It returns a new feed for the user in JSON format.
// The user must be logged, follow other users and these other users must have posted
// a thread recently. It may return an error in case of the following:
// - user is unregistered --------------> USER_UNREGISTERED
// - user is not following other users -> NO_USERS_FOLLOWING
// - there is no more feed available ---> NO_NEW_FEED
// - network or encoding failures ------> INTERNAL_FAILURE
func (r *Router) handleRecycleFeed(userId string, w http.ResponseWriter, 
	req *http.Request) {
	userData, err := r.crudClient.GetFullUserData(context.Background(), 
	&pb.GetFullUserDataRequest{UserId: userId})
	if err != nil {
		if resErr, ok := status.FromError(); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			case codes.Internal:
				log.Printf("Internal error: %v\n", resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			default:
				log.Printf("Unknown error code %v: %v\n", resErr.Code(), 
				resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not get full user data: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	following := len(userData.FollowingIds)
	// Recycle feed only if this user is following other users.
	if following == 0 {
		http.Error(w, "NO_USERS_FOLLOWING", http.StatusBadRequest)
		return
	}
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)
	contentPattern := &pb.ContentPattern{
		Pattern:        templates.FeedPattern,
		DiscardIds:     discard.FeedThreads,
		ContentContext: &pb.Context_Feed{
			UserIds: userData.FollowingIds,
		},
	}
	feed, err := r.recycleContent(contentPattern)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
	}
	// Check whether it couldn't find new feed
	if len(feed.ContentIds) == 0 {
		w.Write([]byte("NO_NEW_FEED"))
		return
	}
	// make a map holding all of the content ids on its only key "user_feed"
	feedIds := map[string][]string{
		"user_feed": feed.ContentIds,
	}
	// Update session
	r.updateDiscardIdsSession(req, w, feedIds, 
		func(discard *pagination.DiscardIds, feedIds map[string][]string) {
			discard.FeedThreads = append(discard.FeedThreads, feedIds["user_feed"]...)
		})
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Explore page "/explore" handler. It returns html containing a feed composed of
// random threads from different sections. It may return an error in case of 
// the following:
// - template rendering failure -> TEMPLATE_ERROR
func (r *Router) handleExplore(w http.ResponseWriter, req *http.Request) {
	contentPattern := &pb.GeneralPattern{
		Pattern:    templates.FeedPattern,
		// Do not discard any thread
		DiscardIds: map[string]*pb.GeneralPattern_Ids{},
	}
	feed, err := r.recycleGeneral(contentPattern)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	data := &templates.SectionView{Feed: feed}
	// Get user and set username
	userId := currentUser(req)
	if userId != "" {
		// A user is logged in. Get its data.
		userData, err := r.crudClient.GetFullUserData(context.Background(), 
			&pb.GetFullUserDataRequest{UserId: userId})
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
				log.Printf("Could not get full user data: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			data.Username = userData.BasicUserData.Username
		}
	}
	// Update session only if there is feed
	if len(feed.ContentIds) > 0 {
		r.updateDiscardIdsSession(req, w, feed.ContentIds, 
			func(discard *pagination.DiscardIds, ids map[string][]string) {
			// replace all the ids
			discard.GeneralThreads = ids
		})
	}
	// render explore page
	if err = r.templates.ExecuteTemplate(w, "explore.html", data); err != nil {
		log.Printf("Could not execute template explore.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Explore Recycle "/explore/recycle" handler. It returns a new feed of explore
// in JSON format, excluding threads previously sent. It may return an error in case
// of the following:
// - encoding failure -> INTERNAL_FAILURE
func (r *Router) handleExploreRecycle(w http.ResponseWriter, req *http.Request) {
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	discard := getDiscardIds(session)
	generalIds := make(map[string]*pb.GeneralPattern_Ids)

	// discard.GeneralThreads holds a map[string][]string, but pb.GeneralPattern
	// requires a map[string]*pb.GeneralPattern.
	// pb.GeneralPattern holds the []string in its Ids field.
	for section, ids := range discard.GeneralThreads {
		generalIds[section] = &pb.GeneralPattern_Ids {
			Ids: ids,
		}
	}
	contentPattern := &pb.GeneralPattern{
		Pattern:    templates.FeedPattern,
		// Discard threads previously seen
		DiscardIds: generalIds,
	}
	feed, err := r.recycleGeneral(contentPattern)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
	}
	// update session only if there is new feed.
	if len(feed.ContentIds) > 0 {
		r.updateDiscardIdsSession(req, w, feed.ContentIds, 
			func(discard *pagination.DiscardIds, ids map[string][]string) {
				// append new feed from every section
				for section, threads := range ids {
					discard.GeneralThreads[section] = append(discard.GeneralThreads[section], 
						threads...)
				}
		})
	}		
	// encode and send new feed
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

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
