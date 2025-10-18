package dtos

import (
	"github.com/google/uuid"
)

type ItemsRules []ItemRule
type ItemRule struct {
	ID           uuid.UUID `json:"id" example:"71c62783-820d-46f8-957d-c8d02db97264"`
	ItemID       uuid.UUID `json:"item_id" example:"7e441237-a818-42d2-bb54-9b8747198305"`
	RuleKey      string    `json:"rule_key" example:"max_flavors"`
	RuleValue    int       `json:"rule_value" example:"2"`
	SelectorType string    `json:"selector_type" example:"multi_select"`
}

func (ir *ItemsRules) Add(rule ItemRule) {
	*ir = append(*ir, rule)
}
