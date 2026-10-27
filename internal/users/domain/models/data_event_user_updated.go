package models

// DataEventUserUpdated avisa al ETL que una cuenta cambió (datos, estado,
// roles) para resincronizar DimUser sin esperar al batch. Lo publican tanto
// users/app como riders/app y shops/app, porque otorgar el rol Rider u Owner
// también muta users.user_roles y por lo tanto DimUser.Roles. El payload no
// es la fuente de verdad, por eso solo lleva el id.
type DataEventUserUpdated struct {
	MessageType string `json:"type"`
	UserID      string `json:"user_id"`
}

func (d *DataEventUserUpdated) BuildFromUserID(userID string) {
	d.MessageType = "user_updated"
	d.UserID = userID
}
