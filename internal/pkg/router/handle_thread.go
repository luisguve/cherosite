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

	request := &pb.GetThreadRequest{ 
		Thread: &pb.Context_Thread{
			Id:         thread,
			SectionCtx: &pb.Context_Section{
				Name: section,
			},
		},
	}
	// Load thread
	content, err := r.crudClient.GetThread(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// Section name or thread id are probably wrong. 
				// Log for debugging.
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
		log.Printf("Could not send request: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}
	var feed templates.ContentsFeed
	// Load comments only if there are comments on this thread
	if content.Metadata.Replies > 0 {
		// Request to load comments
		contentPattern := &pb.ContentPattern{
			Pattern:        templates.FeedPattern,
			ContentContext: &pb.Context_Thread{
				Id:         thread,
				SectionCtx: &pb.Context_Section{
					Name: section,
				},
			},
			// ignore DiscardIds; do not discard any comment
		}
		stream, err = r.crudClient.RecycleContent(context.Background(), 
		contentPattern)
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
	}
	// Update session only if there are comments.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pComments := feed.GetPaginationComments()

			d.ThreadComments[thread] = pComments
		})
	}

	// get current user data for header section
	userId := currentUser(req)
	var userHeader *pb.UserHeaderData
	if userId != "" {
		// A user is logged in. Get its data.
		userHeader = r.getUserHeaderData(w, userId)
	}

	threadView := templates.DataToThreadView(content, feed, userHeader, userId)

	if err := r.templates.ExecuteTemplate(w, "thread.html", threadView); err != nil {
		log.Printf("Could not execute template thread.html: %v\n", err)
		http.Error(w, "TEMPLATE_ERROR", http.StatusInternalServerError)
	}
}

// Recycle thread comments "/{section}/{thread}/recycle" handler.
// It returns a new feed of comments for the thread in JSON format.
// It may return an error in the following cases:
// - invalid section name or thread id -> 404 NOT_FOUND
// - no more comments are available ----> OUT_OF_RANGE
// - network or encoding failures ------> INTERNAL_FAILURE
func (r *Router) handleRecycleComments(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]

	// Get always returns a session, even if empty
	session, _ := r.store.Get(req, "session")
	discardIds := getDiscardIds(session)

	contentPattern := &pb.ContentPattern{
		DiscardIds:     discardIds.FormatThreadComments(thread),
		Pattern:        templates.FeedPattern,
		ContentContext: &pb.Context_Thread{
			ThreadId:   thread,
			SectionCtx: &pb.Context_Section{
				SectionName: section,
			},
		},
	}
	var feed templates.ContentsFeed

	stream, err := r.crudClient.RecycleContent(context.Background(), contentPattern)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Invalid section id %s or thread id %s\n", section, thread)
				http.NotFound(w, r)
				return
			case codes.OutOfRange:
				log.Println("OOR: no more comments on this thread are available")
				http.Error(w, "OUT_OF_RANGE", http.StatusNoContent)
				return
			default:
				log.Printf("Unknown code %v: %v\n", resErr.Code(), resErr.Message())
				http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Could not send request: %v\n", err)
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
		}
	} else {
		feed, err := getFeed(stream)
		if err != nil {
			log.Printf("An error occurred while getting feed: %v\n", err)
			w.WriteHeader(http.StatusPartialContent)
		}
	}
	// update session only if there is content.
	if len(feed.Contents) > 0 {
		r.updateDiscardIdsSession(req, w, func(d *pagination.DiscardIds) {
			pComments := feed.GetPaginationComments()

			d.ThreadComments[thread] = append(d.ThreadComments[thread], pComments...)
		})
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}

// Save thread "/{section}/{thread}/save" handler. It adds the thread id
// to the list of saved threads of the given user, whose id is provided.
// It returns OK on success or an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleSave(userId string, w http.ResponseWriter,
	req *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]

	request := &pb.SaveThreadRequest{
		UserId: userId,
		Thread: &pb.Context_Thread{
			Id: thread,
			SectionCtx: &pb.Context_Section{
				Name: section,
			},
		},
	}
	_, err := r.crudClient.SaveThread(context.Background(), request)
	if err != nil {
		if resErr, ok := status.FromError(err); ok {
			switch resErr.Code() {
			case codes.NotFound:
				// log for debugging
				log.Printf("Invalid section id %s or thread id %s\n", section, thread)
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
		} else {
			log.Printf("Could not send request: %v\n", err)
			http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	return
}

// Post Upvote "/{section}/{thread}/upvote/" handler. It leverages the operation of 
// submitting the upvote to the method handleUpvote, which returns OK on success or 
// an error in case of the following:
// - invalid section name or thread id -> 404 NOT_FOUND
// - section or thread are unavailable -> SECTION_UNAVAILABLE
// - network failures ------------------> INTERNAL_FAILURE
func (r *Router) handleUpvoteThread(userId string, w http.ResponseWriter, 
	req *http.Request) {
	vars := mux.Vars(req)
	section := vars["section"]
	thread := vars["thread"]

	request := &pb.UpvoteRequest{
		UserId: userId,
		ContentContext: &pb.Context_Thread{
			ThreadId: thread,
			SectionCtx: &pb.Context_Section{
				SectionName: section,
			},
		},
	}

	r.handleUpvote(w, req, request)
}
