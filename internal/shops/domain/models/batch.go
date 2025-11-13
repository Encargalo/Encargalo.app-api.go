package models

import "github.com/uptrace/bun"

type Batch struct {
	bun.BaseModel `bun:"table:business.batches"`

	ID     string `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ShopID string `bun:"shop_id"`
	Batch  int    `bun:"batch"`
	Opened bool   `bun:"opened,default:true"`
}
