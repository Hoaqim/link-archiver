package queue

type Job struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Req struct {
	URL string `json:"url"`
}
