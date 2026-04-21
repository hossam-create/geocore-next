'use client';

import {
  LiveKitRoom,
  RoomAudioRenderer,
  useTracks,
  ParticipantTile,
} from '@livekit/components-react';
import { Track } from 'livekit-client';
import { Wifi, Eye } from 'lucide-react';

interface LiveStreamPlayerProps {
  token: string;
  livekitUrl: string;
  simulated?: boolean;
  viewerCount: number;
  onDisconnect?: () => void;
}

function SimulatedPlayer({ viewerCount }: { viewerCount: number }) {
  return (
    <div className="relative w-full aspect-video bg-gray-900 rounded-2xl overflow-hidden flex items-center justify-center">
      <div className="text-center text-white">
        <div className="w-20 h-20 bg-white/10 rounded-full flex items-center justify-center mx-auto mb-4 animate-pulse">
          <Wifi className="w-10 h-10 text-white/60" />
        </div>
        <p className="text-lg font-semibold">Live Stream Preview</p>
        <p className="text-sm text-white/60 mt-1">
          Set LIVEKIT_API_KEY + LIVEKIT_API_SECRET to enable real streaming
        </p>
      </div>
      <div className="absolute top-3 left-3 flex items-center gap-2">
        <span className="bg-red-500 text-white text-xs font-bold px-2.5 py-1 rounded-full flex items-center gap-1.5 animate-pulse">
          <span className="w-1.5 h-1.5 bg-white rounded-full" />
          LIVE
        </span>
        <span className="bg-black/50 text-white text-xs px-2 py-1 rounded-full flex items-center gap-1">
          <Eye className="w-3 h-3" /> {viewerCount}
        </span>
      </div>
    </div>
  );
}

function LiveVideoStage({ viewerCount }: { viewerCount: number }) {
  const tracks = useTracks([{ source: Track.Source.Camera, withPlaceholder: true }]);

  return (
    <div className="relative w-full aspect-video bg-gray-900 rounded-2xl overflow-hidden">
      {tracks.length > 0 ? (
        <ParticipantTile trackRef={tracks[0]} className="w-full h-full" />
      ) : (
        <div className="absolute inset-0 flex items-center justify-center text-white/40">
          <p className="text-sm">Waiting for host video…</p>
        </div>
      )}
      <div className="absolute top-3 left-3 flex items-center gap-2 z-10">
        <span className="bg-red-500 text-white text-xs font-bold px-2.5 py-1 rounded-full flex items-center gap-1.5 animate-pulse">
          <span className="w-1.5 h-1.5 bg-white rounded-full" />
          LIVE
        </span>
        <span className="bg-black/50 text-white text-xs px-2 py-1 rounded-full flex items-center gap-1">
          <Eye className="w-3 h-3" /> {viewerCount}
        </span>
      </div>
      <RoomAudioRenderer />
    </div>
  );
}

export default function LiveStreamPlayer({
  token,
  livekitUrl,
  simulated = false,
  viewerCount,
  onDisconnect,
}: LiveStreamPlayerProps) {
  if (simulated) {
    return <SimulatedPlayer viewerCount={viewerCount} />;
  }

  return (
    <LiveKitRoom
      token={token}
      serverUrl={livekitUrl}
      connect={true}
      video={false}
      audio={false}
      onDisconnected={onDisconnect}
      className="w-full"
    >
      <LiveVideoStage viewerCount={viewerCount} />
    </LiveKitRoom>
  );
}
