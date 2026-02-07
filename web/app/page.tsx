'use client';

import { useEffect, useState } from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Search, SlidersHorizontal, X } from 'lucide-react';
import { ListingCard } from '@/components/ListingCard';
import { api } from '@/lib/api';
import { LISTING_CATEGORIES } from '@/lib/constants';
import type { Listing } from '@/types';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';

export default function MarketplacePage() {
  const [lessorListings, setLessorListings] = useState<Listing[]>([]);
  const [lesseeListings, setLesseeListings] = useState<Listing[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [showFilters, setShowFilters] = useState(false);
  const [filterCategories, setFilterCategories] = useState<string[]>([]);
  const [filterMinFollowers, setFilterMinFollowers] = useState('');

  useEffect(() => {
    const paramsLessor = new URLSearchParams();
    paramsLessor.set('type', 'lessor');
    if (filterCategories.length > 0) paramsLessor.set('categories', filterCategories.join(','));
    const minF = filterMinFollowers.trim();
    if (minF !== '' && !Number.isNaN(Number(minF)) && Number(minF) >= 0) {
      paramsLessor.set('min_followers', String(Number(minF)));
    }
    const paramsLessee = new URLSearchParams();
    paramsLessee.set('type', 'lessee');
    if (filterCategories.length > 0) paramsLessee.set('categories', filterCategories.join(','));

    const load = async () => {
      setLoading(true);
      try {
        const [lessors, lessees] = await Promise.all([
          api<Listing[]>(`/api/v1/market/listings?${paramsLessor.toString()}`),
          api<Listing[]>(`/api/v1/market/listings?${paramsLessee.toString()}`),
        ]);
        setLessorListings(lessors.ok && lessors.data ? lessors.data : []);
        setLesseeListings(lessees.ok && lessees.data ? lessees.data : []);
      } catch (e) {
        console.error(e);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [filterCategories, filterMinFollowers]);

  const filterBySearch = (list: Listing[]) => {
    if (!searchQuery.trim()) return list;
    const q = searchQuery.toLowerCase();
    return list.filter((l) => String(l.id).includes(q) || l.type.toLowerCase().includes(q));
  };
  const filteredLessors = filterBySearch(lessorListings);
  const filteredLessees = filterBySearch(lesseeListings);

  return (
    <div className="min-h-screen pb-20">
      <div className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto max-w-4xl px-4 py-4">
          <h1 className="mb-4 text-2xl font-bold">Marketplace</h1>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search
                size={18}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                placeholder="Search..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <Button
              variant="outline"
              size="icon"
              onClick={() => setShowFilters((v) => !v)}
              className={showFilters ? 'bg-accent' : ''}
            >
              <SlidersHorizontal size={18} />
            </Button>
          </div>
          {showFilters && (
            <div className="mt-4 space-y-4 rounded-lg border border-border bg-muted/30 p-4">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Filters</span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setFilterCategories([]);
                    setFilterMinFollowers('');
                  }}
                >
                  <X size={14} className="mr-1" />
                  Clear
                </Button>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Categories</Label>
                <div className="mt-1 flex flex-wrap gap-2">
                  {LISTING_CATEGORIES.map((cat) => (
                    <button
                      key={cat}
                      type="button"
                      onClick={() =>
                        setFilterCategories((prev) =>
                          prev.includes(cat) ? prev.filter((c) => c !== cat) : [...prev, cat]
                        )
                      }
                      className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                        filterCategories.includes(cat)
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-background hover:bg-muted'
                      }`}
                    >
                      {cat}
                    </button>
                  ))}
                </div>
              </div>
              <div>
                <Label htmlFor="min-followers" className="text-xs text-muted-foreground">
                  Min. channel followers
                </Label>
                <Input
                  id="min-followers"
                  type="number"
                  min={0}
                  placeholder="e.g. 1000"
                  value={filterMinFollowers}
                  onChange={(e) => setFilterMinFollowers(e.target.value)}
                  className="mt-1 max-w-[140px]"
                />
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Only channels with stats (Find Channels tab)
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="mx-auto max-w-4xl px-4 py-4">
        <Tabs defaultValue="channels" className="w-full">
          <TabsList className="mb-6 grid w-full grid-cols-2">
            <TabsTrigger value="channels">Find Channels</TabsTrigger>
            <TabsTrigger value="advertisers">Find Advertisers</TabsTrigger>
          </TabsList>

          <TabsContent value="channels" className="space-y-4">
            {loading ? (
              <div className="flex justify-center py-8">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              </div>
            ) : filteredLessors.length === 0 ? (
              <p className="py-8 text-center text-muted-foreground">No channels found</p>
            ) : (
              <div className="grid gap-4">
                {filteredLessors.map((l) => (
                  <ListingCard key={l.id} listing={l} />
                ))}
              </div>
            )}
          </TabsContent>

          <TabsContent value="advertisers" className="space-y-4">
            {loading ? (
              <div className="flex justify-center py-8">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              </div>
            ) : filteredLessees.length === 0 ? (
              <p className="py-8 text-center text-muted-foreground">No advertiser requests found</p>
            ) : (
              <div className="grid gap-4">
                {filteredLessees.map((l) => (
                  <ListingCard key={l.id} listing={l} />
                ))}
              </div>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
