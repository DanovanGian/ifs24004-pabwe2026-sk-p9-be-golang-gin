package post

type PostRequest struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

type Stats struct {
	Total     int64 `json:"total"`
	Published int64 `json:"published"`
	Draft     int64 `json:"draft"`
}

type UserSummary struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	TotalPosts     int64  `json:"total_posts"`
	TotalPublished int64  `json:"total_published"`
}
