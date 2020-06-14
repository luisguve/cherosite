package templates

import(
	"strings"
	"fmt"

	pbAbi "github.com/luisguve/cheroproto-go/cheroapi"
	pbDataFormat "github.com/luisguve/cheroproto-go/dataformat"
)

func DataToMyProfileView(userData *pbDataFormat.BasicUserData, uhd *pbApi.UserHeaderData)
	*MyProfileView {
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
		}
	}
	// set user header data
	hd := setHeaderData(uhd, recycleSet)
	// set user profile data
	pd := setProfileData(userData)
	// convert each activity into an OverviewRenderer set
	activitySet := contentsToOverviewRendererSet(activity, currentUserId)
	// check whether the current user is a follower of the user viewing
	var isF bool
	if currentUserId == "" {
		isF = false
	} else {
		isF = strings.Contains(strings.Join(userData.FollowersIds, "|"), currentUserId)
	}

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
	// set user header data
	hd := setHeaderData(dData.UserHeaderData, recycleSet)
	// convert user activity set into an OverviewRenderer set
	activitySet := contentsToOverviewRendererSet(activity, dData.UserId)
	// convert saved content set into an OverviewRenderer set
	savedContentSet := contentsToOverviewRendererSet(saved, dData.UserId)
	// convert feed activity into an OverviewRenderer set
	feedSet := contentsToOverviewRendererSet(feed, dData.UserId)
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
	// set user header data
	hd := setHeaderData(uhd, recycleSet)
	// convert feed content into an OverviewRenderer set
	feedSet := contentsToOverviewRendererSet(feed, currentUserId)
	return &ExploreView{
		HeaderData: hd,
		Feed:       feedSet,
	}
}

func DataToThreadView(content *pbApi.ContentRule, feed []*pbApi.ContentRule, 
uhd *pbApi.UserHeaderData, currentUserId string) *ThreadView{
	metadata := content.Metadata
	section := strings.ToLower(strings.Replace(metadata.Section, " ", "", -1))
	recycleSet := []RecycleType{
		RecycleType{
			Label: "Recycle comments",
			Link:  fmt.Sprintf("/%s/%s/recycle", section, metadata.Id),
		},
	}
	// set user header data
	hd := setHeaderData(uhd, recycleSet)
	threadContent := contentToContentRenderer(content, currentUserId)
	threadComments := contentsToOverviewRendererSet(feed, currentUserId)

	return &ThreadView{
		HeaderData: hd,
		Content:    threadContent,
		Comments:   threadComments,
	}
}

func DataToSectionView(feed []*pbApi.ContentRule, uhd *pbApi.UserHeaderData,
currentUserId string) *SectionView {
	var section string
	// just making sure the program doesn't crash in case of a nil feed
	if len(feed) > 0 {
		// get section name from 1st thread in feed
		section = feed[0].Metadata.Section
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
			Date:      pbNotif.Timestamp,
		}
		hd.ReadNotifs = append(hd.ReadNotifs, notif)
	}
	// set unread notifs
	for _, pbNotif := range uhd.UnreadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp,
		}
		hd.UnreadNotifs = append(hd.UnreadNotifs, notif)
	}
	hd.Alias = uhd.Alias
	return hd
}

func setProfileData(userData *pbApi.ViewUserResponse) ProfileData {
	var pd ProfileData
	if userData != nil {
		pd = ProfileData{
			Patillavatar: userData.PicUrl,
			Alias:        userData.Alias,
			Username:     userData.Username,
			Followers:    len(userData.FollowersIds),
			Following:    len(userData.FollowingIds),
			Description:  userData.About,
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

func contentToContentRenderer(pbRule *pbApi.ContentRule, userId string)	ContentRenderer {
	bc := setBasicContent(pbRule, userId)

	metadata := pbRule.Data.Metadata

	threadId := metadata.Id
	sectionId := strings.Replace(strings.ToLower(metadata.Section), " ", "", -1)

	threadLink := fmt.Sprintf("/%s/%s", sectionId, threadId)
	saveLink := fmt.Sprintf("%s/save", threadLink)
	unsaveLink := fmt.Sprintf("%s/unsave", threadLink)
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
		UnsaveLink:   unsaveLink,
		Saved:        saved,
		ReplyLink:    replyLink,
	}
}

func contentToOverviewRendererSet(pbRule *pbApi.ContentRule, userId string) 
	OverviewRenderer {

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
		unsaveLink := fmt.Sprintf("%s/unsave", threadLink)
		replyLink := fmt.Sprintf("%s/comment", threadLink)
		bc.UpvoteLink = fmt.Sprintf("%s/upvote", threadLink)
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
			UnsaveLink:   unsaveLink,
			Saved:        saved,
			ReplyLink:    replyLink,
		}
	// it's a COMMENT
	case *pbApi.ContentRule_CommentCtx:
		// comment context
		comCtx := ctx.CommentCtx

		bc.UpvoteLink = fmt.Sprintf("%s/upvote?c_id=%s", threadLink, comCtx.Id)
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
func contentsToOverviewRendererSet(pbRuleSet []*pbApi.ContentRule, userId string) 
	[]OverviewRenderer {
	var ovwRendererSet []OverviewRenderer

	for _, pbRule := range pbRuleSet {
		ovwRenderer := contentToOverviewRendererSet(pbRule, userId)
		ovwRendererset = append(ovwRendererset, ovwRenderer)
	}
	return ovwRendererSet
}

// setBasicContent returns a *BasicContent object filled with data retrieved from a
// *pbApi.ContentRule. userId is used to check whether the user has upvoted the content.
func setBasicContent(pbRule *pbApi.ContentRule, userId string) *BasicContent {
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
		PublishDate: content.PublishDate,
		ThreadLink:  threadLink,
		SectionLink: sectionLink,
	}
}
