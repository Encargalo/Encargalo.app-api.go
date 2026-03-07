package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	emplService "Encargalo.app-api.go/internal/employee/domain/ports"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"

	"github.com/google/uuid"
)

type productsApp struct {
	repo        ports.ProductsRepo
	uploadimage uploadimage.UploadImage
	emplService emplService.EmployeeApp
}

func NewProductsApp(repo ports.ProductsRepo, uploadimage uploadimage.UploadImage, emplService emplService.EmployeeApp) ports.ProductsApp {
	return &productsApp{repo, uploadimage, emplService}
}

func (p *productsApp) RegisterItem(ctx context.Context, item dtos.RegisterItem) error {

	resp, err := p.uploadimage.UploadImage(ctx, item.Image)
	if err != nil {
		return errcustom.ErrUploadingImage
	}

	imageBytes, err := base64ToBytes(item.Image)
	if err != nil {
		return errcustom.ErrConvertingImage
	}

	publicID := strings.ReplaceAll(resp.PublicID, "\\", "/")

	itemModel := models.Item{}
	itemModel.BuildToModel(item, resp.SecureURL, publicID, sha256Hash(imageBytes))

	if err := p.repo.RegisterItem(ctx, &itemModel); err != nil {
		return err
	}

	return nil
}

func (p *productsApp) UpdateItem(ctx context.Context, item dtos.UpdateItem, itemID, userID uuid.UUID) error {

	itemModel, err := p.repo.SearchItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	if itemModel == nil {
		return errcustom.ErrNotFound
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, userID)
	if err != nil {
		return err
	}

	if itemModel.ShopID != empl.ShopID {
		return errcustom.ErrUnautorized
	}

	var ItemUpdateModel models.UpdateItem
	ItemUpdateModel.BuildToModel(item, itemModel.ID)

	if err := p.repo.UpdateItem(ctx, &ItemUpdateModel, itemModel.ShopID); err != nil {
		return err
	}

	return nil
}

func (p *productsApp) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {
	return p.repo.SearchProductsByShopID(ctx, shopID)
}

func (p *productsApp) SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (dtos.AdditionsResponse, error) {
	return p.repo.SearchAdditionsByShopID(ctx, categoryID)
}

func (p *productsApp) SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error) {
	return p.repo.SearchFlavorsByItemID(ctx, itemID)
}
func (p *productsApp) SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error) {
	return p.repo.SearchBestSellersByShopID(ctx, shopID)
}

func (p *productsApp) AddSoldItem(ctx context.Context, item models.DataProduceOrder) error {
	return p.repo.AddSoldItem(ctx, item)
}

func base64ToBytes(b64 string) ([]byte, error) {
	if strings.Contains(b64, ",") {
		b64 = strings.Split(b64, ",")[1]
	}
	return base64.StdEncoding.DecodeString(b64)
}

func sha256Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
