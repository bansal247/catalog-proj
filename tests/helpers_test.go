package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	domainsvc "github.com/bansal247/catalog-proj/internal/app/product/domain/services"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/get_product"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/list_products"
	"github.com/bansal247/catalog-proj/internal/app/product/repo"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/activate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/apply_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/create_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/deactivate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/remove_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/update_product"
	"github.com/bansal247/catalog-proj/internal/pkg/clock"
	spannerdriver "github.com/bansal247/catalog-proj/internal/pkg/committer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	testProject  = "test-project"
	testInstance = "test-instance"
	testDatabase = "catalog-test"
)

// testEnv holds all usecase dependencies for E2E tests.
type testEnv struct {
	client *spanner.Client

	createProduct    *create_product.Interactor
	updateProduct    *update_product.Interactor
	activateProduct  *activate_product.Interactor
	deactivateProduct *deactivate_product.Interactor
	applyDiscount    *apply_discount.Interactor
	removeDiscount   *remove_discount.Interactor
	getProduct       *get_product.Query
	listProducts     *list_products.Query
}

// setupTestEnv creates a fresh Spanner emulator environment for a test.
// It provisions the instance, database, and schema, then wires all use cases.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	emulatorHost := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorHost == "" {
		emulatorHost = "localhost:9010"
	}

	ctx := context.Background()
	opts := []option.ClientOption{
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	// Create instance
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, opts...)
	require.NoError(t, err, "create instance admin client")
	defer instanceAdmin.Close()

	instancePath := fmt.Sprintf("projects/%s/instances/%s", testProject, testInstance)
	_, err = instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", testProject),
		InstanceId: testInstance,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", testProject),
			DisplayName: "test",
			NodeCount:   1,
		},
	})
	// Ignore already-exists errors
	if err != nil && !isAlreadyExists(err) {
		require.NoError(t, err, "create instance")
	}

	// Create database with schema — use unique name per test to allow parallelism
	dbName := fmt.Sprintf("catalog-test-%s", uuid.New().String()[:8])
	dbPath := fmt.Sprintf("%s/databases/%s", instancePath, dbName)

	dbAdmin, err := database.NewDatabaseAdminClient(ctx, opts...)
	require.NoError(t, err, "create db admin client")
	defer dbAdmin.Close()

	createOp, err := dbAdmin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          instancePath,
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", dbName),
		ExtraStatements: schemaDDL(),
	})
	require.NoError(t, err, "create database")
	_, err = createOp.Wait(ctx)
	require.NoError(t, err, "wait for database creation")

	// Create Spanner client
	client, err := spanner.NewClient(ctx, dbPath, opts...)
	require.NoError(t, err, "create spanner client")

	t.Cleanup(func() {
		client.Close()
		// Best-effort database drop
		_ = dbAdmin.DropDatabase(ctx, &databasepb.DropDatabaseRequest{Database: dbPath})
	})

	// Wire use cases
	clk := clock.Real{}
	committer := spannerdriver.New(client)
	pricingSvc := domainsvc.NewPricingCalculator()
	productRepo := repo.NewProductRepo(client)
	outboxRepo := repo.NewOutboxRepo()
	readModel := repo.NewSpannerReadModel(client, pricingSvc)

	return &testEnv{
		client:            client,
		createProduct:     create_product.New(productRepo, outboxRepo, *committer, clk),
		updateProduct:     update_product.New(productRepo, outboxRepo, *committer, clk),
		activateProduct:   activate_product.New(productRepo, outboxRepo, *committer, clk),
		deactivateProduct: deactivate_product.New(productRepo, outboxRepo, *committer, clk),
		applyDiscount:     apply_discount.New(productRepo, outboxRepo, *committer, clk),
		removeDiscount:    remove_discount.New(productRepo, outboxRepo, *committer, clk),
		getProduct:        get_product.New(readModel),
		listProducts:      list_products.New(readModel),
	}
}

// getOutboxEvents fetches all outbox events for a given aggregate ID.
func getOutboxEvents(t *testing.T, client *spanner.Client, aggregateID string) []outboxRow {
	t.Helper()
	ctx := context.Background()

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT event_id, event_type, aggregate_id, status
		      FROM outbox_events WHERE aggregate_id = @id ORDER BY created_at`,
		Params: map[string]interface{}{"id": aggregateID},
	})
	defer iter.Stop()

	var rows []outboxRow
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var r outboxRow
		_ = row.Columns(&r.EventID, &r.EventType, &r.AggregateID, &r.Status)
		rows = append(rows, r)
	}
	return rows
}

type outboxRow struct {
	EventID     string
	EventType   string
	AggregateID string
	Status      string
}

// schemaDDL returns the DDL statements for the test database.
func schemaDDL() []string {
	return []string{
		`CREATE TABLE products (
			product_id            STRING(36)   NOT NULL,
			name                  STRING(255)  NOT NULL,
			description           STRING(MAX),
			category              STRING(100)  NOT NULL,
			base_price_numerator  INT64        NOT NULL,
			base_price_denominator INT64       NOT NULL,
			discount_percent      FLOAT64,
			discount_start_date   TIMESTAMP,
			discount_end_date     TIMESTAMP,
			status                STRING(20)   NOT NULL,
			created_at            TIMESTAMP    NOT NULL,
			updated_at            TIMESTAMP    NOT NULL,
			archived_at           TIMESTAMP
		) PRIMARY KEY (product_id)`,
		`CREATE INDEX idx_products_category ON products (category, status)`,
		`CREATE TABLE outbox_events (
			event_id     STRING(36)  NOT NULL,
			event_type   STRING(100) NOT NULL,
			aggregate_id STRING(36)  NOT NULL,
			payload      JSON        NOT NULL,
			status       STRING(20)  NOT NULL,
			created_at   TIMESTAMP   NOT NULL,
			processed_at TIMESTAMP
		) PRIMARY KEY (event_id)`,
		`CREATE INDEX idx_outbox_status ON outbox_events (status, created_at)`,
	}
}

func isAlreadyExists(err error) bool {
	return err != nil && (contains(err.Error(), "AlreadyExists") || contains(err.Error(), "already exists"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
