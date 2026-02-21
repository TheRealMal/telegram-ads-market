import Link from 'next/link';
import { Card, CardContent } from '@/components/ui/card';
import { RefreshCw } from 'lucide-react';
import type { Channel } from '@/types';

interface ChannelCardProps {
  channel: Channel;
  onRefreshClick?: (channelId: number) => void;
  refreshDisabled?: boolean;
}

export function ChannelCard({ channel, onRefreshClick, refreshDisabled }: ChannelCardProps) {
  return (
    <Card className="py-1 my-1 transition-shadow hover:shadow-lg">
      <CardContent className="flex items-center gap-3 p-0">
        <Link href={`/profile/channels/${channel.id}`} className="m-3 flex flex-1 min-w-0 items-center gap-3 cursor-pointer">
          {channel.photo ? (
            <img
              src={
                channel.photo.startsWith('http') || channel.photo.startsWith('data:')
                  ? channel.photo
                  : `data:image/jpeg;base64,${channel.photo}`
              }
              alt=""
              className="h-10 w-10 shrink-0 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-muted text-lg text-muted-foreground">
              #
            </div>
          )}
          <div className="min-w-0">
            <p className="font-medium truncate">{channel.title}</p>
            {channel.username && (
              <p className="text-sm text-muted-foreground">@{channel.username}</p>
            )}
          </div>
        </Link>
        {onRefreshClick != null && (
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onRefreshClick(channel.id);
            }}
            disabled={refreshDisabled}
            className="mr-3 flex shrink-0 items-center justify-center rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground disabled:opacity-50 disabled:pointer-events-none"
            aria-label="Refresh channel stats"
          >
            <RefreshCw size={18} />
          </button>
        )}
      </CardContent>
    </Card>
  );
}
