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
)

// Subcomments "/{section}/{thread}/comment/?c_id={c_id}&offset={offset}" handler.
// It returns (JSON) subcomments under a given comment with id equal to c_id. Offset
// query parameter indicates how many subcomments to skip, since these data is stored
// and returned in sequential order. It may return an error in case of the following: 
// - negative or non-number offset query parameter ------------> INVALID_OFFSET
// - offset is out of range; there are not that much comments -> OFFSET_OOR
// - network or encoding failures -----------------------------> INTERNAL_FAILURE
func (r *Router) handleGetSubcomments(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	offset, err := strconv.Atoi(vars["offset"], 10)
	if err != nil || offset < 0 {
		log.Printf("offset (%v) is not valid\n", offset)
		http.Error(w, "INVALID_OFFSET", http.StatusBadRequest)
		return
	}
	commentId := vars["c_id"]
	thread := vars["thread"]
	section := vard["section"]
	subcommentsReq := &pb.GetSubcmmentsRequest{
		Offset: uint32(offset),
		CommentCtx: &pb.Context.Comment{
			CommentId: commentId, 
			ThreadCtx: &pb.Context.Thread{
				ThreadId: thread,
				SectionCtx: &pb.Context.Section{
					SectionName: section,
				},
			},
		},
	}
	// Send request
	stream, err := r.crudClient.GetComments(context.Background(), subcommentsReq)
	if err != nil {
		log.Printf("Failed to communicate to server: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
		return
	}

	var feed templates.FeedSubcomments

	// Continuously receive responses
	for {
		subcomment, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// check for server failures or bad request
			if resErr, ok := status.FromError(err); ok {
				switch resErr.Code() {
				case codes.NotFound:
					log.Println("Could not find comment, thread or section")
					http.NotFound(w, r)
					return
				case codes.OutOfRange:
					log.Printf("Offset is out of range: %v\n", resErr.Message())
					http.Error(w, "OFFSET_OOR", http.StatusBadRequest)
					return
				case codes.Internal:
					log.Printf("Internal server failure: %v", resErr.Message())
				default:
					log.Printf("Unknown error code %v: %v\n", resErr.Code(), 
					resErr.Message())
				}
			} else {
				errMsg := fmt.Sprintf("Error receiving response from stream: %v\n", err)
				log.Printf("%v", errMsg)
				feed.ErrorMsg = errMsg
			}
			w.WriteHeader(http.StatusInternalServerError)
			break
		}
		feed.Subcomments = append(feed.Subcomments, subcomment)
	}
	// Encode and send response
	if err = json.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("Could not encode feed: %v\n", err)
		http.Error(w, "INTERNAL_FAILURE", http.StatusInternalServerError)
	}
}