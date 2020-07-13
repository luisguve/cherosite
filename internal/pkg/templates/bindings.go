package templates

import(
	"sync"
	"strings"
	"fmt"
	"log"
	"time"

	pbApi "github.com/luisguve/cheroproto-go/cheroapi"
	pbDataFormat "github.com/luisguve/cheroproto-go/dataformat"
)

func DataToMyProfileView(userData *pbDataFormat.BasicUserData, uhd *pbApi.UserHeaderData) *MyProfileView {
	// set user header data
	hd := setHeaderData(uhd, nil)
	// set user profile data
	bud := setBasicUserData(userData)
	return &MyProfileView{
		HeaderData:    hd,
		BasicUserData: bud,
	}
}

func DataToProfileView(userData *pbApi.ViewUserResponse, uhd *pbApi.UserHeaderData, 
	activity []*pbApi.ContentRule, currentUserId string) *ProfileView {
	recycleSet := []RecycleType{
		RecycleType{
			Label: fmt.Sprintf("Recycle %s's activity", userData.Alias),
			Link: fmt.Sprintf("/profile/recycle?userid=%s", userData.UserId),
		},
	}
	var (
		hd HeaderData
		pd ProfileData
		activitySet []OverviewRenderer
		wg sync.WaitGroup
	)
	// set user header data
	wg.Add(1)
	go func() {
		defer wg.Done()
		hd = setHeaderData(uhd, recycleSet)
	}()
	// set user profile data
	wg.Add(1)
	go func() {
		defer wg.Done()
		pd = setProfileData(userData)
	}()
	// convert each activity into an OverviewRenderer set
	wg.Add(1)
	go func() {
		defer wg.Done()
		activitySet = contentsToOverviewRendererSet(activity, currentUserId)
	}()
	// check whether the current user is a follower of the user viewing
	var isF bool
	if currentUserId == "" {
		isF = false
	} else {
		isF = strings.Contains(strings.Join(userData.FollowersIds, "|"), currentUserId)
	}
	wg.Wait()
	return &ProfileView{
		HeaderData:  hd,
		ProfileData: pd,
		Activity:    activitySet,
		IsFollower:  isF,
	}
}

func DataToDashboardView(dData *pbApi.DashboardData, feed, activity, 
	saved []*pbApi.ContentRule) *DashboardView {
	recycleSet := []RecycleType{
		RecycleType{
			Label: "Recycle your feed",
			Link: "/recyclefeed",
		},
		RecycleType{
			Label: "Recycle your activity",
			Link: "/recycleactivity",
		},
		RecycleType{
			Label: "Recycle your saved threads",
			Link: "/recyclesaved",
		},
	}
	var (
		hd HeaderData
		activitySet []OverviewRenderer
		savedContentSet []OverviewRenderer
		feedSet []OverviewRenderer
		wg sync.WaitGroup
	)
	// set user header data
	wg.Add(1)
	go func() {
		defer wg.Done()
		hd = setHeaderData(dData.UserHeaderData, recycleSet)
	}()
	// convert user activity set into an OverviewRenderer set
	wg.Add(1)
	go func() {
		defer wg.Done()
		activitySet = contentsToOverviewRendererSet(activity, dData.UserId)
	}()
	// convert saved content set into an OverviewRenderer set
	wg.Add(1)
	go func() {
		defer wg.Done()
		savedContentSet = contentsToOverviewRendererSet(saved, dData.UserId)
	}()
	// convert feed activity into an OverviewRenderer set
	wg.Add(1)
	go func() {
		defer wg.Done()
		feedSet = contentsToOverviewRendererSet(feed, dData.UserId)
	}()
	wg.Wait()
	return &DashboardView{
		HeaderData:   hd,
		Followers:    len(dData.FollowersIds),
		Following:    len(dData.FollowingIds),
		Activity:     activitySet,
		SavedContent: savedContentSet,
		Feed:         feedSet,
	}
}

func DataToExploreView(feed []*pbApi.ContentRule, uhd *pbApi.UserHeaderData,
currentUserId string) *ExploreView {
	recycleSet := []RecycleType{
		RecycleType{
			Label: "Recycle explorer",
			Link:  "/explore/recycle",
		},
	}
	var (
		wg sync.WaitGroup
		hd HeaderData
		feedSet []OverviewRenderer
	)
	// set user header data
	wg.Add(1)
	go func() {
		defer wg.Done()
		hd = setHeaderData(uhd, recycleSet)
	}()
	// convert feed content into an OverviewRenderer set
	wg.Add(1)
	go func() {
		defer wg.Done()
		feedSet = contentsToOverviewRendererSet(feed, currentUserId)
	}()
	wg.Wait()
	return &ExploreView{
		HeaderData: hd,
		Feed:       feedSet,
	}
}

func DataToThreadView(content *pbApi.ContentData, feed []*pbApi.ContentRule, 
uhd *pbApi.UserHeaderData, currentUserId string) *ThreadView{
	metadata := content.Metadata
	section := strings.ToLower(strings.Replace(metadata.Section, " ", "", -1))
	recycleSet := []RecycleType{
		RecycleType{
			Label: "Recycle comments",
			Link:  fmt.Sprintf("/%s/%s/recycle", section, metadata.Id),
		},
	}
	var (
		hd HeaderData
		threadContent ContentRenderer
		threadComments []OverviewRenderer
		wg sync.WaitGroup
	)
	// set user header data
	wg.Add(1)
	go func() {
		defer wg.Done()
		hd = setHeaderData(uhd, recycleSet)
	}()
	// convert *pbApi.ContentRule into a ContentRenderer
	wg.Add(1)
	go func() {
		defer wg.Done()
		threadContent = contentToContentRenderer(content, currentUserId)
	}()
	// convert comments feed into a []OverviewRenderer
	wg.Add(1)
	go func() {
		defer wg.Done()
		threadComments = commentsToOverviewRendererSet(feed, currentUserId)
	}()
	wg.Wait()
	return &ThreadView{
		HeaderData: hd,
		Content:    threadContent,
		Comments:   threadComments,
	}
}

func DataToSectionView(feed []*pbApi.ContentRule, uhd *pbApi.UserHeaderData,
currentUserId string) *SectionView {
	var section string
	// get section name from first valid content rule.
	for _, pbRule := range feed {
		if (pbRule.Data != nil) && (pbRule.Data.Metadata != nil) {
			section = pbRule.Data.Metadata.Section
			break
		}
	}
	if section == "" {
		log.Println("Could not get section name")
	}
	sectionId := strings.Replace(strings.ToLower(section), " ", "", -1)

	recycleSet := []RecycleType{
		RecycleType{
			Label: "Recycle threads",
			Link:  fmt.Sprintf("/%s/recycle", sectionId),
		},
	}
	// set user header data
	hd := setHeaderData(uhd, recycleSet)
	// convert feed into a []OverviewRenderer
	sectionThreads := contentsToOverviewRendererSet(feed, currentUserId)

	return &SectionView{
		HeaderData:  hd,
		Feed:        sectionThreads,
		SectionName: section,
	}
}

func setHeaderData(uhd *pbApi.UserHeaderData, recycleSet []RecycleType) HeaderData {
	hd := HeaderData{RecycleTypes: recycleSet}
	if uhd == nil {
		return hd
	}
	// set read notifs
	for _, pbNotif := range uhd.ReadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp.Seconds,
		}
		hd.User.ReadNotifs = append(hd.User.ReadNotifs, notif)
	}
	// set unread notifs
	for _, pbNotif := range uhd.UnreadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp.Seconds,
		}
		hd.User.UnreadNotifs = append(hd.User.UnreadNotifs, notif)
	}
	hd.User.Alias = uhd.Alias
	return hd
}

func setProfileData(userData *pbApi.ViewUserResponse) ProfileData {
	var pd ProfileData
	if userData != nil {
		pd = ProfileData{
			BasicUserData: BasicUserData{
				Patillavatar: userData.PicUrl,
				Alias:        userData.Alias,
				Username:     userData.Username,
				Description:  userData.About,
			},
			Followers:    len(userData.FollowersIds),
			Following:    len(userData.FollowingIds),
		}
	}
	return pd
}

func setBasicUserData(userData *pbDataFormat.BasicUserData) BasicUserData {
	var bud BasicUserData
	if userData != nil {
		bud = BasicUserData{
			Patillavatar: userData.PicUrl,
			Alias:        userData.Alias,
			Username:     userData.Username,
			Description:  userData.About,
		}
	}
	return bud
}

func contentToContentRenderer(pbData *pbApi.ContentData, userId string)	ContentRenderer {
	if pbData == nil {
		log.Println("pbData has no data")
		return &NoContent{}
	}
	pbRule := &pbApi.ContentRule{
		Data: pbData,
	}
	bc := setBasicContent(pbRule, userId)

	metadata := pbData.Metadata

	threadId := metadata.Id
	sectionId := strings.Replace(strings.ToLower(metadata.Section), " ", "", -1)

	threadLink := fmt.Sprintf("/%s/%s", sectionId, threadId)
	saveLink := fmt.Sprintf("%s/save", threadLink)
	undoSaveLink := fmt.Sprintf("%s/undosave", threadLink)
	replyLink := fmt.Sprintf("%s/comment", threadLink)
	
	var saved bool
	if userId == "" {
		saved = false
	} else {
		saved = strings.Contains(strings.Join(metadata.UsersWhoSaved, "|"), userId)
	}

	return &Thread{
		BasicContent: bc,
		Replies:      metadata.Replies,
		SaveLink:     saveLink,
		UndoSaveLink: undoSaveLink,
		Saved:        saved,
		ReplyLink:    replyLink,
	}
}

// formatCommentContent converts a *pbApi.ContentRule into a *CommentContent and
// returns it along with an error indicating whether or not the content context was
// not a *pbApi.ContentRule_CommentCtx. userId is used to setBasicContent.
func formatCommentContent(pbRule *pbApi.ContentRule, userId string) (OverviewRenderer, error) {
	if pbRule.Data == nil {
		return nil, fmt.Errorf("Comment has no data")
	}
	ctx, ok := pbRule.ContentContext.(*pbApi.ContentRule_CommentCtx)
	if !ok {
		return nil, fmt.Errorf("Failed type assertion to *pbApi.ContentRule_CommentCtx")
	}
	bc := setBasicContent(pbRule, userId)
	metadata := pbRule.Data.Metadata

	threadId := metadata.Id
	sectionId := strings.Replace(strings.ToLower(metadata.Section), " ", "", -1)
	threadLink := fmt.Sprintf("/%s/%s", sectionId, threadId)

	// comment context
	comCtx := ctx.CommentCtx

	replyLink := fmt.Sprintf("%s/comment/?c_id=%s", threadLink, comCtx.Id)
	bc.UpvoteLink = fmt.Sprintf("%s/upvote?c_id=%s", threadLink, comCtx.Id)
	bc.UndoUpvoteLink = fmt.Sprintf("%s/undoupvote?c_id=%s", threadLink, comCtx.Id)

	comContent := &CommentContent{
		BasicContent: bc,
		Id:           comCtx.Id,
		Replies:      metadata.Replies,
		ReplyLink:    replyLink,
	}
	return comContent, nil
}

func commentsToOverviewRendererSet(pbRuleSet []*pbApi.ContentRule, userId string) []OverviewRenderer {
	var (
		ovwRendererSet = make([]OverviewRenderer, len(pbRuleSet))
		wg sync.WaitGroup
	)

	for idx, pbRule := range pbRuleSet {
		wg.Add(1)
		go func(idx int, pbRule *pbApi.ContentRule) {
			defer wg.Done()
			ovwRenderer, err := formatCommentContent(pbRule, userId)
			if err != nil {
				log.Println(err)
				ovwRenderer = &NoContent{}
			}
			ovwRendererSet[idx] = ovwRenderer
		}(idx, pbRule)
	}
	wg.Wait()
	return ovwRendererSet
}

func contentToOverviewRenderer(pbRule *pbApi.ContentRule, userId string) OverviewRenderer {
	if pbRule.Data == nil {
		log.Println("pbRule has no content")
		return &NoContent{}
	}

	var ovwRenderer OverviewRenderer

	bc := setBasicContent(pbRule, userId)
	metadata := pbRule.Data.Metadata

	threadId := metadata.Id
	sectionId := strings.Replace(strings.ToLower(metadata.Section), " ", "", -1)
	threadLink := fmt.Sprintf("/%s/%s", sectionId, threadId)

	switch ctx := pbRule.ContentContext.(type) {
	// it's a THREAD
	case *pbApi.ContentRule_ThreadCtx:
		saveLink := fmt.Sprintf("%s/save", threadLink)
		undoSaveLink := fmt.Sprintf("%s/undosave", threadLink)
		replyLink := fmt.Sprintf("%s/comment", threadLink)
		bc.UpvoteLink = fmt.Sprintf("%s/upvote", threadLink)
		bc.UndoUpvoteLink = fmt.Sprintf("%s/undoupvote", threadLink)
		var saved bool
		if userId == "" {
			saved = false
		} else {
			saved = strings.Contains(strings.Join(metadata.UsersWhoSaved, "|"), userId)
		}

		ovwRenderer = &Thread{
			BasicContent: bc,
			Replies:      metadata.Replies,
			SaveLink:     saveLink,
			UndoSaveLink: undoSaveLink,
			Saved:        saved,
			ReplyLink:    replyLink,
		}
	// it's a COMMENT
	case *pbApi.ContentRule_CommentCtx:
		// comment context
		comCtx := ctx.CommentCtx

		bc.UpvoteLink = fmt.Sprintf("%s/upvote?c_id=%s", threadLink, comCtx.Id)
		bc.UndoUpvoteLink = fmt.Sprintf("%s/undoupvote?c_id=%s", threadLink, comCtx.Id)
		ovwRenderer = &CommentView{
			BasicContent: bc,
			Id:           comCtx.Id,
			Replies:      metadata.Replies,
		}
	// it's a SUBCOMMENT
	case *pbApi.ContentRule_SubcommentCtx:
		// subcomment context
		subcCtx := ctx.SubcommentCtx

		bc.UpvoteLink = fmt.Sprintf("%s/upvote?c_id=%s&sc_id=%s", threadLink, 
			subcCtx.CommentCtx.Id, subcCtx.Id)
		bc.UndoUpvoteLink = fmt.Sprintf("%s/undoupvote?c_id=%s&sc_id=%s", threadLink, 
			subcCtx.CommentCtx.Id, subcCtx.Id)
		ovwRenderer = &SubcommentView{
			BasicContent: bc,
			CommentId:    subcCtx.CommentCtx.Id,
			Id:           subcCtx.Id,
		}
	}
	return ovwRenderer
}

// contentsToOverviewRendererSet converts a slice of *pbApi.ContentRule into a slice of
// OverviewRenderer. userId is used to check whether the user has saved the content
func contentsToOverviewRendererSet(pbRuleSet []*pbApi.ContentRule, userId string) []OverviewRenderer {
	var (
		ovwRendererSet = make([]OverviewRenderer, len(pbRuleSet))
		wg sync.WaitGroup
	)

	for idx, pbRule := range pbRuleSet {
		wg.Add(1)
		go func(idx int, pbRule *pbApi.ContentRule) {
			defer wg.Done()
			ovwRenderer := contentToOverviewRenderer(pbRule, userId)
			ovwRendererSet[idx] = ovwRenderer
		}(idx, pbRule)
	}
	wg.Wait()
	return ovwRendererSet
}

// setBasicContent returns a *BasicContent object filled with data retrieved from a
// *pbApi.ContentRule. userId is used to check whether the user has upvoted the content.
func setBasicContent(pbRule *pbApi.ContentRule, userId string) *BasicContent {
	if pbRule.Data == nil {
		log.Println("pbRule has no data")
		return &BasicContent{}
	}
	author := pbRule.Data.Author
	content := pbRule.Data.Content
	metadata := pbRule.Data.Metadata

	sectionLowercased := strings.ToLower(metadata.Section)
	sectionLink := strings.Replace(fmt.Sprintf("/%s", sectionLowercased), " ", "-", -1)
	
	threadLink := fmt.Sprintf("%s/%s", sectionLink, metadata.Id)

	var summary string
	if len(content.Content) > 75 {
		summary = content.Content[:75]
	} else {
		summary = content.Content
	}
	var upvoted bool
	if userId == "" {
		upvoted = false
	} else {
		upvoted = strings.Contains(strings.Join(metadata.VoterIds, "|"), userId)
	}

	return &BasicContent{
		Title:       content.Title,
		Status:      pbRule.Status,
		Thumbnail:   content.FtFile,
		Permalink:   metadata.Permalink,
		Content:     content.Content,
		Summary:     summary,
		Upvotes:     metadata.Upvotes,
		Upvoted:     upvoted,
		SectionName: metadata.Section,
		Author:      author.Alias,
		Username:    author.Username,
		PublishDate: time.Unix(content.PublishDate.Seconds, 0).Format(time.RFC822),
		ThreadLink:  threadLink,
		SectionLink: sectionLink,
	}
}
