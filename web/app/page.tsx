'use client';

import { useEffect, useState, useRef, useCallback } from 'react';
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
  const [searchFocused, setSearchFocused] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [filterExiting, setFilterExiting] = useState(false);
  const [filterCategories, setFilterCategories] = useState<string[]>([]);
  const [filterMinFollowers, setFilterMinFollowers] = useState('');
  const filterPanelRef = useRef<HTMLDivElement>(null);

  const filterPanelVisible = showFilters || filterExiting;

  const closeFilters = useCallback(() => {
    if (!showFilters || filterExiting) return;
    setShowFilters(false);
    setFilterExiting(true);
  }, [showFilters, filterExiting]);

  const toggleFilters = useCallback(() => {
    if (filterExiting) return;
    if (showFilters) {
      closeFilters();
    } else {
      setShowFilters(true);
    }
  }, [showFilters, filterExiting, closeFilters]);

  useEffect(() => {
    if (!showFilters || filterExiting) return;
    const handlePointerDown = (e: PointerEvent) => {
      if (filterPanelRef.current?.contains(e.target as Node)) return;
      closeFilters();
    };
    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, [showFilters, filterExiting, closeFilters]);

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

  const handleSearchFocus = useCallback(() => {
    setSearchFocused(true);
    if (showFilters && !filterExiting) closeFilters();
  }, [showFilters, filterExiting, closeFilters]);

  return (
    <>
      <div className={`page-with-nav ${loading ? 'opacity-0' : 'opacity-100'}`}>
      <PageTopSpacer />
      <div className="mx-auto max-w-4xl px-4 py-4">
        <div className="relative">
          {/* Search bar + filter button row */}
          <div className="mb-4 flex items-center">
            {/* Search bar — expands to full width when focused */}
            <div className="glass-pill flex flex-1 items-center bg-white/72 shadow-none backdrop-blur-xl backdrop-saturate-150 dark:bg-black/48">
              <Search
                size={18}
                className="ml-3 shrink-0 text-muted-foreground"
              />
              <Input
                placeholder="Search..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={handleSearchFocus}
                onBlur={() => setSearchFocused(false)}
                className="flex-1 min-w-0 border-0 bg-transparent pl-2 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
              />
            </div>

            {/* Filter button — shrinks away when search is focused */}
            <button
              type="button"
              onClick={toggleFilters}
              className={`glass-pill flex h-10 shrink-0 items-center justify-center bg-white/72 shadow-none backdrop-blur-xl backdrop-saturate-150 dark:bg-black/48 overflow-hidden${
                showFilters ? ' filter-btn-open' : ''
              } ${
                filterPanelVisible
                  ? 'text-primary'
                  : 'text-muted-foreground hover:text-foreground'
              } ${
                searchFocused
                  ? 'w-0 opacity-0 ml-0 p-0 pointer-events-none'
                  : 'w-10 ml-2'
              }`}
            >
              <SlidersHorizontal size={18} />
            </button>
          </div>

          {/* Filter dropdown */}
          {filterPanelVisible && (
            <div
              ref={filterPanelRef}
              className={`absolute left-0 right-0 top-10 z-20 ${
                filterExiting ? 'filter-panel-exit' : 'filter-panel-enter'
              }`}
              onAnimationEnd={() => {
                if (filterExiting) setFilterExiting(false);
              }}
            >
              {/* Extension strip — bridges button to panel seamlessly */}
              <div className="flex justify-end">
                <div
                  className="relative z-[1] w-10 bg-background glass-panel-extension"
                  style={{ marginTop: '-1px', marginBottom: '-1px', height: '10px' }}
                />
              </div>
              {/* Panel body */}
              <div className="glass-panel-connected-bottom rounded-lg rounded-tr-none bg-background p-4 space-y-4">
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
