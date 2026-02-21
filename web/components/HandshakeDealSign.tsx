'use client';

import { useState, useEffect, useRef } from 'react';

const EMOJI_SIZE = 100;
const TRAVEL_DIST = 72;

interface HandshakeAnimationProps {
  leftReady: boolean;
  rightReady: boolean;
}

function HandshakeAnimation({ leftReady, rightReady }: HandshakeAnimationProps) {
  const bothReady = leftReady && rightReady;
  const [handsJoined, setHandsJoined] = useState(false);
  const [showSpark, setShowSpark] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const sparkRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    if (sparkRef.current) clearTimeout(sparkRef.current);
    if (bothReady) {
      timerRef.current = setTimeout(() => {
        setHandsJoined(true);
        setShowSpark(true);
        sparkRef.current = setTimeout(() => setShowSpark(false), 600);
      }, 480);
    } else {
      setHandsJoined(false);
      setShowSpark(false);
    }
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      if (sparkRef.current) clearTimeout(sparkRef.current);
    };
  }, [bothReady]);

  return (
    <div
      style={{
        position: 'relative',
        width: 260,
        height: 120,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <div
        style={{
          position: 'absolute',
          left: 0,
          top: '50%',
          fontSize: EMOJI_SIZE,
          lineHeight: 1,
          userSelect: 'none',
          transform: bothReady
            ? `translateY(-50%) translateX(${TRAVEL_DIST}px)`
            : 'translateY(-50%) translateX(0px)',
          opacity: leftReady && !handsJoined ? 1 : 0,
          transition: [
            'transform 0.5s cubic-bezier(0.34,1.2,0.64,1)',
            'opacity 0.28s ease',
          ].join(', '),
          pointerEvents: 'none',
        }}
      >
        🫱
      </div>
      <div
        style={{
          position: 'absolute',
          right: 0,
          top: '50%',
          fontSize: EMOJI_SIZE,
          lineHeight: 1,
          userSelect: 'none',
          transform: bothReady
            ? `translateY(-50%) translateX(-${TRAVEL_DIST}px)`
            : 'translateY(-50%) translateX(0px)',
          opacity: rightReady && !handsJoined ? 1 : 0,
          transition: [
            'transform 0.5s cubic-bezier(0.34,1.2,0.64,1)',
            'opacity 0.28s ease',
          ].join(', '),
          pointerEvents: 'none',
        }}
      >
        🫲
      </div>
      <div
        style={{
          position: 'absolute',
          left: '50%',
          top: '50%',
          transform: 'translate(-50%, -50%)',
          fontSize: EMOJI_SIZE,
          lineHeight: 1,
          userSelect: 'none',
          opacity: handsJoined ? 1 : 0,
          transition: 'opacity 0.3s ease',
          animation: handsJoined ? 'handshakePulse 0.4s ease-out both' : 'none',
          pointerEvents: 'none',
        }}
      >
        🤝
      </div>
      {showSpark && (
        <div
          style={{
            position: 'absolute',
            left: '50%',
            top: '20%',
            transform: 'translate(-50%, -50%)',
            fontSize: 28,
            animation: 'sparkPop 0.55s ease-out both',
            pointerEvents: 'none',
            zIndex: 10,
          }}
        >
          ✨
        </div>
      )}
      <style>{`
        @keyframes handshakePulse {
          0%   { transform: translate(-50%, -50%) scale(0.65); }
          60%  { transform: translate(-50%, -50%) scale(1.13); }
          100% { transform: translate(-50%, -50%) scale(1); }
        }
        @keyframes sparkPop {
          0%   { opacity: 0; transform: translate(-50%, -50%) scale(0.3) rotate(-20deg); }
          40%  { opacity: 1; transform: translate(-50%, -50%) scale(1.6) rotate(10deg); }
          100% { opacity: 0; transform: translate(-50%, -100%) scale(1) rotate(0deg); }
        }
      `}</style>
    </div>
  );
}

export interface HandshakeDealSignProps {
  lessorSigned: boolean;
  lesseeSigned: boolean;
  canSignNow: boolean;
  signing: boolean;
  onSignDeal: () => void;
}

export function HandshakeDealSign({
  lessorSigned,
  lesseeSigned,
  canSignNow,
  signing,
  onSignDeal,
}: HandshakeDealSignProps) {
  return (
    <div className="flex flex-col items-center gap-2 py-4">
      <button
        type="button"
        onClick={canSignNow && !signing ? onSignDeal : undefined}
        disabled={!canSignNow || signing}
        className="cursor-pointer rounded-xl transition-opacity hover:opacity-90 disabled:cursor-default disabled:opacity-100"
        aria-label={canSignNow ? 'Tap to sign the deal' : 'Waiting for both parties to connect wallet and sign'}
      >
        <HandshakeAnimation leftReady={lessorSigned} rightReady={lesseeSigned} />
      </button>
      <p className="max-w-xs text-center text-xs text-muted-foreground">
        Both sides need to connect wallet to sign. Tap here to sign the deal.
      </p>
    </div>
  );
}
