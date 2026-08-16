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
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"

	"github.com/google/uuid"
)

const trendingResultLimit = 10

type productsApp struct {
	repo        ports.ProductsRepo
	uploadimage uploadimage.UploadImage
	bcv         bcvPorts.BcvApp
	shopPromo   ports.ShopLaunchPromoProvider
}

func NewProductsApp(repo ports.ProductsRepo, uploadimage uploadimage.UploadImage, bcv bcvPorts.BcvApp, shopPromo ports.ShopLaunchPromoProvider) ports.ProductsApp {
	return &productsApp{repo, uploadimage, bcv, shopPromo}
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
		applyRateToItems(rate, &(*data)[i].Items)
	}
}

func applyRateToItems(rate float64, data *dtos.ItemsResponse) {
	for i := range *data {
		item := &(*data)[i]
		item.Bs = money.RoundBs(item.Ref * rate)
		item.EffectiveBs = money.RoundBs(item.EffectiveRef * rate)
		applyRateToAdditions(rate, &item.Details.Additions)
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
	if data == nil {
		return
	}
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

	selectedFlavors, err := p.resolveItemFlavors(ctx, item.CategoryID, item.FlavorIDs)
	if err != nil {
		return err
	}

	itemDetails, err := p.buildItemDetails(ctx, item.CategoryID, selectedFlavors, item.MinSelection, item.MaxSelection, models.DefaultItemDetails())
	if err != nil {
		return err
	}

	selectedAdditions, err := p.resolveItemAdditions(ctx, item.ShopID, item.AdditionIDs)
	if err != nil {
		return err
	}
	itemDetails.Additions = selectedAdditions
	itemDetails.ApplyFixedAdditionsLimits()

	itemModel := models.Item{}
	itemModel.BuildToModel(item, resp.SecureURL, publicID, sha256Hash(imageBytes))
	itemModel.Details = itemDetails
	itemModel.HasFlavors = itemDetails.HasFlavors()
	itemModel.IsAvailable = models.SyncIsAvailableFlag(
		models.EffectiveItemAvailability(itemModel.AvailabilityStatus, itemDetails),
	)

	return p.repo.RegisterItem(ctx, &itemModel)
}

func (p *productsApp) buildItemDetails(
	ctx context.Context,
	categoryID uuid.UUID,
	flavors models.FlavorEntries,
	minSelection, maxSelection *int,
	current models.ItemDetails,
) (models.ItemDetails, error) {
	details := current
	details.Flavors = flavors

	if len(flavors) == 0 {
		details.MinSelection = 0
		details.MaxSelection = 0
		return details, nil
	}

	category, err := p.repo.SearchCategoryByID(ctx, categoryID)
	if err != nil {
		return models.ItemDetails{}, err
	}

	minSel, maxSel, err := resolveSelectionLimits(
		category.Details.SelectorType,
		minSelection,
		maxSelection,
		len(flavors),
	)
	if err != nil {
		return models.ItemDetails{}, err
	}
	details.MinSelection = minSel
	details.MaxSelection = maxSel

	if err := validateSelectionLimits(category.Details.SelectorType, details.MinSelection, details.MaxSelection, len(flavors)); err != nil {
		return models.ItemDetails{}, err
	}

	return details, nil
}

func resolveSelectionLimits(
	selectorType string,
	minSelection, maxSelection *int,
	flavorCount int,
) (int, int, error) {
	if !models.SupportsFlavorSelection(selectorType) {
		return 0, 0, errcustom.ErrCategoryWithoutFlavors
	}

	// Defaults: al menos 1 sabor y como máximo todos los asignados al item.
	// En single_select eso es "sabores distintos"; en multi_select, el tope
	// de unidades lo define el front con max_selection si lo envía.
	minSel := 1
	maxSel := flavorCount
	if minSelection != nil {
		minSel = *minSelection
	}
	if maxSelection != nil {
		maxSel = *maxSelection
	}

	return minSel, maxSel, nil
}

func validateSelectionLimits(selectorType string, minSelection, maxSelection, flavorCount int) error {
	if !models.SupportsFlavorSelection(selectorType) {
		return errcustom.ErrCategoryWithoutFlavors
	}
	if minSelection < 0 || maxSelection < 0 || minSelection > maxSelection {
		return errcustom.ErrInvalidSelection
	}
	// single_select: min/max cuentan sabores distintos → no pueden superar el catálogo del item.
	if selectorType == models.SelectorSingleSelect && flavorCount > 0 {
		if maxSelection > flavorCount || minSelection > flavorCount {
			return errcustom.ErrInvalidSelection
		}
	}
	// multi_select: min/max son tope de unidades (puede repetir el mismo sabor),
	// así que max puede ser mayor que la cantidad de sabores distintos.
	return nil
}

// resolveItemFlavors valida que los IDs pertenezcan al catálogo de la categoría
// y devuelve las entradas embebidas que se persistirán en items.details.
func (p *productsApp) resolveItemFlavors(ctx context.Context, categoryID uuid.UUID, flavorIDs []uuid.UUID) (models.FlavorEntries, error) {
	if len(flavorIDs) == 0 {
		return nil, nil
	}

	category, err := p.repo.SearchCategoryByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	if !models.SupportsFlavorSelection(category.Details.SelectorType) {
		return nil, errcustom.ErrCategoryWithoutFlavors
	}

	selected := category.Details.Flavors.FilterByIDs(flavorIDs)
	if len(selected) != len(flavorIDs) {
		return nil, errcustom.ErrInvalidFlavors
	}

	return selected, nil
}

func (p *productsApp) resolveItemAdditions(ctx context.Context, shopID uuid.UUID, additionIDs []uuid.UUID) (models.AdditionEntries, error) {
	if len(additionIDs) == 0 {
		return models.AdditionEntries{}, nil
	}

	items, err := p.repo.SearchAdditionItemsByIDs(ctx, shopID, additionIDs)
	if err != nil {
		return nil, err
	}
	if len(items) != len(additionIDs) {
		return nil, errcustom.ErrInvalidAdditions
	}

	byID := make(map[uuid.UUID]models.Item, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	selected := make(models.AdditionEntries, 0, len(additionIDs))
	for _, id := range additionIDs {
		item, ok := byID[id]
		if !ok {
			return nil, errcustom.ErrInvalidAdditions
		}
		selected = append(selected, models.AdditionEntryFromItem(item))
	}
	return selected, nil
}

func (p *productsApp) UpdateItem(ctx context.Context, item dtos.UpdateItem, itemID, shopID uuid.UUID) error {

	itemModel, err := p.repo.SearchItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	if itemModel == nil {
		return errcustom.ErrNotFound
	}

	if itemModel.ShopID != shopID {
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

	// has_flavors y details: derivar según campos enviados
	hasFlavors := itemModel.HasFlavors
	itemDetails := itemModel.Details
	itemFlavors := itemModel.Details.Flavors

	if item.FlavorIDs != nil {
		selectedFlavors, err := p.resolveItemFlavors(ctx, itemModel.CategoryID, *item.FlavorIDs)
		if err != nil {
			return err
		}
		itemFlavors = selectedFlavors
	}

	if item.FlavorIDs != nil || item.MinSelection != nil || item.MaxSelection != nil {
		builtDetails, err := p.buildItemDetails(
			ctx,
			itemModel.CategoryID,
			itemFlavors,
			item.MinSelection,
			item.MaxSelection,
			itemDetails,
		)
		if err != nil {
			return err
		}
		itemDetails = builtDetails
		hasFlavors = itemDetails.HasFlavors()
	}

	if item.AdditionIDs != nil {
		selectedAdditions, err := p.resolveItemAdditions(ctx, itemModel.ShopID, *item.AdditionIDs)
		if err != nil {
			return err
		}
		itemDetails.Additions = selectedAdditions
		itemDetails.ApplyFixedAdditionsLimits()
	}

	var ItemUpdateModel models.UpdateItem
	status := itemModel.AvailabilityStatus
	if item.AvailabilityStatus != nil {
		status = models.NormalizeAvailabilityStatus(*item.AvailabilityStatus)
	}
	ItemUpdateModel.BuildToModel(item, itemModel.ID, hasFlavors, itemDetails, status, imageURL, imagePublicID, imageHash)

	if err := p.repo.UpdateItem(ctx, &ItemUpdateModel, itemModel.ShopID); err != nil {
		return err
	}

	isAddition, err := p.repo.IsAdditionItem(ctx, itemModel.ID)
	if err != nil {
		return err
	}
	if isAddition {
		entry := models.AdditionEntryFromItem(models.Item{
			ID:                 itemModel.ID,
			Name:               ItemUpdateModel.Name,
			Price:              ItemUpdateModel.Price,
			AvailabilityStatus: ItemUpdateModel.AvailabilityStatus,
			Details:            ItemUpdateModel.Details,
		})
		if err := p.repo.SyncAdditionSnapshots(ctx, itemModel.ShopID, entry); err != nil {
			return err
		}
	}

	return nil
}

func (p *productsApp) SearchProductsByShopID(ctx context.Context, shopID uuid.UUID, forCustomer bool) (dtos.CategoriesResponse, error) {
	data, err := p.repo.SearchProductsByShopID(ctx, shopID, forCustomer)
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
	data, err := p.repo.SearchProductsByShopID(ctx, shopID, false)
	if err != nil {
		return nil, err
	}
	applyRateToCategories(p.bcvRate(ctx), &data)
	return data, nil
}

func (p *productsApp) SearchAdditionsByItemID(ctx context.Context, itemID uuid.UUID) (dtos.AdditionsResponse, error) {
	data, err := p.repo.SearchAdditionsByItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	applyRateToAdditions(p.bcvRate(ctx), &data)
	return data, nil
}

func (p *productsApp) SearchAdditionsCatalogByShopID(ctx context.Context, shopID uuid.UUID) (dtos.AdditionsResponse, error) {
	data, err := p.repo.SearchAdditionsCatalogByShopID(ctx, shopID)
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
	details := models.DefaultCategoryDetails()
	selectorType := models.NormalizeSelectorType(category.SelectorType)
	if !models.IsValidSelectorType(selectorType) {
		return errcustom.ErrInvalidSelection
	}
	details.SelectorType = selectorType

	if len(category.Flavors) > 0 {
		if !models.SupportsFlavorSelection(selectorType) {
			return errcustom.ErrCategoryWithoutFlavors
		}
		flavors := make(models.FlavorEntries, 0, len(category.Flavors))
		seen := make(map[string]struct{}, len(category.Flavors))
		for _, input := range category.Flavors {
			if _, ok := seen[input.Name]; ok {
				return errcustom.ErrInvalidSelection
			}
			seen[input.Name] = struct{}{}
			flavors = append(flavors, models.FlavorEntry{
				ID:          uuid.New(),
				Name:        input.Name,
				Description: input.Description,
				Available:   models.BoolPtr(true),
			})
		}
		details.Flavors = flavors
	}

	macroCategoryID := category.MacroCategoryID
	categoryModel := models.Category{
		ShopID:             category.ShopID,
		Name:               category.Name,
		MacroCategoryID:    &macroCategoryID,
		AvailabilityStatus: models.AvailabilityActive,
		Details:            details,
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

func (p *productsApp) UpdateCategory(ctx context.Context, category dtos.UpdateCategory, categoryID, shopID uuid.UUID) error {
	categoryModel, err := p.repo.SearchCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}

	if categoryModel == nil {
		return errcustom.ErrNotFound
	}

	if categoryModel.ShopID != shopID {
		return errcustom.ErrUnautorized
	}

	details := categoryModel.Details
	if category.SelectorType != nil {
		selectorType := models.NormalizeSelectorType(*category.SelectorType)
		if !models.IsValidSelectorType(selectorType) {
			return errcustom.ErrInvalidSelection
		}
		if selectorType == models.SelectorNone && len(categoryModel.Details.Flavors) > 0 {
			return errcustom.ErrCategoryHasFlavors
		}
		details.SelectorType = selectorType
	}

	var updateModel models.UpdateCategory
	updateModel.BuildToModel(category, categoryID, details, categoryModel.AvailabilityStatus)

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

	if !models.SupportsFlavorSelection(categoryModel.Details.SelectorType) {
		return errcustom.ErrCategoryWithoutFlavors
	}

	if categoryModel.Details.Flavors.HasName(flavor.Name, uuid.Nil) {
		return errcustom.ErrUnexpectedError
	}

	catalogFlavor := models.BuildCatalogFlavorFromDTO(flavor)
	return p.repo.RegisterFlavor(ctx, catalogFlavor)
}

func (p *productsApp) UpdateFlavor(ctx context.Context, flavor dtos.UpdateFlavor, flavorID, shopID uuid.UUID) error {
	flavorModel, err := p.repo.SearchFlavorByID(ctx, flavorID)
	if err != nil {
		return err
	}

	if flavorModel == nil {
		return errcustom.ErrNotFound
	}

	if flavorModel.ShopID != shopID {
		return errcustom.ErrUnautorized
	}

	category, err := p.repo.SearchCategoryByID(ctx, flavorModel.CategoryID)
	if err != nil {
		return err
	}

	if category.Details.Flavors.HasName(flavor.Name, flavorID) {
		return errcustom.ErrUnexpectedError
	}

	entry := flavorModel.Entry
	entry.Name = flavor.Name
	if flavor.Description != nil {
		entry.Description = *flavor.Description
	} else {
		entry.Description = ""
	}
	if flavor.Available != nil {
		entry.Available = flavor.Available
	}

	return p.repo.UpdateFlavor(ctx, flavorID, entry, flavorModel.ShopID, flavorModel.CategoryID)
}

func (p *productsApp) DeleteFlavor(ctx context.Context, flavorID, shopID uuid.UUID) error {
	flavorModel, err := p.repo.SearchFlavorByID(ctx, flavorID)
	if err != nil {
		return err
	}

	if flavorModel == nil {
		return errcustom.ErrNotFound
	}

	if flavorModel.ShopID != shopID {
		return errcustom.ErrUnautorized
	}

	return p.repo.DeleteFlavor(ctx, *flavorModel)
}

func (p *productsApp) SearchMacroCategories(ctx context.Context, shopType string) (dtos.MacroCategoriesResponse, error) {
	if shopType != "" && !shopsmodels.IsValidShopType(shopType) {
		return nil, errcustom.ErrInvalidShopType
	}
	macroCategories, err := p.repo.SearchMacroCategories(ctx, shopType)
	if err != nil {
		return nil, err
	}
	return macroCategories.ToDomainDTO(), nil
}

func (p *productsApp) SeedDefaultCategories(ctx context.Context, shopID uuid.UUID, shopType string) error {
	micros, err := p.repo.SearchMicroCategories(ctx, shopType)
	if err != nil {
		return err
	}
	for i := range micros {
		macroID := micros[i].MacroCategoryID
		category := &models.Category{
			ShopID:             shopID,
			Name:               micros[i].Name,
			MacroCategoryID:    &macroID,
			AvailabilityStatus: models.AvailabilityActive,
			Details:            models.DefaultCategoryDetails(),
		}
		if err := p.repo.RegisterCategory(ctx, category); err != nil {
			return err
		}
	}
	return nil
}

// validateDiscountValue asegura que el valor del descuento tenga sentido
// según su tipo: porcentaje entre 1 y 99, o monto fijo positivo y menor al
// precio del producto (nunca puede dejarlo en 0 o negativo).
func validateDiscountValue(discountType string, value float64, itemPrice int) error {
	switch discountType {
	case models.DiscountTypePercentage:
		if value < 1 || value > 99 {
			return errcustom.ErrInvalidDiscountPercent
		}
	case models.DiscountTypeFixed:
		if value <= 0 {
			return errcustom.ErrInvalidDiscountFixed
		}
		if value >= float64(itemPrice) {
			return errcustom.ErrInvalidDiscountFixed
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

func (p *productsApp) UpdateItemDiscount(ctx context.Context, discount dtos.UpdateItemDiscount, discountID, shopID uuid.UUID) error {
	discountModel, err := p.repo.SearchItemDiscountByID(ctx, discountID)
	if err != nil {
		return err
	}

	if discountModel == nil {
		return errcustom.ErrNotFound
	}

	if discountModel.ShopID != shopID {
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

func (p *productsApp) DeleteItemDiscount(ctx context.Context, discountID, shopID uuid.UUID) error {
	discountModel, err := p.repo.SearchItemDiscountByID(ctx, discountID)
	if err != nil {
		return err
	}

	if discountModel == nil {
		return errcustom.ErrNotFound
	}

	if discountModel.ShopID != shopID {
		return errcustom.ErrUnautorized
	}

	return p.repo.DeleteItemDiscount(ctx, discountModel)
}

func (p *productsApp) DeleteItemDiscountByItemID(ctx context.Context, itemID, shopID uuid.UUID) error {
	itemModel, err := p.repo.SearchItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	if itemModel == nil {
		return errcustom.ErrNotFound
	}

	if itemModel.ShopID != shopID {
		return errcustom.ErrUnautorized
	}

	discountModel, err := p.repo.SearchActiveItemDiscountByItemID(ctx, itemID)
	if err != nil {
		return err
	}

	if discountModel == nil {
		return errcustom.ErrNotFound
	}

	return p.repo.DeleteItemDiscount(ctx, discountModel)
}

func (p *productsApp) SearchCategoriesByShopID(ctx context.Context, shopID uuid.UUID) (dtos.CategoriesResponse, error) {
	categories, err := p.repo.SearchCategoriesByShopID(ctx, shopID)
	if err != nil {
		return nil, err
	}
	return categories.ToOwnerCatalogDTO(), nil
}

func (p *productsApp) SearchCategorySummariesByShopID(ctx context.Context, shopID uuid.UUID, filter string) (dtos.CategorySummariesResponse, error) {
	categories, err := p.repo.SearchCategorySummariesByShopID(ctx, shopID, filter)
	if err != nil {
		return nil, err
	}
	return categories.ToSummaryDTO(), nil
}
