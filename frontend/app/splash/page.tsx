'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function SplashPage() {
  const router = useRouter();

  useEffect(() => {
    const isMobile =
      /Android|iPhone|iPad|iPod|Opera Mini|IEMobile|WPDesktop/i.test(navigator.userAgent) ||
      window.innerWidth < 768;

    if (!isMobile) {
      router.replace('/');
      return;
    }

    const t = setTimeout(() => router.replace('/'), 2000);
    return () => clearTimeout(t);
  }, [router]);

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 9999,
        overflow: 'hidden',
        animation: 'fadeIn 0.3s ease-out',
      }}
    >
      {/* Full-screen brand splash image */}
      <img
        src="/splash-screen.png"
        alt="mnbarh"
        style={{
          width: '100%',
          height: '100%',
          objectFit: 'cover',
          objectPosition: 'center',
        }}
      />

      {/* Loading bar at bottom */}
      <div
        style={{
          position: 'absolute',
          bottom: 48,
          left: '50%',
          transform: 'translateX(-50%)',
          width: 120,
          height: 3,
          borderRadius: 2,
          background: 'rgba(255,255,255,0.25)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            background: '#FFC220',
            borderRadius: 2,
            animation: 'progressBar 1.8s ease-out forwards',
          }}
        />
      </div>

      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; }
          to   { opacity: 1; }
        }
        @keyframes progressBar {
          from { width: 0%; }
          to   { width: 100%; }
        }
      `}</style>
    </div>
  );
}
