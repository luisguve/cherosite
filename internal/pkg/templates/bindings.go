package templates

import(
	pb "github.com/luisguve/cheropatilla/internal/cheropatillapb"
)

func DataToMyProfileView(userData *pb.BasicUserData, userHeader *pb.UserHeaderData)
	*MyProfileView {

	var notifs []*Notif
	// set read notifs
	for pbNotif := range userHeader.ReadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp,
		}
		notifs = append(notifs, notif)
	}
	// set unread notifs
	for pbNotif := range userHeader.UnreadNotifs {
		notif := &Notif{
			Permalink: pbNotif.Permalink,
			Title:     pbNotif.Subject,
			Message:   pbNotif.Message,
			Date:      pbNotif.Timestamp,
		}
		notifs = append(notifs, notif)
	}

	headerData := HeaderData{
		User: &UserHeader{
			Alias: userHeader.Alias,
			Notifications: notifs,
		},
		RecycleTypes: nil,
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