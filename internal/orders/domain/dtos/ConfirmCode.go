package dtos

import "fmt"

type ConfirmCodeRequest struct {
	Code string `json:"code"`
}

func (r *ConfirmCodeRequest) Validate() error {
	if len(r.Code) != 4 {
		return fmt.Errorf("code debe tener exactamente 4 dígitos")
	}
	for _, c := range r.Code {
		if c < '0' || c > '9' {
			return fmt.Errorf("code debe contener solo dígitos")
		}
	}
	return nil
}
