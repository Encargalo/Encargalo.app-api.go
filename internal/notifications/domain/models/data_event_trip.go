package models

import "github.com/redis/go-redis/v9"

type DataEventTrip struct {
	MessageType string
	TripID      string
}

func (d *DataEventTrip) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.TripID = asString(msg.Values["trip_id"])
}
