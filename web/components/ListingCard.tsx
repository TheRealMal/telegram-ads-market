'use client';

import Link from 'next/link';
import { Card, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { parseListingPrices, formatPriceEntry } from '@/lib/formatPrice';
import type { Listing } from '@/types';
import { Clock } from 'lucide-react';

function formatFollowers(n: number): string {
  if (n >= 1e6) return `${(n / 1e6).toFixed(1).replace(/\.0$/, '')}M`;
  if (n >= 1e3) return `${(n / 1e3).toFixed(1).replace(/\.0$/, '')}k`;
  return n.toLocaleString();
}

interface ListingCardProps {
  listing: Listing;
  showApply?: boolean;
}

export function ListingCard({ listing, showApply = false }: ListingCardProps) {
  const priceRows = parseListingPrices(listing.prices);
  const displayRows = priceRows.slice(0, 2);

  return (
    <Card className="cursor-pointer transition-shadow hover:shadow-lg">
      <Link href={`/listings/${listing.id}`}>
        <CardContent className="py-1 px-4">
          <div className="mb-3 flex items-start gap-3">
            {listing.channel_photo ? (
              <img
                src={
                  listing.channel_photo.startsWith('http') || listing.channel_photo.startsWith('data:')
                    ? listing.channel_photo
                    : `data:image/jpeg;base64,${listing.channel_photo}`
                }
                alt=""
                className="h-12 w-12 flex-shrink-0 rounded-full object-cover"
              />
            ) : (
              <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-muted text-lg text-muted-foreground">
                #
              </div>
            )}
            <div className="min-w-0 flex-1">
              <h3 className="truncate font-semibold">Listing #{listing.id}</h3>
              {(listing.channel_title != null || listing.channel_username != null) ? (
                <p className="mt-0.5 truncate text-sm text-muted-foreground">
                  {listing.channel_username ? (
                    <span
                      role="button"
                      tabIndex={0}
                      className="cursor-pointer text-primary underline hover:no-underline"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        window.open(`https://t.me/${listing.channel_username}`, '_blank', 'noopener,noreferrer');
                      }}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          e.stopPropagation();
                          window.open(`https://t.me/${listing.channel_username}`, '_blank', 'noopener,noreferrer');
                        }
                      }}
                    >
                      {listing.channel_title ?? listing.channel_username}
                    </span>
                  ) : (
                    listing.channel_title ?? '—'
                  )}
                </p>
              ) : (
                <p className="mt-0.5 truncate text-sm text-muted-foreground">
                  {listing.type === 'lessor' ? 'Channel' : 'Advertiser'}
                </p>
              )}
              {listing.channel_followers != null && listing.channel_followers > 0 && (
                <p className="mt-0.5 text-xs text-muted-foreground">
                  {formatFollowers(listing.channel_followers)} followers
                </p>
              )}
              <div className="mt-1 flex items-center gap-2">
                {listing.status !== 'active' && (
                  <Badge variant="secondary" className="text-xs">
                    {listing.status}
                  </Badge>
                )}
              </div>
              {listing.categories && listing.categories.length > 0 && (
                <div className="mt-1 flex flex-wrap gap-1">
                  {listing.categories.slice(0, 3).map((c) => (
                    <Badge key={c} variant="outline" className="text-xs font-normal">
                      {c}
                    </Badge>
                  ))}
                  {listing.categories.length > 3 && (
                    <span className="text-xs text-muted-foreground">+{listing.categories.length - 3}</span>
                  )}
                </div>
              )}
            </div>
          </div>

          {displayRows.length > 0 && (
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground">Pricing</p>
              <div className="flex flex-wrap gap-2">
                {displayRows.map((row, i) => (
                  <div
                    key={i}
                    className="flex items-center gap-1 rounded bg-muted/50 px-2 py-1 text-xs"
                  >
                    <Clock size={12} />
                    <span>{formatPriceEntry(row.duration, row.price)}</span>
                  </div>
                ))}
                {priceRows.length > 2 && (
                  <span className="self-center text-xs text-muted-foreground">
                    +{priceRows.length - 2} more
                  </span>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Link>
      {showApply && (
        <CardFooter className="p-4 pt-0">
          <Link
            href={`/listings/${listing.id}`}
            className={cn(
              'inline-flex h-9 w-full items-center justify-center rounded-md px-4 py-2 text-sm font-medium shadow-sm transition-colors',
              listing.type === 'lessor'
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
            )}
          >
            {listing.type === 'lessor' ? 'Apply to Channel' : 'Contact Advertiser'}
          </Link>
        </CardFooter>
      )}
    </Card>
  );
}
