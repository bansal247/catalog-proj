package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bansal247/catalog-proj/internal/app/product/domain"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/activate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/apply_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/create_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/deactivate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/remove_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/update_product"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/list_products"
)

// TestProductCreationFlow verifies the happy-path creation flow:
// product is persisted and a ProductCreated outbox event is stored.
func TestProductCreationFlow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	productID, err := env.createProduct.Execute(ctx, create_product.Request{
		Name:                 "Test Widget",
		Description:          "A wonderful widget",
		Category:             "widgets",
		BasePriceNumerator:   1999,
		BasePriceDenominator: 100,
	})
	require.NoError(t, err)
	require.NotEmpty(t, productID)

	// Product must be retrievable.
	dto, err := env.getProduct.Execute(ctx, productID)
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, "Test Widget", dto.Name)
	assert.Equal(t, "widgets", dto.Category)
	assert.Equal(t, "A wonderful widget", dto.Description)
	assert.InEpsilon(t, 19.99, dto.BasePrice, 0.001)
	assert.Equal(t, "draft", dto.Status)

	// Effective price equals base price when no discount exists.
	assert.InEpsilon(t, dto.BasePrice, dto.EffectivePrice, 0.001)

	// Outbox must contain exactly one event.
	events := getOutboxEvents(t, env.client, productID)
	require.Len(t, events, 1)
	assert.Equal(t, "product.created", events[0].EventType)
	assert.Equal(t, "pending", events[0].Status)
}

// TestProductUpdateFlow verifies that updates persist and emit an event.
func TestProductUpdateFlow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	productID, err := env.createProduct.Execute(ctx, create_product.Request{
		Name:                 "Old Name",
		Category:             "electronics",
		BasePriceNumerator:   5000,
		BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = env.updateProduct.Execute(ctx, update_product.Request{
		ProductID:   productID,
		Name:        "New Name",
		Description: "Updated description",
		Category:    "gadgets",
	})
	require.NoError(t, err)

	dto, err := env.getProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", dto.Name)
	assert.Equal(t, "gadgets", dto.Category)
	assert.Equal(t, "Updated description", dto.Description)

	// Two events: created + updated.
	events := getOutboxEvents(t, env.client, productID)
	assert.Len(t, events, 2)
	assert.Equal(t, "product.updated", events[1].EventType)
}

// TestActivationDeactivationFlow tests the status state machine.
func TestActivationDeactivationFlow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	productID, err := env.createProduct.Execute(ctx, create_product.Request{
		Name:                 "Status Test",
		Category:             "test",
		BasePriceNumerator:   100,
		BasePriceDenominator: 1,
	})
	require.NoError(t, err)

	// Activate
	err = env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	dto, err := env.getProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "active", dto.Status)

	// Deactivate
	err = env.deactivateProduct.Execute(ctx, deactivate_product.Request{ProductID: productID})
	require.NoError(t, err)

	dto, err = env.getProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "inactive", dto.Status)

	// Four outbox events: created, activated, deactivated.
	events := getOutboxEvents(t, env.client, productID)
	assert.Len(t, events, 3)
	assert.Equal(t, "product.activated", events[1].EventType)
	assert.Equal(t, "product.deactivated", events[2].EventType)
}

// TestDiscountApplicationFlow verifies that discounts are persisted and effective
// price is computed correctly.
func TestDiscountApplicationFlow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create and activate a product at $100.00
	productID, err := env.createProduct.Execute(ctx, create_product.Request{
		Name:                 "Discounted Item",
		Category:             "sale",
		BasePriceNumerator:   10000,
		BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	// Apply 20% discount
	now := time.Now()
	err = env.applyDiscount.Execute(ctx, apply_discount.Request{
		ProductID:          productID,
		DiscountPercentage: 20.0,
		StartDate:          now.Add(-time.Hour),
		EndDate:            now.Add(24 * time.Hour),
	})
	require.NoError(t, err)

	dto, err := env.getProduct.Execute(ctx, productID)
	require.NoError(t, err)
	require.NotNil(t, dto.Discount)
	assert.InEpsilon(t, 100.0, dto.BasePrice, 0.001)
	// Effective price = 100 * (1 - 0.20) = $80.00
	assert.InEpsilon(t, 80.0, dto.EffectivePrice, 0.001)
	assert.InEpsilon(t, 20.0, dto.Discount.Percentage, 0.001)

	// Outbox: created + activated + discount_applied
	events := getOutboxEvents(t, env.client, productID)
	assert.Len(t, events, 3)
	assert.Equal(t, "product.discount_applied", events[2].EventType)
}

// TestRemoveDiscountFlow verifies that discounts can be removed.
func TestRemoveDiscountFlow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	productID, err := env.createProduct.Execute(ctx, create_product.Request{
		Name:                 "Remove Discount Test",
		Category:             "test",
		BasePriceNumerator:   5000,
		BasePriceDenominator: 100,
	})
	require.NoError(t, err)
	require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID}))

	now := time.Now()
	require.NoError(t, env.applyDiscount.Execute(ctx, apply_discount.Request{
		ProductID:          productID,
		DiscountPercentage: 10.0,
		StartDate:          now.Add(-time.Hour),
		EndDate:            now.Add(24 * time.Hour),
	}))

	// Remove the discount
	err = env.removeDiscount.Execute(ctx, remove_discount.Request{ProductID: productID})
	require.NoError(t, err)

	dto, err := env.getProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Nil(t, dto.Discount)
	// Effective price should revert to base price
	assert.InEpsilon(t, dto.BasePrice, dto.EffectivePrice, 0.001)
}

// TestBusinessRuleValidation verifies that domain rules are enforced.
func TestBusinessRuleValidation(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	t.Run("cannot apply discount to draft product", func(t *testing.T) {
		productID, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Draft Product",
			Category:             "test",
			BasePriceNumerator:   1000,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)

		now := time.Now()
		err = env.applyDiscount.Execute(ctx, apply_discount.Request{
			ProductID:          productID,
			DiscountPercentage: 10.0,
			StartDate:          now,
			EndDate:            now.Add(24 * time.Hour),
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, domain.ErrProductNotActive.Error())
	})

	t.Run("cannot apply discount to inactive product", func(t *testing.T) {
		productID, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Inactive Product",
			Category:             "test",
			BasePriceNumerator:   1000,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
		require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID}))
		require.NoError(t, env.deactivateProduct.Execute(ctx, deactivate_product.Request{ProductID: productID}))

		now := time.Now()
		err = env.applyDiscount.Execute(ctx, apply_discount.Request{
			ProductID:          productID,
			DiscountPercentage: 10.0,
			StartDate:          now,
			EndDate:            now.Add(24 * time.Hour),
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, domain.ErrProductNotActive.Error())
	})

	t.Run("cannot activate an already active product", func(t *testing.T) {
		productID, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Active Product",
			Category:             "test",
			BasePriceNumerator:   1000,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
		require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID}))

		err = env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
		require.Error(t, err)
		assert.ErrorContains(t, err, domain.ErrProductAlreadyActive.Error())
	})

	t.Run("cannot apply two concurrent discounts", func(t *testing.T) {
		productID, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Double Discount Test",
			Category:             "test",
			BasePriceNumerator:   1000,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
		require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID}))

		now := time.Now()
		require.NoError(t, env.applyDiscount.Execute(ctx, apply_discount.Request{
			ProductID:          productID,
			DiscountPercentage: 10.0,
			StartDate:          now.Add(-time.Hour),
			EndDate:            now.Add(24 * time.Hour),
		}))

		err = env.applyDiscount.Execute(ctx, apply_discount.Request{
			ProductID:          productID,
			DiscountPercentage: 20.0,
			StartDate:          now.Add(-time.Hour),
			EndDate:            now.Add(24 * time.Hour),
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, domain.ErrDiscountAlreadyActive.Error())
	})

	t.Run("cannot remove non-existent discount", func(t *testing.T) {
		productID, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "No Discount",
			Category:             "test",
			BasePriceNumerator:   1000,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)

		err = env.removeDiscount.Execute(ctx, remove_discount.Request{ProductID: productID})
		require.Error(t, err)
		assert.ErrorContains(t, err, domain.ErrNoActiveDiscount.Error())
	})

	t.Run("invalid discount period", func(t *testing.T) {
		productID, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Invalid Discount Period",
			Category:             "test",
			BasePriceNumerator:   1000,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
		require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: productID}))

		now := time.Now()
		// End before start → invalid
		err = env.applyDiscount.Execute(ctx, apply_discount.Request{
			ProductID:          productID,
			DiscountPercentage: 10.0,
			StartDate:          now.Add(24 * time.Hour),
			EndDate:            now.Add(time.Hour),
		})
		require.Error(t, err)
	})
}

// TestListProductsFlow verifies that listing and filtering work correctly.
func TestListProductsFlow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create products in two categories
	for i := 0; i < 3; i++ {
		id, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Widget",
			Category:             "widgets",
			BasePriceNumerator:   int64(1000 + i),
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
		require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: id}))
	}
	for i := 0; i < 2; i++ {
		id, err := env.createProduct.Execute(ctx, create_product.Request{
			Name:                 "Gadget",
			Category:             "gadgets",
			BasePriceNumerator:   int64(2000 + i),
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
		require.NoError(t, env.activateProduct.Execute(ctx, activate_product.Request{ProductID: id}))
	}

	// List all products (no filter)
	all, err := env.listProducts.Execute(ctx, list_products.ListProductsRequest{PageSize: 20})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all.Products), 5)

	// Filter by category
	widgets, err := env.listProducts.Execute(ctx, list_products.ListProductsRequest{Category: "widgets", PageSize: 20})
	require.NoError(t, err)
	assert.Len(t, widgets.Products, 3)
	for _, p := range widgets.Products {
		assert.Equal(t, "widgets", p.Category)
	}
}
