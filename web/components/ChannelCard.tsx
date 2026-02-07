import Link from 'next/link';
import { Card, CardContent } from '@/components/ui/card';
import type { Channel } from '@/types';

interface ChannelCardProps {
  channel: Channel;
}

export function ChannelCard({ channel }: ChannelCardProps) {
  return (
    <Link href={`/profile/channels/${channel.id}`}>
      <Card className="my-1 cursor-pointer transition-shadow hover:shadow-lg">
        <CardContent className="flex items-center gap-3 p-0">
          <div className="m-3 flex flex-1 items-center gap-3">
          {channel.photo ? (
            <img
              src={channel.photo}
              alt=""
              className="h-10 w-10 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-lg text-muted-foreground">
              #
            </div>
          )}
          <div>
            <p className="font-medium">{channel.title}</p>
            {channel.username && (
              <p className="text-sm text-muted-foreground">@{channel.username}</p>
            )}
          </div>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
