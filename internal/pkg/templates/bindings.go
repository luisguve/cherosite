package templates

import(
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

func DataToMyProfileView(userData *pb.BasicUserData, userHeader *pb.UserHeaderData)
	*MyProfileView {

	var readNotifs []*Notif
	var unreadNotifs []*Notif
	// set read notifs
	for pbNotif := range userHeader.ReadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp,
		}
		readNotifs = append(readNotifs, notif)
	}
	// set unread notifs
	for pbNotif := range userHeader.UnreadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp,
		}
		unreadNotifs = append(unreadNotifs, notif)
	}

	headerData := HeaderData{
		RecycleTypes: nil,
		User:         &UserHeader{
			Alias:        userHeader.Alias,
			UnreadNotifs: unreadNotifs,
			ReadNotifs:   readNotifs,
		},
	}
	// set user profile data
	profileData := ProfileData{
		Patillavatar: userData.PicUrl,
		Alias:        userData.Alias,
		Username:     userData.Username,
		Description:  userData.About,
	}

	return &MyProfileView{
		HeaderData:  headerData,
		ProfileData: profileData,
	}
}

func DataToProfileView(userData *pb.ViewUserResponse, userHeader *pb.UserHeaderData, 
	activity []*pb.ActivityRule) *ProfileView {
		
}