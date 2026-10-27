package models

// DataEventRiderUpdated avisa al ETL que un conductor cambió (perfil,
// solicitud, revisión, activación/suspensión, contacto) para resincronizar
// DimRider sin esperar al batch. El payload no es la fuente de verdad, por
// eso solo lleva el id.
type DataEventRiderUpdated struct {
	MessageType string `json:"type"`
	RiderID     string `json:"rider_id"`
}

func (d *DataEventRiderUpdated) BuildFromRiderID(riderID string) {
	d.MessageType = "rider_updated"
	d.RiderID = riderID
}
