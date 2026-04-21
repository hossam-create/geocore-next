'use client';

import { useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from '@/store/auth';
import { ErrorBoundary, SentryProvider } from '@/lib/sentry';
import AIChatWidget from '@/components/ai/AIChatWidget';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

function AuthRestorer({ children }: { children: React.ReactNode }) {
  const { restoreSession } = useAuthStore();
  useEffect(() => {
    restoreSession();
  }, []);
  return <>{children}</>;
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <SentryProvider>
      <ErrorBoundary>
        <QueryClientProvider client={queryClient}>
          <AuthRestorer>
            {children}
            <AIChatWidget />
          </AuthRestorer>
        </QueryClientProvider>
      </ErrorBoundary>
    </SentryProvider>
  );
}
