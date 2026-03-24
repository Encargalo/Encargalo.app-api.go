package pagination

type Params struct {
	Page  int
	Limit int
}

func (p *Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

func TotalPages(total, limit int) int {
	return (total + limit - 1) / limit
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type Response[T any] struct {
	Data       T    `json:"data"`
	Pagination Meta `json:"pagination"`
}
