package router

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	"github.com/luisguve/cherosite/internal/pkg/pagination"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Dashboard "/" handler. It displays the dashboard of the logged in user that
// consists of the user notifications, the threads saved, threads created, the
// number of followers and following. Below that, a list of the active threads
// created by users that this user is following.
// It may return an error in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
// - template rendering ---> TEMPLATE_ERROR
func (r *Router) handleRoot(userId string, w http.ResponseWriter, req *http.Request) {
	request := &pbApi.GetDashboardDataRequest{
		UserId: userId,
	}
	dData, err := r.crudClient.GetDashboardData(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
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

	var wg sync.WaitGroup
	// Get dashboard feed only if this user is following other users
	var feed templates.ContentsFeed
	following := len(dData.FollowingIds)
	if following > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			activityPattern := &pbApi.ActivityPattern{
				Pattern: templates.FeedPattern,
				Context: &pbApi.ActivityPattern_Users{
					Users: &pbApi.UserList{
						Ids: dData.FollowingIds,
					},
				},
				// ignore DiscardIds; do not discard any activity
			}

			stream, err := r.crudClient.RecycleActivity(context.Background(), activityPattern)
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
			// FOR DEBUGGING
			if len(feed.Contents) == 0 {
				log.Printf("Could not get any threads created by %v\n", dData.FollowingIds)
			}
		}()
	} else {
		// FOR DEBUGGING
		log.Println("This user isn't following anybody")
	}

	// Get user activity
	var userActivity templates.ContentsFeed
	wg.Add(1)
	go func() {
		defer wg.Done()
		activityPattern := &pbApi.ActivityPattern{
			Pattern: templates.CompactPattern,
			Context: &pbApi.ActivityPattern_UserId{
				UserId: dData.UserId,
			},
			// ignore DiscardIds; do not discard any activity
		}
		stream, err := r.crudClient.RecycleActivity(context.Background(), activityPattern)
		if err != nil {
			log.Printf("Could not send request: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		} else {
			userActivity, err = getFeed(stream)
			if err != nil {
				log.Printf("An error occurred while getting feed: %v\n", err)
				w.WriteHeader(http.StatusPartialContent)
			}
		}
	}()

	var savedThreads templates.ContentsFeed
	// Load saved threads only if this user has saved some threads.
	if dData.SavedThreads > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			savedPattern := &pbApi.SavedPattern{
				Pattern: templates.CompactPattern,
				UserId:  dData.UserId,
				// ignore DiscardIds; do not discard any thread
			}
			stream, err := r.crudClient.RecycleSaved(context.Background(), savedPattern)
			if err != nil {
				log.Printf("Could not send request: %v\n", err)
				w.WriteHeader(http.StatusPartialContent)
			} else {
				savedThreads, err = getFeed(stream)
				if err != nil {
					log.Printf("An error occurred while getting feed: %v\n", err)
					w.WriteHeader(http.StatusPartialContent)
				}
			}
		}()
	}
	wg.Wait()
	// update session only if there is content.
	switch {
	case len(feed.Contents) > 0:
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pActivity := feed.GetPaginationActivity()

			for userId, content := range pActivity {
				a := d.FeedActivity[userId]
				a.ThreadsCreated = content.ThreadsCreated
				a.Comments = content.Comments
				a.Subcomments = content.Subcomments
				d.FeedActivity[userId] = a
			}
		})
	case len(userActivity.Contents) > 0:
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pActivity := userActivity.GetUserPaginationActivity()

			// avoid conflict with profile view by adding a preffix dashboard-
			id := "dashboard-" + dData.UserId
			a := d.UserActivity[id]
			a.ThreadsCreated = pActivity.ThreadsCreated
			a.Comments = pActivity.Comments
			a.Subcomments = pActivity.Subcomments
			d.UserActivity[id] = a
		})
	case len(savedThreads.Contents) > 0:
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pThreads := savedThreads.GetPaginationThreads()

			for section, threadIds := range pThreads {
				d.SavedThreads[section] = threadIds
			}
		})
	}
	dashboardView := templates.DataToDashboardView(dData, feed.Contents,
		userActivity.Contents, savedThreads.Contents)

	err = r.templates.ExecuteTemplate(w, "dashboard.html", dashboardView)
	if err != nil {
		log.Printf("Could not execute template dashboard.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle Feed "/recyclefeed" handler. It returns a new activity feed of several users
// in JSON format. The user must be logged in and follow other users, whose recent
// activity will compose up the returned feed. It may return an error in case of
// the following:
// - user is unregistered --------------> USER_UNREGISTERED
// - user is not following other users -> NO_USERS_FOLLOWING
// - network or encoding failures ------> INTERNAL_FAILURE
func (r *Router) handleRecycleFeed(userId string, w http.ResponseWriter,
	req *http.Request) {
	request := &pbApi.GetBasicUserDataRequest{
		UserId: userId,
	}
	following, err := r.crudClient.GetUserFollowingIds(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			default:
				log.Printf("Unknown error code %v: %v\n", resErr.Code(),
					resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request to get following ids: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	// Recycle feed only if this user is following other users.
	if len(following.Ids) == 0 {
		http.Error(w, "NO_USERS_FOLLOWING", http.StatusBadRequest)
		return
	}
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)

	var feed templates.ContentsFeed

	activityPattern := &pbApi.ActivityPattern{
		Pattern: templates.FeedPattern,
		Context: &pbApi.ActivityPattern_Users{
			Users: &pbApi.UserList{
				Ids: following.Ids,
			},
		},
		DiscardIds: discard.FormatFeedActivity(following.Ids),
	}

	stream, err := r.crudClient.RecycleActivity(context.Background(), activityPattern)
	if err != nil {
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
	// FOR DEBUGGING
	if len(feed.Contents) == 0 {
		log.Printf("Could not get any threads created by %v\n", following.Ids)
	}
	// Update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pActivity := feed.GetPaginationActivity()

			for userId, content := range pActivity {
				a := d.FeedActivity[userId]
				a.ThreadsCreated = append(a.ThreadsCreated, content.ThreadsCreated...)
				a.Comments = append(a.Comments, content.Comments...)
				a.Subcomments = append(a.Subcomments, content.Subcomments...)
				d.FeedActivity[userId] = a
			}
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed.Contents); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Recycle activity "/recycleactivity" handler. It returns a new feed of user
// activity in JSON format. The user must be logged in, and its recent activity
// will compose up the returned feed. It may return an error in case of the
// following:
// - user is unregistered ---------> USER_UNREGISTERED
// - network or encoding failures -> INTERNAL_FAILURE
func (r *Router) handleRecycleMyActivity(userId string, w http.ResponseWriter,
	req *http.Request) {
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)

	var userActivity templates.ContentsFeed

	activityPattern := &pbApi.ActivityPattern{
		Pattern: templates.CompactPattern,
		Context: &pbApi.ActivityPattern_UserId{
			UserId: userId,
		},
		DiscardIds: discard.FormatUserActivity("dashboard-" + userId),
	}

	stream, err := r.crudClient.RecycleActivity(context.Background(), activityPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
				return
			default:
				log.Printf("Unknown error code %v: %v\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	} else {
		userActivity, err = getFeed(stream)
		if err != nil {
			log.Printf("An error occurred while getting feed: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	// FOR DEBUGGING
	if len(userActivity.Contents) == 0 {
		log.Printf("Could not get any activity of %v\n", userId)
	}
	// Update session only if there is content.
	if len(userActivity.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pActivity := userActivity.GetUserPaginationActivity()

			// avoid conflict with view profile by adding a preffix dashboard-
			id := "dashboard-" + userId

			a := d.UserActivity[id]
			a.ThreadsCreated = append(a.ThreadsCreated, pActivity.ThreadsCreated...)
			a.Comments = append(a.Comments, pActivity.Comments...)
			a.Subcomments = append(a.Subcomments, pActivity.Subcomments...)
			d.UserActivity[id] = a
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(userActivity.Contents); err != nil {
		log.Printf("Could not encode user activity: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Recycle saved "/recyclesaved" handler. It returns a new feed of user saved
// content in JSON format. The user must be logged in, and its saved content
// will compose up the returned feed. It may return an error in case of the
// following:
// - user is unregistered ---------> USER_UNREGISTERED
// - network or encoding failures -> INTERNAL_FAILURE
func (r *Router) handleRecycleMySaved(userId string, w http.ResponseWriter,
	req *http.Request) {
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)

	var savedThreads templates.ContentsFeed

	savedPattern := &pbApi.SavedPattern{
		Pattern:    templates.CompactPattern,
		UserId:     userId,
		DiscardIds: discard.FormatSavedThreads(),
	}

	stream, err := r.crudClient.RecycleSaved(context.Background(), savedPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered\n", userId)
				http.Error(w, "USER_UNREGISTERED", http.StatusUnauthorized)
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
	} else {
		savedThreads, err = getFeed(stream)
		if err != nil {
			log.Printf("An error occurred while getting feed: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	// FOR DEBUGGING
	if len(savedThreads.Contents) == 0 {
		log.Printf("Could not get any activity of %v\n", userId)
	}
	// Update session only if there is content.
	if len(savedThreads.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pThreads := savedThreads.GetPaginationThreads()
			for section, threadIds := range pThreads {
				d.SavedThreads[section] = append(d.SavedThreads[section], threadIds...)
			}
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(savedThreads.Contents); err != nil {
		log.Printf("Could not encode saved threads: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Explore page "/explore" handler. It returns html containing a feed composed of
// random threads from different sections. It may return an error in case of
// the following:
// - template rendering failure -> TEMPLATE_ERROR
func (r *Router) handleExplore(w http.ResponseWriter, req *http.Request) {
	generalPattern := &pbApi.GeneralPattern{
		Pattern: templates.FeedPattern,
		// ignore DiscardIds; do not discard any thread
	}
	var feed templates.ContentsFeed
	stream, err := r.crudClient.RecycleGeneral(context.Background(), generalPattern)
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
	// get current user data for header section
	userId := r.currentUser(req)
	var userHeader *pbApi.UserHeaderData
	if userId != "" {
		// A user is logged in. Get its data.
		userHeader = r.getUserHeaderData(w, userId)
	}

	exploreView := templates.DataToExploreView(feed.Contents, userHeader, userId)

	// Update session only if there is feed
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pThreads := feed.GetPaginationThreads()
			for section, threadIds := range pThreads {
				d.GeneralThreads[section] = threadIds
			}
		})
	}
	// render explore page
	if err = r.templates.ExecuteTemplate(w, "explore.html", exploreView); err != nil {
		log.Printf("Could not execute template explore.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Explore Recycle "/explore/recycle" handler. It returns a new feed of explore
// in JSON format, excluding threads previously sent. It may return an error in case
// of the following:
// - encoding failure or network error -> INTERNAL_FAILURE
func (r *Router) handleExploreRecycle(w http.ResponseWriter, req *http.Request) {
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	discard := getDiscardIds(session)

	generalPattern := &pbApi.GeneralPattern{
		Pattern:    templates.FeedPattern,
		DiscardIds: discard.FormatGeneralThreads(),
	}

	var feed templates.ContentsFeed
	stream, err := r.crudClient.RecycleGeneral(context.Background(), generalPattern)
	if err != nil {
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
	// update session only if there is new feed.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pThreads := feed.GetPaginationThreads()
			for section, threadIds := range pThreads {
				d.GeneralThreads[section] = append(d.GeneralThreads[section],
					threadIds...)
			}
		})
	}
	// encode and send new feed
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}
