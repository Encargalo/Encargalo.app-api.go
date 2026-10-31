package repo

import (
	"context"
	"errors"

	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shipmentsDtos "Encargalo.app-api.go/internal/shipments/domain/dtos"
	shipmentsPorts "Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
)

type shipmentRiderPublicInfoLookup struct {
	profiles ridersPorts.RiderProfileRepo
}

func NewShipmentRiderPublicInfoLookup(profiles ridersPorts.RiderProfileRepo) shipmentsPorts.RiderPublicInfoLookup {
	return &shipmentRiderPublicInfoLookup{profiles: profiles}
}

func (l *shipmentRiderPublicInfoLookup) GetByRiderID(ctx context.Context, riderID uuid.UUID) (shipmentsDtos.RiderPublicInfo, error) {
	name, surName, phone, _, err := l.profiles.GetAccountBasics(ctx, riderID)
	if err != nil {
		return shipmentsDtos.RiderPublicInfo{}, err
	}
	info := shipmentsDtos.RiderPublicInfo{
		Name:    name,
		SurName: surName,
		Phone:   phone,
	}

	profile, err := l.profiles.GetByUserID(ctx, riderID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return info, nil
		}
		return shipmentsDtos.RiderPublicInfo{}, err
	}
	info.ProfilePhotoURL = profile.ProfilePhotoURL
	info.VehicleType = profile.VehicleType
	info.VehiclePlate = profile.VehiclePlate
	info.VehicleBrand = profile.VehicleBrand
	info.VehicleModel = profile.VehicleModel
	info.VehicleColor = profile.VehicleColor
	return info, nil
}
