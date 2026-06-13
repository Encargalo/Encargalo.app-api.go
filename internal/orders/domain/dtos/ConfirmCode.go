package dtos

import "fmt"

type ConfirmCodeRequest struct {
	Code string `json:"code"`
}

func (r *ConfirmCodeRequest) Validate() error {
	if len(r.Code) != 6 {
		return fmt.Errorf("code must be exactly 6 digits")
	}
	for _, c := range r.Code {
		if c < '0' || c > '9' {
			return fmt.Errorf("code must contain only digits")
		}
	}
	return nil
}
