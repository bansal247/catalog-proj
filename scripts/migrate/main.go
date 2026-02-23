// scripts/migrate/main.go applies the Spanner DDL schema to the emulator.
// Run via: go run ./scripts/migrate/main.go -project=... -instance=... -database=...
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	project := flag.String("project", "test-project", "GCP project ID")
	inst := flag.String("instance", "test-instance", "Spanner instance ID")
	db := flag.String("database", "catalog", "Spanner database ID")
	flag.Parse()

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

	// 1. Ensure instance exists
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, opts...)
	if err != nil {
		log.Fatalf("instance admin: %v", err)
	}
	defer instanceAdmin.Close()

	instancePath := fmt.Sprintf("projects/%s/instances/%s", *project, *inst)
	_, err = instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", *project),
		InstanceId: *inst,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", *project),
			DisplayName: "catalog",
			NodeCount:   1,
		},
	})
	if err != nil && !isAlreadyExists(err) {
		log.Fatalf("create instance: %v", err)
	}
	log.Printf("Instance ready: %s", instancePath)

	// 2. Create database + schema
	dbAdmin, err := database.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		log.Fatalf("db admin: %v", err)
	}
	defer dbAdmin.Close()

	dbPath := fmt.Sprintf("%s/databases/%s", instancePath, *db)
	createOp, err := dbAdmin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          instancePath,
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", *db),
		ExtraStatements: schemaDDL(),
	})
	if err != nil {
		if isAlreadyExists(err) {
			log.Printf("Database already exists: %s — applying DDL updates", dbPath)
			updateOp, err := dbAdmin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbPath,
				Statements: schemaDDL(),
			})
			if err != nil {
				log.Fatalf("update DDL: %v", err)
			}
			if err := updateOp.Wait(ctx); err != nil {
				log.Printf("DDL update warning (may be idempotent): %v", err)
			}
		} else {
			log.Fatalf("create database: %v", err)
		}
	} else {
		if _, err := createOp.Wait(ctx); err != nil {
			log.Fatalf("wait create: %v", err)
		}
	}
	log.Printf("Database ready: %s", dbPath)
}

func schemaDDL() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS products (
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
		`CREATE INDEX IF NOT EXISTS idx_products_category ON products (category, status)`,
		`CREATE TABLE IF NOT EXISTS outbox_events (
			event_id     STRING(36)  NOT NULL,
			event_type   STRING(100) NOT NULL,
			aggregate_id STRING(36)  NOT NULL,
			payload      JSON        NOT NULL,
			status       STRING(20)  NOT NULL,
			created_at   TIMESTAMP   NOT NULL,
			processed_at TIMESTAMP
		) PRIMARY KEY (event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox_events (status, created_at)`,
	}
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "AlreadyExists") || contains(s, "already exists")
}

func contains(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
