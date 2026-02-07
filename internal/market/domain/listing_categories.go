package domain

import (
	"errors"
	"slices"
)

var ErrInvalidListingCategory = errors.New("listing category must be one of the predefined values")

// Predefined listing categories. Only these may be stored.
var ListingCategories = []string{
	"Tech",
	"Crypto",
	"Gaming",
	"News",
	"Education",
	"Entertainment",
	"Lifestyle",
	"Business",
	"Finance",
	"Sports",
	"Other",
}

// ValidateListingCategories returns nil if every category in list is predefined; otherwise ErrInvalidListingCategory.
func ValidateListingCategories(categories []string) error {
	for _, c := range categories {
		if c == "" {
			continue
		}
		if !slices.Contains(ListingCategories, c) {
			return ErrInvalidListingCategory
		}
	}
	return nil
}
