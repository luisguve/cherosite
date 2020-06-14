package templates

type BasicUserData struct {
	Patillavatar string // URL to user profile pic
	Alias        string
	Username     string
	Description  string
}

type ProfileData struct {
	BasicUserData
	Followers int
	Following int
}

type Notif struct {
	Permalink string
	Title     string
	Message   string
	Date      string
}

// UserHeader holds information about the user currently logged in
type UserHeader struct {
	Alias         string
	UnreadNotifs []*Notif
	ReadNotifs   []*Notif
}

type RecycleType struct {
	// Content type identifier
	Label string
	// Link to send request to recycle content
	Link string
}

// HeaderData holds information to render the header section of a page.
type HeaderData struct {
	User *UserHeader
	// A page shows its content grouped together in different sections, 
	// e.g. the dashboard contains feed, user activity and user saved content, 
	// but profile pages contains only user activity.
	// RecycleTypes holds the possible content types a user can select to recycle.
	RecycleTypes []RecycleType
}

type ProfileView struct {
	HeaderData
	ProfileData
	Activity   []OverviewRenderer
	// IsFollower indicates whether the current user is following another user,
	// in a context in which it is viewing another user's profile or content
	IsFollower bool
}

type DashboardView struct {
	HeaderData
	Followers    int
	Following    int
	Activity     []OverviewRenderer
	SavedContent []OverviewRenderer
	Feed         []OverviewRenderer
}

type ExploreView struct {
	HeaderData
	Feed []OverviewRenderer
}

type SectionView struct {
	HeaderData
	Feed        []OverviewRenderer
	SectionName string
}

type ThreadView struct {
	HeaderData
	Content  ContentRenderer
	Comments []OverviewRenderer
}

type MyProfileView struct {
	HeaderData
	BasicUserData
}
