import PocketBase from 'pocketbase';

// PocketBase client pointing at the local daemon.
// In development, Vite proxies /api/ requests to this URL.
// In production (Wails), the embedded server runs on this port.
const PB_URL = import.meta.env.VITE_PB_URL || 'http://127.0.0.1:8090';

export const pb = new PocketBase(PB_URL);

// Disable auto-cancellation so multiple concurrent requests don't
// abort each other (SolidJS fires many resource fetches in parallel).
pb.autoCancellation(false);
