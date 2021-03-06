package router

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"

	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbUsers "github.com/luisguve/cheroproto-go/userapi"
	"github.com/luisguve/cherosite/internal/pkg/pagination"
	"github.com/luisguve/cherosite/internal/pkg/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Dashboard "/" handler. It displays the dashboard of the logged in user that
// consists of the activity of users following, saved threads, user activity,
// notifications and the number of followers and following.
// It may return an error in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
// - template rendering ---> TEMPLATE_ERROR
func (r *Router) handleRoot(userId string, w http.ResponseWriter, req *http.Request) {
	request := &pbUsers.GetDashboardDataRequest{
		UserId: userId,
	}
	dData, err := r.usersClient.GetDashboardData(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("User %s unregistered. Deleting session... ", userId)
				if err = r.deleteSession(req, w); err != nil {
					log.Printf("Could not save session because: %v\n", err)
				} else {
					log.Println("Done.")
				}
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
				Users:   dData.FollowingIds,
				// ignore DiscardIds; do not discard any activity
			}

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
		}()
	}

	// Get user activity
	var userActivity templates.ContentsFeed
	wg.Add(1)
	go func() {
		defer wg.Done()
		activityPattern := &pbApi.ActivityPattern{
			Pattern: templates.CompactPattern,
			Users:   []string{dData.UserId},
			// ignore DiscardIds; do not discard any activity
		}
		stream, err := r.generalClient.RecycleActivity(context.Background(), activityPattern)
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
			stream, err := r.generalClient.RecycleSaved(context.Background(), savedPattern)
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
	if len(feed.Contents) > 0 {
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
	}
	if len(userActivity.Contents) > 0 {
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
	}
	if len(savedThreads.Contents) > 0 {
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
// in HTML format. The user must be logged in and follow other users, whose recent
// activity will compose up the returned feed. It may return an error in case of
// the following:
// - user is unregistered --------------> USER_UNREGISTERED
// - user is not following other users -> NO_USERS_FOLLOWING
// - network or encoding failures ------> INTERNAL_FAILURE
// Note: NO_USERS_FOLLOWING is returned along with a 200 status code.
func (r *Router) handleRecycleFeed(userId string, w http.ResponseWriter,
	req *http.Request) {
	request := &pbUsers.GetBasicUserDataRequest{
		UserId: userId,
	}
	following, err := r.usersClient.GetUserFollowingIds(context.Background(), request)
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
		w.Write([]byte("NO_USERS_FOLLOWING"))
		return
	}
	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)

	activityPattern := &pbApi.ActivityPattern{
		Pattern:    templates.FeedPattern,
		Users:      following.Ids,
		DiscardIds: discard.FormatFeedActivity(following.Ids),
	}

	stream, err := r.generalClient.RecycleActivity(context.Background(), activityPattern)
	if err != nil {
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}

	feed, err := getFeed(stream)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
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
	res := templates.FeedToBytes(feed.Contents, userId, true)
	contentLength := strconv.Itoa(len(res))
	w.Header().Set("Content-Length", contentLength)
	w.Header().Set("Content-Type", "text/html")
	if _, err = w.Write(res); err != nil {
		log.Println("Recycle activity: could not send response:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Recycle activity "/recycleactivity" handler. It returns a new feed of user
// activity in HTML format. The user must be logged in, and its recent activity
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

	discardActivity := discard.FormatUserActivity("dashboard-" + userId)
	if len(discardActivity) > 0 {
		// Strip prefix "dashboard-" from key.
		discardActivity[userId] = discardActivity["dashboard-" + userId]
		delete(discardActivity, "dashboard-" + userId)
	}

	var userActivity templates.ContentsFeed

	activityPattern := &pbApi.ActivityPattern{
		DiscardIds: discardActivity,
		Pattern:    templates.CompactPattern,
		Users:      []string{userId},
	}

	stream, err := r.generalClient.RecycleActivity(context.Background(), activityPattern)
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
	}

	userActivity, err = getFeed(stream)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
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
	res := templates.FeedToBytes(userActivity.Contents, userId, true)
	contentLength := strconv.Itoa(len(res))
	w.Header().Set("Content-Length", contentLength)
	w.Header().Set("Content-Type", "text/html")
	if _, err = w.Write(res); err != nil {
		log.Println("Recycle my activity: could not send response:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Recycle saved "/recyclesaved" handler. It returns a new feed of user saved
// content in HTML format. The user must be logged in, and its saved content
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

	stream, err := r.generalClient.RecycleSaved(context.Background(), savedPattern)
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
	}

	savedThreads, err = getFeed(stream)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.InvalidArgument:
				w.Write([]byte(""))
				return
			}
		}
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
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
	res := templates.FeedToBytes(savedThreads.Contents, userId, true)
	contentLength := strconv.Itoa(len(res))
	w.Header().Set("Content-Length", contentLength)
	w.Header().Set("Content-Type", "text/html")
	if _, err = w.Write(res); err != nil {
		log.Println("Recycle saved: could not send response:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Explore page "/explore" handler. It displays a page containing a feed made up
// of random threads from different sections. It may return an error in case of
// the following:
// - template rendering failure --------> TEMPLATE_ERROR
// - encoding failure or network error -> INTERNAL_FAILURE
func (r *Router) handleExplore(w http.ResponseWriter, req *http.Request) {
	generalPattern := &pbApi.GeneralPattern{
		Pattern: templates.FeedPattern,
		// ignore DiscardIds; do not discard any thread
	}

	stream, err := r.generalClient.RecycleGeneral(context.Background(), generalPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			log.Printf("Unknown error code %v: %v\n", resErr.Code(), resErr.Message())
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
		}
		log.Println("Could not send request:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	feed, err := getFeed(stream)
	if err != nil {
		log.Printf("An error occurred while getting feed: %v\n", err)
		w.WriteHeader(http.StatusPartialContent)
	}
	// get current user data for header section
	userId := r.currentUser(req)
	var userHeader *pbUsers.UserHeaderData
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
// in HTML format, excluding threads already seen. It may return an error in case
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

	stream, err := r.generalClient.RecycleGeneral(context.Background(), generalPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			log.Printf("Unknown error code %v: %v\n", resErr.Code(), resErr.Message())
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
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
	// Get current user id.
	userId := r.currentUser(req)
	res := templates.FeedToBytes(feed.Contents, userId, true)
	contentLength := strconv.Itoa(len(res))
	w.Header().Set("Content-Length", contentLength)
	w.Header().Set("Content-Type", "text/html")
	if _, err = w.Write(res); err != nil {
		log.Println("Recycle explore: could not send response:", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}
