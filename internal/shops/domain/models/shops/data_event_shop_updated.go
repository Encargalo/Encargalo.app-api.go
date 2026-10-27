package shops

// DataEventShopUpdated avisa al ETL que un local se creó o cambió (datos
// de perfil, abrir/cerrar) para resincronizar DimShop sin esperar al batch.
// El payload no es la fuente de verdad, por eso solo lleva el id.
type DataEventShopUpdated struct {
	MessageType string `json:"type"`
	ShopID      string `json:"shop_id"`
}

func (d *DataEventShopUpdated) BuildFromShopID(shopID string) {
	d.MessageType = "shop_updated"
	d.ShopID = shopID
}
