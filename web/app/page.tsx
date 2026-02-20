'use client';

import { useEffect, useState } from 'react';
import { Tabs, TabsContent, TabsListWithPill, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Search, SlidersHorizontal, X } from 'lucide-react';
import { ListingCard } from '@/components/ListingCard';
import { PageTopSpacer } from '@/components/PageTopSpacer';
import { api } from '@/lib/api';
import { LISTING_CATEGORIES } from '@/lib/constants';
import type { Listing } from '@/types';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { LoadingScreen } from '@/components/LoadingScreen';

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
    <>
      <div className={`page-with-nav ${loading ? 'opacity-0' : 'opacity-100'}`}>
      <PageTopSpacer />
      <div className="mx-auto max-w-4xl px-4 py-4">
        <div className="relative">
        <div className="flex gap-2 mb-4">
          <div className="glass-pill relative flex-1 overflow-hidden rounded-full bg-white/72 shadow-none backdrop-blur-xl backdrop-saturate-150 dark:bg-black/48">
            <Search
              size={18}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              placeholder="Search..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="border-0 bg-transparent pl-10 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
            />
          </div>
          <button
            type="button"
            onClick={() => setShowFilters((v) => !v)}
            className={
              'glass-pill flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-white/72 shadow-none backdrop-blur-xl backdrop-saturate-150 dark:bg-black/48 ' +
              (showFilters
                ? 'ring-2 ring-primary ring-offset-2 ring-offset-background text-primary'
                : 'text-muted-foreground hover:text-foreground')
            }
          >
            <SlidersHorizontal size={18} />
          </button>
        </div>
        {showFilters && (
          <div className="absolute left-0 right-0 top-12 z-20 space-y-4 rounded-lg border border-border bg-background p-4 shadow-lg">
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
            </div>
          </div>
        )}
        <Tabs defaultValue="channels" className="w-full">
          <TabsListWithPill className="glass-pill mb-6 grid w-full grid-cols-2 gap-0.5 rounded-full border-0 bg-white/72 p-1 shadow-none backdrop-blur-xl backdrop-saturate-150 dark:bg-black/48">
            <TabsTrigger
              value="channels"
              className="rounded-full border-0 data-[state=active]:bg-muted data-[state=active]:text-foreground data-[state=active]:shadow-none"
            >
              Find Channels
            </TabsTrigger>
            <TabsTrigger
              value="advertisers"
              className="rounded-full border-0 data-[state=active]:bg-muted data-[state=active]:text-foreground data-[state=active]:shadow-none"
            >
              Find Advertisers
            </TabsTrigger>
          </TabsListWithPill>

          <TabsContent value="channels" className="space-y-4">
            {filteredLessors.length === 0 ? (
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
            {filteredLessees.length === 0 ? (
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
      </div>
      <LoadingScreen show={loading} />
    </>
  );
}
