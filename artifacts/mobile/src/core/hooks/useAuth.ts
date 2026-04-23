import { useEffect } from "react";

import { useAuthStore } from "../../features/auth/store/auth.store";

/**
 * Ergonomic hook that exposes the current auth state. On first mount it
 * triggers `restoreSession` so callers don't need to wire that into their
 * root components — the store itself dedupes concurrent restores.
 */
export function useAuth() {
  const state = useAuthStore();
  useEffect(() => {
    if (state.isRestoring && !state.isAuthenticated) {
      void state.restoreSession();
    }
    // run once — subsequent state changes should not re-trigger restore
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return state;
}
