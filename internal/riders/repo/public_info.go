package repo

import (
	"context"
	"errors"

	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	tripsPorts "Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type riderPublicInfoLookup struct {
	profiles ridersPorts.RiderProfileRepo
}

func NewRiderPublicInfoLookup(profiles ridersPorts.RiderProfileRepo) tripsPorts.RiderPublicInfoLookup {
	return &riderPublicInfoLookup{profiles: profiles}
}

func (l *riderPublicInfoLookup) GetByRiderID(ctx context.Context, riderID uuid.UUID) (tripsDtos.RiderPublicInfo, error) {
	name, surName, phone, _, err := l.profiles.GetAccountBasics(ctx, riderID)
	if err != nil {
		return tripsDtos.RiderPublicInfo{}, err
	}
	info := tripsDtos.RiderPublicInfo{
		Name:    name,
		SurName: surName,
		Phone:   phone,
	}

	profile, err := l.profiles.GetByUserID(ctx, riderID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return info, nil
		}
		return tripsDtos.RiderPublicInfo{}, err
	}
	info.ProfilePhotoURL = profile.ProfilePhotoURL
	info.VehicleType = profile.VehicleType
	info.VehiclePlate = profile.VehiclePlate
	info.VehicleBrand = profile.VehicleBrand
	info.VehicleModel = profile.VehicleModel
	info.VehicleColor = profile.VehicleColor
	return info, nil
}
