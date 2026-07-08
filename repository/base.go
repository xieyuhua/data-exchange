package repository

// Pagination 通用分页参数
type Pagination struct {
	Page     int
	PageSize int
}

// Normalize 规整分页参数，避免越界
func (p Pagination) Normalize() Pagination {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	return p
}

// Offset 计算 SQL 偏移量
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
