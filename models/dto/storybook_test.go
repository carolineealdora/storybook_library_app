package dto

import "testing"

func TestPageQuery_LimitOffset_DefaultsAndPaging(t *testing.T) {
	cases := []struct {
		q                  PageQuery
		page, size, offset int
	}{
		{PageQuery{}, 1, 20, 0},
		{PageQuery{Page: 3, PageSize: 10}, 3, 10, 20},
		{PageQuery{Page: 2}, 2, 20, 20},
	}
	for _, c := range cases {
		p := c.q.Normalize()
		if p.Page != c.page || p.PageSize != c.size || p.Offset() != c.offset {
			t.Errorf("%+v -> %+v offset %d, want page %d size %d offset %d", c.q, p, p.Offset(), c.page, c.size, c.offset)
		}
	}
}
