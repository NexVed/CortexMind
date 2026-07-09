import { QueryClient } from '@tanstack/solid-query';

// Single shared query client for the whole app. Sensible defaults for a
// local-first daemon: data is fresh for 30s, retries once, and we don't
// aggressively refetch on window focus (the daemon is on the same machine).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});
