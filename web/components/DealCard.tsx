import Link from 'next/link';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { formatPriceKey, formatPriceValue } from '@/lib/formatPrice';
import type { Deal } from '@/types';
import { getDealStatusLabel } from '@/types';

interface DealCardProps {
  deal: Deal;
}

export function DealCard({ deal }: DealCardProps) {
  return (
    <Link href={`/deals/${deal.id}`}>
      <Card className="cursor-pointer transition-shadow hover:shadow-lg py-1">
        <CardContent className="flex items-center justify-between p-4">
          <div>
            <p className="font-medium">Deal #{deal.id}</p>
            <p className="text-xs text-muted-foreground">
              {formatPriceKey(String(deal.duration))} – {formatPriceValue(deal.price)}
            </p>
          </div>
          <Badge>{getDealStatusLabel(deal.status)}</Badge>
        </CardContent>
      </Card>
    </Link>
  );
}
