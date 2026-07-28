// The browser-dev mock backend, assembled from two method halves (each kept
// under 500 lines). Used by api.ts when no real Wails runtime is present.
import type { Backend } from "./apiTypes";
import { mockA } from "./apiMockA";
import { mockB } from "./apiMockB";

export const mockBackend: Backend = { ...mockA, ...mockB } as Backend;
