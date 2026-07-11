package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	emplService "Encargalo.app-api.go/internal/employee/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"

	"github.com/google/uuid"
)

const trendingResultLimit = 10

type productsApp struct {
	repo        ports.ProductsRepo
	uploadimage uploadimage.UploadImage
	emplService emplService.EmployeeApp
	bcv         bcvPorts.BcvApp
	shopPromo   ports.ShopLaunchPromoProvider
}

func NewProductsApp(repo ports.ProductsRepo, uploadimage uploadimage.UploadImage, emplService emplService.EmployeeApp, bcv bcvPorts.BcvApp, shopPromo ports.ShopLaunchPromoProvider) ports.ProductsApp {
	return &productsApp{repo, uploadimage, emplService, bcv, shopPromo}
}

// launchPromoEndsAt resuelve la ventana de promo día 1 del negocio; si el
// puerto cross-domain falla, se ignora la promo en vez de romper el catálogo.
func (p *productsApp) launchPromoEndsAt(ctx context.Context, shopID uuid.UUID) *time.Time {
	endsAt, err := p.shopPromo.GetLaunchPromoEndsAt(ctx, shopID)
	if err != nil {
		slog.Warn("launch promo window unavailable, catalog shown without discount", "shopID", shopID, "error", err)
		return nil
	}
	return endsAt
}

func (p *productsApp) bcvRate(ctx context.Context) float64 {
	rate, err := p.bcv.GetRate(ctx)
	if err != nil {
		slog.Warn("bcv rate unavailable, bs prices will be 0", "error", err)
		return 0
	}
	return rate.Rate
}

func applyRateToCategories(rate float64, data *dtos.CategoriesResponse) {
	for i := range *data {
		for j := range (*data)[i].Items {
			item := &(*data)[i].Items[j]
			item.Bs = money.RoundBs(item.Ref * rate)
			item.EffectiveBs = money.RoundBs(item.EffectiveRef * rate)
		}
	}
}

func applyRateToItems(rate float64, data *dtos.ItemsResponse) {
	for i := range *data {
		item := &(*data)[i]
		item.Bs = money.RoundBs(item.Ref * rate)
		item.EffectiveBs = money.RoundBs(item.EffectiveRef * rate)
	}
}

// applyLaunchDiscount aplica, sobre el precio ya rebajado por el descuento
// propio del negocio (EffectiveRef), el 50% de la promo de lanzamiento del
// día 1 si la ventana sigue vigente. Devuelve si se aplicó o no.
func applyLaunchDiscount(item *dtos.ItemResponse, endsAt *time.Time) {
	if endsAt == nil || !time.Now().Before(*endsAt) {
		return
	}
	item.EffectiveRef = money.RoundBs(item.EffectiveRef * 0.5)
	item.HasLaunchPromo = true
}

func applyLaunchDiscountToItems(items *dtos.ItemsResponse, endsAt *time.Time) {
	for i := range *items {
		applyLaunchDiscount(&(*items)[i], endsAt)
	}
}

func applyLaunchDiscountToCategories(data *dtos.CategoriesResponse, endsAt *time.Time) {
	for i := range *data {
		applyLaunchDiscountToItems(&(*data)[i].Items, endsAt)
	}
}

func applyRateToAdditions(rate float64, data *dtos.AdditionsResponse) {
	for i := range *data {
		(*data)[i].Bs = money.RoundBs((*data)[i].Ref * rate)
	}
}

func applyRateToTrending(rate float64, data *dtos.TrendingResponse) {
	for i := range *data {
		(*data)[i].Product.Bs = money.RoundBs((*data)[i].Product.Ref * rate)
		(*data)[i].Product.EffectiveBs = money.RoundBs((*data)[i].Product.EffectiveRef * rate)
		(*data)[i].Shop.DeliveryFeeBs = money.RoundBs((*data)[i].Shop.DeliveryFee * rate)
	}
}

func toWindows(schedules []dtos.ScheduleResponse) []bizclock.Window {
	windows := make([]bizclock.Window, len(schedules))
	for i, s := range schedules {
		windows[i] = bizclock.Window{
			DayOfWeek: s.DayOfWeek,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			Enabled:   s.Enabled,
		}
	}
	return windows
}

func filterCategoriesBySchedule(data dtos.CategoriesResponse, now time.Time) dtos.CategoriesResponse {
	filtered := make(dtos.CategoriesResponse, 0, len(data))
	for _, cat := range data {
		if !bizclock.IsAvailableAt(toWindows(cat.Schedules), now) {
			continue
		}
		filtered = append(filtered, cat)
	}
	return filtered
}

func filterItemsBySchedule(data dtos.ItemsResponse, now time.Time) dtos.ItemsResponse {
	filtered := make(dtos.ItemsResponse, 0, len(data))
	for _, item := range data {
		if !bizclock.IsAvailableAt(toWindows(item.Schedules), now) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterTrendingBySchedule(data dtos.TrendingResponse, now time.Time) dtos.TrendingResponse {
	filtered := make(dtos.TrendingResponse, 0, len(data))
	for _, entry := range data {
		if !bizclock.IsAvailableAt(toWindows(entry.Product.Schedules), now) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
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

	if err := p.validateFlavorsBelongToCategory(ctx, item.CategoryID, item.FlavorIDs); err != nil {
		return err
	}

	itemModel := models.Item{}
	itemModel.BuildToModel(item, resp.SecureURL, publicID, sha256Hash(imageBytes))

	if err := p.repo.RegisterItem(ctx, &itemModel); err != nil {
		return err
	}

	if item.HasFlavors && len(item.FlavorIDs) > 0 {
		if err := p.repo.RegisterItemFlavors(ctx, itemModel.ID, item.FlavorIDs); err != nil {
			return err
		}
	}

	return nil
}

// validateFlavorsBelongToCategory asegura que un producto solo pueda tener
// asignados sabores que pertenezcan a su propia categoría.
func (p *productsApp) validateFlavorsBelongToCategory(ctx context.Context, categoryID uuid.UUID, flavorIDs []uuid.UUID) error {
	if len(flavorIDs) == 0 {
		return nil
	}

	flavors, err := p.repo.SearchFlavorsByIDs(ctx, flavorIDs)
	if err != nil {
		return err
	}

	found := make(map[uuid.UUID]bool, len(flavors))
	for _, flavor := range flavors {
		if flavor.CategoryID != categoryID {
			return errcustom.ErrInvalidFlavors
		}
		found[flavor.ID] = true
	}

	for _, id := range flavorIDs {
		if !found[id] {
			return errcustom.ErrInvalidFlavors
		}
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

	// Imagen: solo subir si es diferente (comparar hash)
	imageURL, imagePublicID, imageHash := itemModel.Image, itemModel.ImagePublicID, itemModel.ImageHash
	if item.Image != "" {
		imageBytes, err := base64ToBytes(item.Image)
		if err != nil {
			return errcustom.ErrConvertingImage
		}
		newHash := sha256Hash(imageBytes)
		if newHash != itemModel.ImageHash {
			resp, err := p.uploadimage.UploadImage(ctx, item.Image)
			if err != nil {
				return errcustom.ErrUploadingImage
			}
			imageURL = resp.SecureURL
			imagePublicID = strings.ReplaceAll(resp.PublicID, "\\", "/")
			imageHash = newHash
		}
	}

	// has_flavors: derivar según FlavorIDs si se envían, mantener el actual si no
	hasFlavors := itemModel.HasFlavors
	if item.FlavorIDs != nil {
		hasFlavors = len(*item.FlavorIDs) > 0
		if err := p.validateFlavorsBelongToCategory(ctx, itemModel.CategoryID, *item.FlavorIDs); err != nil {
			return err
		}
	}

	var ItemUpdateModel models.UpdateItem
	ItemUpdateModel.BuildToModel(item, itemModel.ID, hasFlavors, imageURL, imagePublicID, imageHash)

	if err := p.repo.UpdateItem(ctx, &ItemUpdateModel, itemModel.ShopID); err != nil {
		return err
	}

	// Sabores: si se enviaron (incluso vacío), reemplazar
	if item.FlavorIDs != nil {
		if err := p.repo.DeleteItemFlavors(ctx, itemID); err != nil {
			return err
		}
		if len(*item.FlavorIDs) > 0 {
			if err := p.repo.RegisterItemFlavors(ctx, itemID, *item.FlavorIDs); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *productsApp) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {
	data, err := p.repo.SearchProductsByShopID(ctx, shopID)
	if err != nil {
		return nil, err
	}
	data = filterCategoriesBySchedule(data, bizclock.Now())
	applyLaunchDiscountToCategories(&data, p.launchPromoEndsAt(ctx, shopID))
	applyRateToCategories(p.bcvRate(ctx), &data)
	return data, nil
}

// SearchProductsByShopIDForOwner es la versión sin filtrar por horario, usada por el
// dashboard del dueño (GetMyProducts) para que siempre pueda ver y editar todo su catálogo.
func (p *productsApp) SearchProductsByShopIDForOwner(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {
	data, err := p.repo.SearchProductsByShopID(ctx, shopID)
	if err != nil {
		return nil, err
	}
	applyRateToCategories(p.bcvRate(ctx), &data)
	return data, nil
}

func (p *productsApp) SearchAdditionsByShopID(ctx context.Context, categoryID uuid.UUID) (dtos.AdditionsResponse, error) {
	data, err := p.repo.SearchAdditionsByShopID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	applyRateToAdditions(p.bcvRate(ctx), &data)
	return data, nil
}

func (p *productsApp) SearchFlavorsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.FlavorsResponse, error) {
	return p.repo.SearchFlavorsByItemID(ctx, itemID)
}

func (p *productsApp) SearchFlavorsByCategoryID(ctx context.Context, categoryID uuid.UUID) (dtos.FlavorsResponse, error) {
	return p.repo.SearchFlavorsByCategoryID(ctx, categoryID)
}

func (p *productsApp) SearchBestSellersByShopID(ctx context.Context, shopID uuid.UUID) (dtos.ItemsResponse, error) {
	data, err := p.repo.SearchBestSellersByShopID(ctx, shopID)
	if err != nil {
		return nil, err
	}
	data = filterItemsBySchedule(data, bizclock.Now())
	applyLaunchDiscountToItems(&data, p.launchPromoEndsAt(ctx, shopID))
	applyRateToItems(p.bcvRate(ctx), &data)
	return data, nil
}

func (p *productsApp) AddSoldItem(ctx context.Context, item models.DataProduceOrder) error {
	return p.repo.AddSoldItem(ctx, item)
}

func (p *productsApp) GetTrendingByCoords(ctx context.Context, coords dtos.TrendingCoords) (dtos.TrendingResponse, error) {
	data, err := p.repo.GetTrendingByCoords(ctx, coords)
	if err != nil {
		return nil, err
	}
	data = filterTrendingBySchedule(data, bizclock.Now())
	if len(data) > trendingResultLimit {
		data = data[:trendingResultLimit]
	}
	applyRateToTrending(p.bcvRate(ctx), &data)
	return data, nil
}

func (p *productsApp) RegisterCategory(ctx context.Context, category dtos.RegisterCategory) error {
	categoryModel := models.Category{
		ShopID:          category.ShopID,
		Name:            category.Name,
		MacroCategoryID: category.MacroCategoryID,
	}

	if err := p.repo.RegisterCategory(ctx, &categoryModel); err != nil {
		return err
	}

	if len(category.Schedules) == 0 {
		return nil
	}

	schedules := buildScheduleModels(categoryModel.ID, category.Schedules)
	return p.repo.ReplaceCategorySchedules(ctx, categoryModel.ID, categoryModel.ShopID, schedules)
}

func (p *productsApp) UpdateCategory(ctx context.Context, category dtos.UpdateCategory, categoryID, userID uuid.UUID) error {
	categoryModel, err := p.repo.SearchCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}

	if categoryModel == nil {
		return errcustom.ErrNotFound
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, userID)
	if err != nil {
		return err
	}

	if categoryModel.ShopID != empl.ShopID {
		return errcustom.ErrUnautorized
	}

	var updateModel models.UpdateCategory
	updateModel.BuildToModel(category, categoryID)

	if err := p.repo.UpdateCategory(ctx, &updateModel, categoryModel.ShopID); err != nil {
		return err
	}

	if category.Schedules == nil {
		return nil
	}

	schedules := buildScheduleModels(categoryID, *category.Schedules)
	return p.repo.ReplaceCategorySchedules(ctx, categoryID, categoryModel.ShopID, schedules)
}

func buildScheduleModels(categoryID uuid.UUID, inputs []dtos.ScheduleInput) []models.Schedule {
	schedules := make([]models.Schedule, len(inputs))
	for i, s := range inputs {
		schedules[i] = models.Schedule{
			CategoryID: &categoryID,
			StartTime:  s.StartTime,
			EndTime:    s.EndTime,
			Enabled:    s.EnabledOrDefault(),
		}
	}
	return schedules
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

func (p *productsApp) RegisterFlavor(ctx context.Context, flavor dtos.RegisterFlavor) error {
	categoryModel, err := p.repo.SearchCategoryByID(ctx, flavor.CategoryID)
	if err != nil {
		return err
	}

	if categoryModel == nil {
		return errcustom.ErrNotFound
	}

	if categoryModel.ShopID != flavor.ShopID {
		return errcustom.ErrUnautorized
	}

	flavorModel := models.Flavor{}
	flavorModel.BuildToModel(flavor)

	return p.repo.RegisterFlavor(ctx, &flavorModel)
}

func (p *productsApp) UpdateFlavor(ctx context.Context, flavor dtos.UpdateFlavor, flavorID, userID uuid.UUID) error {
	flavorModel, err := p.repo.SearchFlavorByID(ctx, flavorID)
	if err != nil {
		return err
	}

	if flavorModel == nil {
		return errcustom.ErrNotFound
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, userID)
	if err != nil {
		return err
	}

	if flavorModel.ShopID != empl.ShopID {
		return errcustom.ErrUnautorized
	}

	var updateModel models.UpdateFlavor
	updateModel.BuildToModel(flavor, flavorID)

	return p.repo.UpdateFlavor(ctx, &updateModel, flavorModel.ShopID, flavorModel.CategoryID)
}

func (p *productsApp) DeleteFlavor(ctx context.Context, flavorID, userID uuid.UUID) error {
	flavorModel, err := p.repo.SearchFlavorByID(ctx, flavorID)
	if err != nil {
		return err
	}

	if flavorModel == nil {
		return errcustom.ErrNotFound
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, userID)
	if err != nil {
		return err
	}

	if flavorModel.ShopID != empl.ShopID {
		return errcustom.ErrUnautorized
	}

	return p.repo.DeleteFlavor(ctx, flavorModel)
}

func (p *productsApp) SearchMacroCategories(ctx context.Context) (dtos.MacroCategoriesResponse, error) {
	macroCategories, err := p.repo.SearchMacroCategories(ctx)
	if err != nil {
		return nil, err
	}
	return macroCategories.ToDomainDTO(), nil
}

// validateDiscountValue asegura que el valor del descuento tenga sentido
// según su tipo: porcentaje entre 1 y 99, o monto fijo positivo y menor al
// precio del producto (nunca puede dejarlo en 0 o negativo).
func validateDiscountValue(discountType string, value float64, itemPrice int) error {
	switch discountType {
	case models.DiscountTypePercentage:
		if value < 1 || value > 99 {
			return errcustom.ErrInvalidDiscountValue
		}
	case models.DiscountTypeFixed:
		if value <= 0 || value >= float64(itemPrice) {
			return errcustom.ErrInvalidDiscountValue
		}
	default:
		return errcustom.ErrInvalidDiscountValue
	}
	return nil
}

func (p *productsApp) RegisterItemDiscount(ctx context.Context, discount dtos.RegisterItemDiscount) error {
	itemModel, err := p.repo.SearchItemByID(ctx, discount.ItemID)
	if err != nil {
		return err
	}

	if itemModel == nil {
		return errcustom.ErrNotFound
	}

	if itemModel.ShopID != discount.ShopID {
		return errcustom.ErrUnautorized
	}

	if err := validateDiscountValue(discount.DiscountType, discount.Value, itemModel.Price); err != nil {
		return err
	}

	discountModel := models.ItemDiscount{}
	discountModel.BuildToModel(discount)

	return p.repo.RegisterItemDiscount(ctx, &discountModel)
}

func (p *productsApp) UpdateItemDiscount(ctx context.Context, discount dtos.UpdateItemDiscount, discountID, userID uuid.UUID) error {
	discountModel, err := p.repo.SearchItemDiscountByID(ctx, discountID)
	if err != nil {
		return err
	}

	if discountModel == nil {
		return errcustom.ErrNotFound
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, userID)
	if err != nil {
		return err
	}

	if discountModel.ShopID != empl.ShopID {
		return errcustom.ErrUnautorized
	}

	itemModel, err := p.repo.SearchItemByID(ctx, discountModel.ItemID)
	if err != nil {
		return err
	}

	if err := validateDiscountValue(discount.DiscountType, discount.Value, itemModel.Price); err != nil {
		return err
	}

	var updateModel models.UpdateItemDiscount
	updateModel.BuildToModel(discount, discountID)

	return p.repo.UpdateItemDiscount(ctx, &updateModel, discountModel.ShopID, discountModel.ItemID)
}

func (p *productsApp) DeleteItemDiscount(ctx context.Context, discountID, userID uuid.UUID) error {
	discountModel, err := p.repo.SearchItemDiscountByID(ctx, discountID)
	if err != nil {
		return err
	}

	if discountModel == nil {
		return errcustom.ErrNotFound
	}

	empl, err := p.emplService.SearchEmployeeByID(ctx, userID)
	if err != nil {
		return err
	}

	if discountModel.ShopID != empl.ShopID {
		return errcustom.ErrUnautorized
	}

	return p.repo.DeleteItemDiscount(ctx, discountModel)
}
