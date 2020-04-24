package router

import(
	"log"
	"net/http"
	"context"
	"encoding/json"

	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/gorilla/mux"
	pb "github.com/luisguve/cheropatilla/internal/pkg/cheropatillapb"
	"github.com/luisguve/cheropatilla/internal/pkg/templates"
	"github.com/luisguve/cheropatilla/internal/pkg/pagination"
)

// Thread "/{section}/{thread}" handler. It looks for a thread using its identifier 
// under the given section name, and displays a layout showing buttons for 
// viewing profile, creating a thread and submitting a comment on the current thread.
// That's the only difference between the logged in user and the non-logged in user
// views. It may return an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - template rendering failures -------> TEMPLATE_ERROR
// - netwotk failures ------------------> INTERNAL_FAILURE
func (r *Router) handleViewThread(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]

	// Load thread
	content, err := r.crudClient.GetThread(context.Background(),
		&pb.GetThreadRequest{ 
			Thread: &pb.FullUserData.ThreadInfo{Id: thread, Section: section},
		})
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// Section name or thread id are probably wrong. 
				// Send 404 NOT_FOUND and return.
				log.Printf("Could not find thread (id: %s) on section %s\n",
			 	thread, section)
				http.NotFound(w, r)
				return
			 case codes.Unavailable:
			 	// Section unavailable
			 	log.Printf("Section %s unavailable\n", section)
			 	http.Error(w, "SECTION_UNAVAILABLE", http.StatusNoContent)
			 	return
			 default:
			 	log.Printf("Unknown error code %v: %v\n", resErr.Code(), 
			 		resErr.Message())
			 	http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			 	return
			}
		}
		log.Printf("Error while getting thread: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	// Request to load comments
	contentPattern := &pb.ContentPattern{
		Pattern:        templates.FeedPattern,
		// Do not discard any comment
		DiscardIds:     []string{},
		ContentContext: &pb.Context.Thread{
			SectionCtx: &pb.Context.Section{SectionName: section},
			ThreadId:   thread,
		},
	}
	feed := templates.FeedContent{}
	// Load comments only if there are comments on this thread
	if content.Thread.ExtraData.ThreadRelated.Replies > 0 {
		feed, err = r.recycleContent(contentPattern)
		if err != nil {
			log.Printf("An error occurred while getting comments: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	if len(feed.ContentIds) > 0 {
		// Update session
		updateDiscardIdsSession(req, w, feed.ContentIds, 
			func(discard *pagination.DiscardIds, ids []string) {
				discard.ThreadComments[thread] = feed.ContentIds
			})
	}
	data := &templates.ThreadView{Content: content, Feed: feed}
	userId := currentUser(req)
	if userId != "" {
		// User has logged in. Get user info.
		userData, err := r.crudClient.GetFullUserData(context.Background(), 
		&pb.GetFullUserDataRequest{UserId: userId})
		if err != nil {
			log.Println("Could not get user data")
			if resErr, ok := status.FromError(err); ok {
				if resErr.Code() == codes.Unauthenticated {
					log.Printf("User %s is unregistered\n", userId)
				}
			}
		} else {
			data.Username = userData.BasicUserData.Username
		}
	}

	if err := r.templates.ExecuteTemplate(w, "thread.html", data); err != nil {
		log.Printf("Could not execute template thread.html because... %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle thread comments "/{section}/{thread}/recycle" handler.
// It returns a new feed for the thread in JSON format. It may return an error
// in the case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - no more comments are available ----> OUT_OF_RANGE
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network or encoding failures ------> INTERNAL_FAILURE
func (r *Router) handleRecycleComments (w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]

	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	// Get id of contents to be discarded
	discard := getDiscardIds(session)
	contentPattern := &pb.ContentPattern{
		Pattern:        templates.FeedPattern,
		// Discard previous comments
		DiscardIds:     discard.ThreadComments[thread],
		ContentContext: &pb.Context.Thread{
			ThreadId:   thread,
			SectionCtx: &pb.Context.Section{
				SectionName: section,
			},
		},
	}
	feed, err := r.recycleContent(contentPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				log.Printf("Invalid section id %s or thread id %s\n", section, thread)
				http.NotFound(w, r)
				return
			case codes.OutOfRange:
				log.Println("OOR: no more comments on this thread are available")
				http.Error(w, "OUT_OF_RANGE", http.StatusNoContent)
				return
			case codes.Internal:
				log.Printf("Internal error: %s\n", resErr.Message())
				// only return an error if there were no comments found.
				if len(feed.ContentIds) == 0 {
					http.Error(w, "SECTION_UNAVAILABLE", http.StatusNoContent)
					return
				}
				// if it could fetch some comments, return these.
				log.Println("Could not get all the comments requested.")
				w.WriteHeader(http.StatusPartialContent)
			default:
				log.Printf("Unknown code %v: %v\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Could not recycle comments: %v\n", err)
			if len(feed.ContentIds) == 0 {
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		}
	}
	// Update session
	r.updateDiscardIdsSession(req, w, feed.ContentIds, 
		func(discard *pagination.DiscardIds, ids []string){
		discard.ThreadComments[thread] = append(discard.ThreadComments[thread], ids...)
	})
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Post Upvote "/{section}/{thread}/upvote/" handler. It returns OK on success or an
// error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleUpvoteThread(userId string, w http.ResponseWriter, 
	r *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]

	request := &pb.UpvoteRequest{
		UserId: userId,
		ContentContext: &pb.Context.Thread{
			ThreadId: thread,
			SectionCtx: &pb.Context.Section{
				SectionName: section,
			},
		},
	}

	err = r.postUpvote(request)

	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// section or thread not found
				http.NotFound(w, r)
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
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Post Upvote "/{section}/{thread}/upvote/?c_id={c_id}" handler. It returns OK on 
// success or an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleUpvoteComment(userId string, w http.ResponseWriter, 
	r *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]
	comment := vars["c_id"]

	request := &pb.UpvoteRequest{
		UserId: userId,
		ContentContext: &pb.Context.Comment{
			CommentId: comment,
			ThreadCtx: &pb.Context.Thread{
				ThreadId: thread,
				SectionCtx: &pb.Context.Section{
					SectionName: section,
				},
			},
		},
	}
	err = r.postUpvote(request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// section or thread not found
				http.NotFound(w, r)
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
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (r *Router) postUpvote(postUpvoteRequest *pb.UpvoteRequest) error {
	stream, err := r.crudClient.Upvote(context.Background(), request)
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
