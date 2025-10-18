package models

import (
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ItemsRules []ItemRule

type ItemRule struct {
	bun.BaseModel `bun:"table:products.items_rules"`

	ID           uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ItemID       uuid.UUID `bun:"item_id,notnull" json:"item_id"`
	RuleKey      string    `bun:"rule_key,notnull" json:"rule_key"`
	RuleValue    int       `bun:"rule_value,notnull" json:"rule_value"`
	SelectorType string    `bun:"selector_type,notnull" json:"selector_type"`

	CreatedAt time.Time  `bun:"created_at,default:now()" json:"created_at"`
	UpdatedAt time.Time  `bun:"updated_at,default:now()" json:"updated_at"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at,omitempty"`
}

func (ir *ItemRule) ToDomainDTO() dtos.ItemRule {
	return dtos.ItemRule{
		ID:           ir.ID,
		RuleKey:      ir.RuleKey,
		RuleValue:    ir.RuleValue,
		SelectorType: ir.SelectorType,
	}
}

func (ir *ItemsRules) ToDomainDTO() dtos.ItemsRules {
	var rules dtos.ItemsRules

	for _, v := range *ir {
		rules.Add(v.ToDomainDTO())
	}

	return rules
}
