package e2e_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"Encargalo.app-api.go/api/router/groups"
	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	ordersapp "Encargalo.app-api.go/internal/orders/app"
	ordersmodels "Encargalo.app-api.go/internal/orders/domain/models"
	ordersports "Encargalo.app-api.go/internal/orders/domain/ports"
	ordersrepo "Encargalo.app-api.go/internal/orders/repo"
	paymentsapp "Encargalo.app-api.go/internal/payments/app"
	paymentsmodels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsports "Encargalo.app-api.go/internal/payments/domain/ports"
	paymentshandler "Encargalo.app-api.go/internal/payments/handler"
	paymentsrepo "Encargalo.app-api.go/internal/payments/repo"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	tripsapp "Encargalo.app-api.go/internal/trips/app"
	tripsports "Encargalo.app-api.go/internal/trips/domain/ports"
	tripshandler "Encargalo.app-api.go/internal/trips/handler"
	tripsrepo "Encargalo.app-api.go/internal/trips/repo"
	usersapp "Encargalo.app-api.go/internal/users/app"
	usersmodels "Encargalo.app-api.go/internal/users/domain/models"
	usersports "Encargalo.app-api.go/internal/users/domain/ports"
	usershandler "Encargalo.app-api.go/internal/users/handler"
	usersrepo "Encargalo.app-api.go/internal/users/repo"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	bcvRate       = 40.0
	orderTotalUSD = 10.0
	proofURL      = "https://res.cloudinary.com/encargalo/image/upload/proof.jpg"
)

var (
	db          *bun.DB
	sqlDB       *sql.DB
	baseDSN     string
	httpServer  *httptest.Server
	staffID     uuid.UUID
	shopID      uuid.UUID
	ownerID     uuid.UUID
	reviewsApp  usersports.CustomerFinanceApp
	reviewsPort paymentsports.PaymentReviewsApp
	reviewsRepo paymentsports.PaymentReviewsRepo
	ordersRepo  ordersports.OrdersRepo
	phoneSeq    atomic.Int64
	useRemote   bool
)

func TestMain(m *testing.M) {
	code := run(m)
	os.Exit(code)
}

func run(m *testing.M) int {
	if os.Getenv("E2E_REMOTE") == "1" {
		return runRemote(m)
	}
	return runDocker(m)
}

func runRemote(m *testing.M) int {
	useRemote = true
	if err := loadProjectEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: .env: %v\n", err)
		return 1
	}
	config.Environments()
	phoneSeq.Store(time.Now().UnixNano() % 10000000)

	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	name := os.Getenv("POSTGRES_DATABASE")
	if name == "" {
		name = os.Getenv("POSTGRES_DB")
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + name,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	baseDSN = u.String()

	sqlDB = openSQL(baseDSN)
	db = bun.NewDB(sqlDB, pgdialect.New())
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: no se pudo conectar al Postgres remoto: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "e2e: usando Postgres remoto %s/%s\n", host, name)
	return finish(m)
}

func runDocker(m *testing.M) int {
	loadTestConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("encargalo"),
		postgres.WithUsername("encargalo"),
		postgres.WithPassword("encargalo"),
		postgres.BasicWaitStrategies(),
		testcontainers.WithCmd("postgres", "-c", "shared_preload_libraries=pg_stat_statements"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: no se pudo levantar Postgres (hace falta Docker): %v\n", err)
		return 1
	}
	defer func() {
		downCtx, downCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer downCancel()
		_ = pg.Terminate(downCtx)
	}()

	baseDSN, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: connection string: %v\n", err)
		return 1
	}

	sqlDB = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(baseDSN)))
	db = bun.NewDB(sqlDB, pgdialect.New())
	defer func() { _ = db.Close() }()

	return finish(m)
}

func finish(m *testing.M) int {
	if err := migrateUp(sqlDB); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: goose up: %v\n", err)
		return 1
	}

	if err := seedBase(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: seed: %v\n", err)
		return 1
	}
	wire()

	e := echo.New()
	e.HTTPErrorHandler = errcustom.HTTPErrorHandler
	groups.NewPaymentsGroup(staffSessionAuth{staffID: staffID}, paymentshandler.NewStaffReviewsHandler(hub, reviewsPort)).Resource(e)
	mountTripPaymentHTTP(e)
	httpServer = httptest.NewServer(e)
	defer httpServer.Close()

	return m.Run()
}

func loadProjectEnv() error {
	root, err := findModuleRoot()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// hub is set in wire and used by the HTTP server.
var hub *paymentsapp.StaffReviewHubImpl

func loadTestConfig() {
	envs := map[string]string{
		"SERVER_PORT":              "3000",
		"POSTGRES_HOST":            "localhost",
		"POSTGRES_PORT":            "5432",
		"POSTGRES_USER":            "test",
		"POSTGRES_PASSWORD":        "test",
		"POSTGRES_DATABASE":        "test",
		"REDIS_URL":                "redis://localhost:6379",
		"REDIS_STREAM_STREAM_NAME": "test-stream",
		"JWT_SECRET":               "test-secret",
		"ENVIRONMENT_CURRENT":      "test",
		"GOOGLE_SERVER_KEY":        "test",
		"GOOGLE_CLIENT_ID":         "test",
		"CLOUDINARY_URL":           "cloudinary://test",
		"CLOUDINARY_FOLDER":        "test",
		"EXTERNAL_API_KEY":         "test",
		"INTERNAL_GATEWAY_SECRET":  "test",
	}
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	config.Environments()
}

func migrateUp(instance *sql.DB) error {
	root, err := findModuleRoot()
	if err != nil {
		return err
	}
	goose.SetBaseFS(os.DirFS(filepath.Join(root, "internal/shared/adapters/postgres")))
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(instance, "migrations")
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func seedBase(ctx context.Context) error {
	licenseID := uuid.MustParse("6e02bcef-0372-4e80-ad1b-ed5d348dde0e")
	if _, err := db.ExecContext(ctx, `
		INSERT INTO licenses.licenses (id, name, price)
		VALUES (?, 'e2e', 0)
		ON CONFLICT (id) DO NOTHING
	`, licenseID); err != nil {
		return fmt.Errorf("license: %w", err)
	}

	ownerID = uuid.New()
	staffID = uuid.New()
	if err := insertAccount(ctx, ownerID, "Owner"); err != nil {
		return err
	}
	if err := insertAccount(ctx, staffID, "Staff"); err != nil {
		return err
	}

	shopID = uuid.New()
	shop := &shopmodels.Shop{
		ID:        shopID,
		OwnerID:   ownerID,
		LicenseID: licenseID,
		Name:      "Local e2e",
		Type:      "Restaurant",
		HomePhone: nextPhone(),
		Status:    shopmodels.ShopStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if _, err := db.NewInsert().Model(shop).Exec(ctx); err != nil {
		return fmt.Errorf("shop: %w", err)
	}
	return nil
}

func insertAccount(ctx context.Context, id uuid.UUID, name string) error {
	phone := nextPhone()
	acc := &usersmodels.Account{
		ID:       id,
		Name:     name,
		Phone:    &phone,
		Password: "secret",
	}
	if _, err := db.NewInsert().Model(acc).Exec(ctx); err != nil {
		return fmt.Errorf("account %s: %w", name, err)
	}
	return nil
}

func nextPhone() string {
	return fmt.Sprintf("0414%07d", phoneSeq.Add(1))
}

func wire() {
	reviewsRepo = paymentsrepo.NewPaymentReviewsRepo(db)
	ordersRepo = ordersrepo.NewOrdersRepo(db)
	financeRepo := usersrepo.NewCustomerFinanceRepo(db)
	reviewsApp = usersapp.NewCustomerFinanceApp(financeRepo, nil, nil, nil, nil)
	creditor := paymentsports.CustomerBalanceCreditor(reviewsApp)

	registry := paymentsapp.NewSubjectRegistry()
	subject := ordersapp.NewOrderPaymentSubject(ordersRepo, reviewsRepo, creditor, nil)
	registry.Register(paymentsmodels.ReferenceTypeOrder, subject)
	tripsRepo := tripsrepo.NewTripsRepo(db)
	registry.Register(paymentsmodels.ReferenceTypeTrip, tripsapp.NewTripPaymentSubject(tripsRepo, nil, nil, creditor, reviewsRepo))

	hub = paymentsapp.NewStaffReviewHub(reviewsRepo)
	reviewsPort = paymentsapp.NewPaymentReviewsApp(reviewsRepo, creditor, hub, registry)
	reviewsPort.SetSnapshotProvider(ordersapp.NewOrderStaffSnapshot(ordersRepo, reviewsRepo))
	hub.SetSnapshotProvider(reviewsPort.ListActiveForStaff)
	hub.SetSnapshotProvider(tripsapp.NewTripStaffSnapshot(tripsRepo, reviewsRepo))

	quoteStore = newMemQuotes()
	tripsHTTP = tripsapp.NewTripsApp(tripsRepo, nil, fixedBCV{}, quoteStore, fakeUpload{}, reviewsPort, hub, nil, reviewsApp, nil, nil, nil, nil, nil, nil, nil)
}

// testCustomerHeader sustituye la sesión que Traefik inyecta. El handler es el de producción.
const testCustomerHeader = "X-Test-Customer-Id"

var (
	quoteStore *memQuotes
	tripsHTTP  tripsports.TripsApp
)

type headerCustomerAuth struct{}

func (headerCustomerAuth) Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := uuid.Parse(c.Request().Header.Get(testCustomerHeader))
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "no autorizado")
		}
		session := authmodels.ActiveSession{UserID: id, Roles: []authmodels.Role{authmodels.Customer}}
		ctx := context.WithValue(c.Request().Context(), authmodels.SessionKey, session)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

func (headerCustomerAuth) Require(...authmodels.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
}

func mountTripPaymentHTTP(e *echo.Echo) {
	authn := headerCustomerAuth{}
	trips := tripshandler.NewTripsHandler(tripsHTTP, nil)
	tg := e.Group("/trips")
	tg.POST("", trips.RequestTrip, middleware.BodyLimit("8M"), authn.Auth, authn.Require(authmodels.Customer))
	tg.GET("/:id", trips.GetTrip, authn.Auth, authn.Require(authmodels.Customer))
	tg.POST("/:id/cancel", trips.CancelTrip, authn.Auth, authn.Require(authmodels.Customer))

	finance := usershandler.NewCustomerFinanceHandler(reviewsApp)
	ug := e.Group("/users")
	ug.GET("/balance", finance.GetBalance, authn.Auth, authn.Require(authmodels.Customer))
	ug.PUT("/pago-movil", finance.UpsertPagoMovil, authn.Auth, authn.Require(authmodels.Customer))
	ug.POST("/balance/refund-requests", finance.CreateRefundRequest, authn.Auth, authn.Require(authmodels.Customer))
}

type staffSessionAuth struct {
	staffID uuid.UUID
}

func (a staffSessionAuth) Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session := authmodels.ActiveSession{UserID: a.staffID, Roles: []authmodels.Role{authmodels.Staff}}
		ctx := context.WithValue(c.Request().Context(), authmodels.SessionKey, session)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

func (staffSessionAuth) Require(...authmodels.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
}

func newCustomer(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, insertAccount(context.Background(), id, "Cliente"))
	return id
}

type seededOrder struct {
	Order    *ordersmodels.Order
	Review   *paymentsmodels.PaymentReview
	Expected float64
}

func seedPendingOrder(t *testing.T, customerID uuid.UUID, balanceUsedBs, pmBs float64) seededOrder {
	t.Helper()
	expected := money.RoundBs(pmBs)
	if expected <= 0 {
		expected = money.RoundBs(orderTotalUSD * bcvRate)
		pmBs = expected
	}
	details := map[string]any{
		"customer": map[string]any{
			"id":        customerID.String(),
			"name":      "Ana",
			"last_name": "Perez",
			"phone":     "04120000000",
		},
		"payment": map[string]any{
			"balance_amount_bs":    balanceUsedBs,
			"pago_movil_amount_bs": pmBs,
			"bcv_rate":             bcvRate,
			"payment_proof_url":    proofURL,
		},
	}
	raw, err := json.Marshal(details)
	require.NoError(t, err)

	order := &ordersmodels.Order{
		ID:            uuid.New(),
		ShopID:        shopID,
		CustomerID:    customerID,
		Address:       "Calle 1",
		Latitude:      10.5,
		Longitude:     -66.9,
		MethodPayment: "PagoMovil",
		Status:        string(ordersmodels.StatusPaymentReview),
		DeliveryFee:   1,
		TotalPrice:    orderTotalUSD,
		Details:       raw,
		Number:        int(phoneSeq.Add(1)),
		CreatedAt:     time.Now(),
	}
	_, err = db.NewInsert().Model(order).Exec(context.Background())
	require.NoError(t, err)

	review, err := reviewsPort.CreateReview(context.Background(), paymentsmodels.ReferenceTypeOrder, order.ID, proofURL, expected, bcvRate)
	require.NoError(t, err)
	return seededOrder{Order: order, Review: review, Expected: expected}
}

func dialStaff(t *testing.T) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/payments/staff/reviews"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readUntil(t *testing.T, conn *websocket.Conn, timeout time.Duration, done func([]map[string]any) bool) []map[string]any {
	t.Helper()
	// Un timeout de lectura deja el websocket inutilizable. Una sola fecha
	// límite cubre toda la espera.
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(timeout)))
	var msgs []map[string]any
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("timeout esperando mensajes WS, recibidos: %#v (%v)", msgs, err)
		}
		var msg map[string]any
		require.NoError(t, json.Unmarshal(raw, &msg))
		msgs = append(msgs, msg)
		if done(msgs) {
			return msgs
		}
	}
}

func hasMessage(msgs []map[string]any, typ string, reviewID uuid.UUID) bool {
	return findMessage(msgs, typ, reviewID) != nil
}

func findMessage(msgs []map[string]any, typ string, reviewID uuid.UUID) map[string]any {
	for _, msg := range msgs {
		if msg["type"] != typ {
			continue
		}
		if reviewID != uuid.Nil && msg["review_id"] != reviewID.String() {
			continue
		}
		return msg
	}
	return nil
}

func freshLast4(t *testing.T) string {
	t.Helper()
	for range 40 {
		n := (time.Now().UnixNano() + phoneSeq.Add(1)) % 10000
		if n < 0 {
			n = -n
		}
		last4 := fmt.Sprintf("%04d", n)
		exists, err := reviewsRepo.ExistsResolvedReferenceLast4(context.Background(), last4)
		require.NoError(t, err)
		if !exists {
			return last4
		}
	}
	t.Fatal("no hay reference_last4 libre")
	return ""
}

func sendStaff(t *testing.T, conn *websocket.Conn, payload any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, raw))
}

func reloadOrder(t *testing.T, id uuid.UUID) *ordersmodels.Order {
	t.Helper()
	order, err := ordersRepo.SearchOrdersByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, order)
	return order
}

func paymentDetails(t *testing.T, order *ordersmodels.Order) ordersmodels.PaymentDetails {
	t.Helper()
	var details ordersmodels.Details
	require.NoError(t, json.Unmarshal(order.Details, &details))
	require.NotNil(t, details.PaymentDetails)
	return *details.PaymentDetails
}

func reloadReview(t *testing.T, id uuid.UUID) *paymentsmodels.PaymentReview {
	t.Helper()
	review, err := reviewsRepo.GetByID(context.Background(), id)
	require.NoError(t, err)
	return review
}

type ledgerRow struct {
	bun.BaseModel `bun:"table:finance.transactions"`

	ID            uuid.UUID  `bun:"id,pk"`
	ReferenceID   *uuid.UUID `bun:"reference_id"`
	ReferenceType *string    `bun:"reference_type"`
	OrderID       *uuid.UUID `bun:"order_id"`
	Amount        float64    `bun:"amount"`
	MovementType  string     `bun:"movement_type"`
	EntityID      uuid.UUID  `bun:"entity_id"`
}

func ledgerForReference(t *testing.T, referenceID uuid.UUID) []ledgerRow {
	t.Helper()
	var rows []ledgerRow
	err := db.NewSelect().Model(&rows).Where("reference_id = ?", referenceID).OrderExpr("created_at ASC").Scan(context.Background())
	require.NoError(t, err)
	return rows
}

func swapDatabase(dsn, name string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		panic(err)
	}
	u.Path = "/" + name
	return u.String()
}

func openSQL(dsn string) *sql.DB {
	return sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
}
