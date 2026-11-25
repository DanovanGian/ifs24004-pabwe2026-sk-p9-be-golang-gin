package ai

type GenerateRequest struct {
	Topic            string `json:"topic" binding:"required"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

type Article struct {
	Title    string         `json:"title"`
	Category string         `json:"category"`
	Excerpt  string         `json:"excerpt"`
	Content  string         `json:"content"`
	Sources  []SearchResult `json:"sources"`
}
