package claps

// ClapRequest is sent by the reader to add claps.
type ClapRequest struct {
	SessionID string `json:"session_id" validate:"required"`
	Count     int    `json:"count"      validate:"required,min=1,max=50"`
}

// ClapResponse is the response returned after clapping.
type ClapResponse struct {
	PostID     string `json:"post_id"`
	TotalClaps int64  `json:"total_claps"`
	UserClaps  int    `json:"user_claps"`
}
