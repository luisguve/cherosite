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

// Dashboard "/" handler. It displays the dashboard of the logged in user that 
// consists of the user notifications, the threads saved, threads created, the 
// number of followers and following. Below that, a list of the active threads 
// created by users that this user is following. 
// It may return an error in case of the following:
// - user is unregistered -> USER_UNREGISTERED
// - network failures -----> INTERNAL_FAILURE
// - template rendering ---> TEMPLATE_ERROR
func (r *Router) handleRoot(userId string, w http.ResponseWriter, req *http.Request) {
	request := &pb.GetDashboardDataRequest{
		UserId: userId,
	}
	dData, err = r.crudClient.GetDashboardData(context.Background(), request)
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

	followers := len(dData.FollowersIds)
	following := len(dData.FollowingIds)

	var feed templates.ContentsFeed
	// Get dashboard feed only if this user is following other users
	if following > 0 {
		activityPattern := &pb.ActivityPattern{
			Pattern: templates.FeedPattern,
			Context: &pb.ActivityPattern_Users{
				Users: &pb.UserList{
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
	} else {
		// FOR DEBUGGING
		log.Printf("This user isn't following anybody\n")
	}

	// Get user activity
	var userActivity templates.ContentsFeed
	activityPattern := &pb.ActivityPattern{
		Pattern: templates.CompactPattern,
		Context: &pb.ActivityPattern_UserId{
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

	var savedThreads templates.ContentsFeed
	// Load threads saved only if this user has saved some threads.
	if dData.ThreadsSaved > 0 {
		savedPattern := &pb.ContentPattern{
			Pattern:        templates.CompactPattern,
			ContentContext: &pb.ContentPattern_UserId{
				UserId: dData.UserId,
			},
			// ignore DiscardIds; do not discard any thread
		}
		stream, err = r.crudClient.RecycleContent(context.Background(), savedPattern)
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
	}
	// update session only if there is content.
	switch {
	case len(feed.Contents) > 0:
		r.updateDiscardIdsSession(req, w, feed, 
			func(d *pagination.DiscardIds, cf templates.ContentsFeed) {
			pActivity := cf.GetPaginationActivity()
			for userId, content := range pActivity {
				d.FeedActivity[userId].ThreadsCreated = content.ThreadsCreated
				d.FeedActivity[userId].Comments = content.Comments
				d.FeedActivity[userId].Subcomments = content.Subcomments
			}
		})
	case len(userActivity.Contents) > 0:
		r.updateDiscardIdsSession(req, w, userActivity, 
			func(d *pagination.DiscardIds, cf templates.ContentsFeed) {
			pActivity := cf.GetPaginationActivity()
			id := dData.UserId
			d.UserActivity[id].ThreadsCreated = pActivity[id].ThreadsCreated
			d.UserActivity[id].Comments = pActivity[id].Comments
			d.UserActivity[id].Subcomments = pActivity[id].Subcomments
		})
	case len(savedThreads.Contents) > 0:
		r.updateDiscardIdsSession(req, w, savedThreads, 
			func(d *pagination.DiscardIds, cf templates.ContentsFeed) {
			pThreads := savedThreads.GetPaginationThreads()
			for section, threadIds := range pThreads {
				d.ThreadsSaved[section] = threadIds
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

// Recycle Feed "/recyclefeed" handler. It returns a new feed for the user in JSON format.
// The user must be logged in and follow other users, whose recent activity will compose 
// up the returned feed. It may return an error in case of the following:
// - user is unregistered --------------> USER_UNREGISTERED
// - user is not following other users -> NO_USERS_FOLLOWING
// - network or encoding failures ------> INTERNAL_FAILURE
func (r *Router) handleRecycleFeed(userId string, w http.ResponseWriter, 
	req *http.Request) {
	request := &pb.GetBasicUserDataRequest{
		UserId: userId,
	}
	following, err := r.crudClient.GetUserFollowingIds(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(); ok {
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

	activityPattern := &pb.ActivityPattern{
		Pattern: templates.FeedPattern,
		Context: &pb.ActivityPattern_Users{
			Users: &pb.UserList{
				Ids: following.Ids,
			},
		},
		DiscardIds: discard.FormatFeedActivity(following.Ids)
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
	// Update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, feed, 
		func(d *pagination.DiscardIds, cf templates.ContentsFeed) {
			pActivity := cf.GetPaginationActivity()
			for userId, content := range pActivity {
				tc := d.FeedActivity[userId].ThreadsCreated
				tc = append(tc, content.ThreadsCreated...)

				c := d.FeedActivity[userId].Comments
				c = append(c, content.Comments...)

				sc := d.FeedActivity[userId].Subcomments
				sc = append(sc, content.Subcomments...)
			}
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed.Contents); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// "/recycleactivity"
func (r *Router) handleRecycleMyActivity(userId string, w http.ResponseWriter, 
	req *http.Request) {

}

// "/recyclesaved"
func (r *Router) handleRecycleMySaved(userId string, w http.ResponseWriter, 
	req *http.Request) {

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
