// Predefined listing categories (must match backend domain.ListingCategories).
export const LISTING_CATEGORIES = [
  'Tech',
  'Crypto',
  'Gaming',
  'News',
  'Education',
  'Entertainment',
  'Lifestyle',
  'Business',
  'Finance',
  'Sports',
  'Other',
] as const;

export type ListingCategory = (typeof LISTING_CATEGORIES)[number];
