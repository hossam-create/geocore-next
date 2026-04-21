'use client';
import { useEffect, Component, ReactNode } from 'react';
import * as Sentry from '@sentry/react';

// Initialize Sentry - call this in layout or _app
export function initSentry() {
  const dsn = process.env.NEXT_PUBLIC_SENTRY_DSN;
  if (!dsn) {
    console.info('Sentry disabled: NEXT_PUBLIC_SENTRY_DSN not set');
    return;
  }

  Sentry.init({
    dsn,
    environment: process.env.NODE_ENV,
    release: process.env.NEXT_PUBLIC_APP_VERSION || '1.0.0',
    tracesSampleRate: 0.2,
    replaysSessionSampleRate: 0.1,
    replaysOnErrorSampleRate: 1.0,
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration(),
    ],
  });
}

// Error Boundary component
interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
    
    // Report to Sentry
    if (process.env.NEXT_PUBLIC_SENTRY_DSN) {
      Sentry.withScope((scope: Sentry.Scope) => {
        scope.setTag('errorBoundary', true);
        scope.setExtra('componentStack', errorInfo.componentStack);
        Sentry.captureException(error);
      });
    }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
          <div className="max-w-md w-full bg-white rounded-2xl shadow-sm p-8 text-center">
            <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-1.964-1.333-2.732 0L3.082 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Something went wrong</h1>
            <p className="text-gray-500 mb-6">
              We're sorry, but something unexpected happened. Our team has been notified.
            </p>
            <button
              onClick={() => {
                this.setState({ hasError: false, error: undefined });
                window.location.reload();
              }}
              className="inline-flex items-center gap-2 bg-[#0071CE] text-white px-6 py-3 rounded-xl font-semibold hover:bg-[#005ba3] transition-colors"
            >
              Reload Page
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

// Hook for capturing errors manually
export function useSentry() {
  const captureError = (error: Error, context?: Record<string, unknown>) => {
    if (process.env.NEXT_PUBLIC_SENTRY_DSN) {
      Sentry.withScope((scope: Sentry.Scope) => {
        if (context) {
          Object.entries(context).forEach(([key, value]) => {
            scope.setExtra(key, value);
          });
        }
        Sentry.captureException(error);
      });
    }
  };

  const captureMessage = (message: string, level: Sentry.SeverityLevel = 'info') => {
    if (process.env.NEXT_PUBLIC_SENTRY_DSN) {
      Sentry.captureMessage(message, { level });
    }
  };

  const setUser = (id: string, email?: string, name?: string) => {
    if (process.env.NEXT_PUBLIC_SENTRY_DSN) {
      Sentry.setUser({ id, email, username: name });
    }
  };

  const clearUser = () => {
    if (process.env.NEXT_PUBLIC_SENTRY_DSN) {
      Sentry.setUser(null);
    }
  };

  return { captureError, captureMessage, setUser, clearUser };
}

// Sentry Provider component to wrap app
export function SentryProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    initSentry();
  }, []);

  return <>{children}</>;
}
