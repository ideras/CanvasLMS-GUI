package models

// AnnouncementAuthor identifies who posted an announcement.
type AnnouncementAuthor struct {
	ID          int    `json:"id"`
	DisplayName string `json:"display_name"`
}

// Announcement represents a Canvas announcement (a discussion topic with
// is_announcement=true).  It is returned by the discussion_topics API.
type Announcement struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// Message is the HTML body of the announcement.
	Message  string `json:"message"`
	PostedAt string `json:"posted_at"`
	// DelayedPostAt is set when the announcement is scheduled for future posting.
	DelayedPostAt *string `json:"delayed_post_at"`
	// Author identifies who posted the announcement.
	Author AnnouncementAuthor `json:"author"`
	// ReadState is "read" or "unread" for the current user.
	ReadState string `json:"read_state"`
	// DiscussionSubentryCount is the number of replies.
	DiscussionSubentryCount int `json:"discussion_subentry_count"`
	// URL is the Canvas web URL for the announcement.
	URL string `json:"url"`
}
