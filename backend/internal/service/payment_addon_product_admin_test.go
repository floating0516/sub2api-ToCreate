package service

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func newAddonProductAdminService(repo SubscriptionAddonRepository) *PaymentService {
	return &PaymentService{subscriptionSvc: &SubscriptionService{addonRepo: repo}}
}

func TestListAddonProductsForAdminIncludesProductsNotForSale(t *testing.T) {
	repo := &subscriptionAddonRepoStub{products: map[int64]*SubscriptionAddonProduct{
		1: {ID: 1, SKU: "available", ForSale: true},
		2: {ID: 2, SKU: "hidden", ForSale: false},
	}}

	products, err := newAddonProductAdminService(repo).ListAddonProductsForAdmin(context.Background())

	require.NoError(t, err)
	require.Len(t, products, 2)
	require.NotNil(t, repo.listForSaleOnly)
	require.False(t, *repo.listForSaleOnly)
}

func TestUpdateAddonProductUpdatesMutableFieldsAndPreservesSKU(t *testing.T) {
	repo := &subscriptionAddonRepoStub{products: map[int64]*SubscriptionAddonProduct{
		7: {ID: 7, SKU: "addon-usd-30", Name: "Old", QuotaUSD: 30, Price: 5.49, ForSale: true},
	}}
	originalPrice := 6.99

	product, err := newAddonProductAdminService(repo).UpdateAddonProduct(context.Background(), 7, UpdateSubscriptionAddonProductInput{
		Name:          "  30 USD Add-on  ",
		QuotaUSD:      35,
		Price:         5.25,
		OriginalPrice: &originalPrice,
		ForSale:       false,
		SortOrder:     25,
	})

	require.NoError(t, err)
	require.Equal(t, "addon-usd-30", product.SKU)
	require.Equal(t, "30 USD Add-on", product.Name)
	require.Equal(t, 35.0, product.QuotaUSD)
	require.Equal(t, 5.25, product.Price)
	require.Equal(t, originalPrice, *product.OriginalPrice)
	require.False(t, product.ForSale)
	require.Equal(t, 25, product.SortOrder)
	require.Equal(t, "30 USD Add-on", repo.updatedProduct.Name)
}

func TestUpdateAddonProductRejectsInvalidValues(t *testing.T) {
	negativeOriginalPrice := -1.0
	nanOriginalPrice := math.NaN()
	valid := UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: 10, Price: 1.99, ForSale: true}
	tests := []struct {
		name  string
		id    int64
		input UpdateSubscriptionAddonProductInput
	}{
		{name: "invalid id", id: 0, input: valid},
		{name: "blank name", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "  ", QuotaUSD: 10, Price: 1}},
		{name: "zero quota", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", Price: 1}},
		{name: "nan quota", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: math.NaN(), Price: 1}},
		{name: "infinite quota", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: math.Inf(1), Price: 1}},
		{name: "zero price", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: 10}},
		{name: "nan price", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: 10, Price: math.NaN()}},
		{name: "negative original price", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: 10, Price: 1, OriginalPrice: &negativeOriginalPrice}},
		{name: "nan original price", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: 10, Price: 1, OriginalPrice: &nanOriginalPrice}},
		{name: "negative sort order", id: 1, input: UpdateSubscriptionAddonProductInput{Name: "Add-on", QuotaUSD: 10, Price: 1, SortOrder: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &subscriptionAddonRepoStub{products: map[int64]*SubscriptionAddonProduct{1: {ID: 1, SKU: "addon-usd-10"}}}
			_, err := newAddonProductAdminService(repo).UpdateAddonProduct(context.Background(), tt.id, tt.input)
			require.Error(t, err)
			require.Nil(t, repo.updatedProduct)
		})
	}
}
