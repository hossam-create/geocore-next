type Params = Record<string, string | number | boolean | null | undefined>;

/**
 * Minimal analytics facade — swap implementation (Segment/PostHog/Firebase)
 * without touching call sites. Defaults to console logging in development.
 */
export interface AnalyticsService {
  identify(userId: string, traits?: Params): void;
  track(event: string, params?: Params): void;
  screen(name: string, params?: Params): void;
  reset(): void;
}

class ConsoleAnalytics implements AnalyticsService {
  identify(userId: string, traits?: Params): void {
    if (__DEV__) console.log("[analytics] identify", userId, traits);
  }
  track(event: string, params?: Params): void {
    if (__DEV__) console.log("[analytics] track", event, params);
  }
  screen(name: string, params?: Params): void {
    if (__DEV__) console.log("[analytics] screen", name, params);
  }
  reset(): void {
    if (__DEV__) console.log("[analytics] reset");
  }
}

export const analytics: AnalyticsService = new ConsoleAnalytics();
