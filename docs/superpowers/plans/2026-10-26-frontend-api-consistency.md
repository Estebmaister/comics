# Frontend API Consistency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route every React REST call through one typed, runtime-validating client so public reads remain anonymous, admin writes are authenticated, failed mutations never change UI state, and every successful mutation reconciles the visible catalog.

**Architecture:** Add a single `RestClient` as the only production owner of `fetch`, bearer injection, JSON parsing, normalized errors, abort propagation, and 401 notification. Thin typed `authApi` and `comicsApi` endpoint modules provide request/response decoders; React providers and components consume those modules, keep the catalog list as the refresh boundary, and use the existing global toast system for user-facing outcomes.

**Tech Stack:** React 19.1, TypeScript 6, React Router 7, Vitest 3 + Testing Library, Playwright 1.64.1, Go/Gin REST JSON API.

## Global Constraints

- Preserve the approved security model: `GET /comics`, `GET /comics/:id`, and `GET /comics/search/:title` are public; create, update, delete, merge, cover visibility, and manual scrape operations require an admin JWT.
- The server remains the authorization boundary. Hiding admin controls is UX hardening, not a replacement for Workstream 1 route middleware.
- The browser stores only `comics_access_token` in `sessionStorage`; do not add local-storage persistence or client-provided role headers.
- Preserve Workstream 6's rotating refresh contract: refresh credentials stay
  only in the HttpOnly cookie, requests use `credentials: 'include'`, browser
  mutations send `X-CSRF-Protection: 1`, refresh calls are deduplicated with
  Web Locks plus an in-tab promise, and logout calls the server after clearing
  local state.
- The sole production `fetch` call must live in `src/frontend/api/client.ts`.
- Every successful JSON response is decoded from `unknown`; malformed JSON and shape mismatches become `ApiError` instances and never reach component state.
- Normalize the target `{ "error": { "code": string, "message": string, "details"?: unknown } }` envelope and the rollout-compatible legacy `{ "message": string }` and `{ "error": string }` envelopes.
- Invoke centralized unauthorized handling only for `401` responses to requests marked `auth: 'required'`. A `403` keeps the session and reports that admin access is required.
- Catalog callers suppress their workflow-specific error toast for authenticated `401` failures because the centralized session-expired toast already explains the action.
- Public catalog reads use `auth: 'none'` plus `credentials: 'omit'`, so they send neither bearer headers nor auth cookies even when a session exists.
- Forward the caller's `AbortSignal` unchanged. Abort failures are control flow: do not wrap them, clear catalog data, or show error toasts.
- Abort superseded/unmounted list and scrape-status reads. Do not present client abort as rollback for an already-dispatched catalog mutation or manual scrape; their server outcome remains authoritative.
- Never update local card/list state, close a modal, or call an invalidation callback until the client has checked `response.ok` and decoded the success payload.
- Every successful catalog mutation invalidates the list. Returned comics may update the card immediately, but the subsequent list request is canonical for filtering, ordering, counts, and pagination.
- Reset `from` to `0` with history replacement whenever viewport-derived `limit` changes.
- Remove request/payload `console.debug` statements. Keep unrelated deliberate `console.error` diagnostics and the development-only web-vitals logging.
- Do not add a query cache library, schema-validation dependency, new
  server-side refresh-token behavior, service worker, backend route, or
  database migration in this workstream. Port Workstream 6's existing
  `/refresh-token` and `/logout` browser client behavior into the typed client;
  do not remove it with `AuthHelpers.ts`.

## Dependency Gates

1. **Workstream 1 — security boundary:** `docs/superpowers/plans/2026-10-26-security-boundary.md` must protect the existing mutation and scrape paths with JWT + admin middleware while retaining the public read paths. It creates the minimal bearer helper and tests at `src/frontend/api/client.ts` and `src/frontend/api/client.test.ts`; Task 1 evolves those files into the typed transport instead of creating a competing client. Final mutation smoke tests cannot pass until this dependency lands.
2. **Workstream 6 — authentication hardening:** `docs/superpowers/plans/2026-10-26-auth-hardening.md` owns active-user enforcement, refresh rotation/revocation, OAuth state, and server-side role trust. This plan consumes its access-token, HttpOnly refresh-cookie, CSRF, deduplication, server-logout, and profile-role contracts; it must not decode JWT claims, expose refresh tokens to JavaScript, or send a `Role` header.
3. **Workstream 5 — scrape reliability:** `docs/superpowers/plans/2026-10-26-scrape-reliability.md` may enrich scrape success/status payloads. Keep its response decoder in `src/frontend/api/comics.ts`; update that decoder and its fixture in the Workstream 5 integration commit rather than parsing scrape payloads in the component.
4. Before the final rollout commit, run the frontend contract suite against the merged Workstream 1 and 6 branches. Required shared contract: JSON `401`, JSON `403`, profile `role: "user" | "admin"`, unchanged catalog route paths, refresh cookie rotation, idempotent server logout, and CORS allowing `Authorization`, `Accept`, `Content-Type`, and `X-CSRF-Protection`.

## File and Responsibility Map

**Create**

- `src/frontend/auth/session.ts` — access-token storage key and storage helpers only.
- `src/frontend/auth/session.test.ts` — storage round-trip and clear behavior.
- `src/frontend/api/contracts.ts` — small runtime primitives for records, strings, numbers, booleans, arrays, and optional values.
- `src/frontend/api/contracts.test.ts` — primitive decoder failure messages.
- `src/frontend/api/comics.ts` — comic/list/pagination/scrape decoders and all catalog endpoints.
- `src/frontend/api/comics.test.ts` — URL/query, headers, request body, auth mode, and response-shape tests.
- `src/frontend/api/auth.ts` — login, signup, refresh, logout, profile, and profile-update endpoints, decoders, and refresh deduplication.
- `src/frontend/api/auth.test.ts` — auth envelope/profile decoding, CSRF/cookie behavior, refresh deduplication/Web Locks, logout, and required/anonymous auth-mode tests.
- `src/frontend/context/AuthContext.test.tsx` — rehydration, role, centralized 401, and non-401 retention tests.
- `src/frontend/hooks/useComicActions.test.tsx` — mutation success/error ordering and invalidation tests.
- `src/frontend/components/Comics/Edition/CreateComic.test.tsx` — create success/error toast and invalidation tests.
- `src/frontend/components/Comics/Edition/EditComic.test.tsx` — edit success/error state ordering and invalidation tests.
- `playwright.config.ts` — one Chromium project and local Vite web server.
- `tests/e2e/frontend-api-consistency.spec.ts` — public-read, bearer-write, 401, invalidation, and resize browser flows.
- `scripts/smoke/frontend-api-consistency.mjs` — opt-in live API security and admin mutation round trip.

**Modify**

- `src/frontend/api/client.ts` and `src/frontend/api/client.test.ts` — replace Workstream 1's minimal bearer helper and DOM session-expired event with the sole typed `fetch` transport and one registered unauthorized callback while preserving its 401/403 behavior.
- `src/frontend/util/Config.ts` and `src/frontend/util/Config.test.ts` — keep environment resolution; ensure the client receives a trailing-slash-free base URL.
- `src/frontend/components/Comics/types.ts` — match the actual REST comic shape and add exact mutation/list types.
- `src/frontend/context/AuthContext.tsx` — consume `authApi`, preserve Workstream 6's `status`, expose `isAdmin`/`authReady`, bootstrap through cookie refresh, call server logout, rehydrate profile, and register one unauthorized handler.
- `src/index.tsx` — mount `ToastProvider` above `AuthProvider` and all routes.
- `src/frontend/components/Auth/LoginPage.tsx`, `AuthNav.tsx`, and
  `ProfilePage.tsx` — consume the final context/API contract and normalized
  failures without legacy-helper imports or raw response-status handling.
- `src/frontend/components/Comics/MainPage/MainPage.tsx` and `MainPage.test.tsx` — typed list call, real abort propagation, unified invalidation, admin gating, and resize offset reset.
- `src/frontend/components/Comics/Card/ComicsList.tsx`, `ComicCard.tsx`, `ComicCard.test.tsx`, `ComicCardContext.tsx`, and `ComicCover.tsx` — admin mutation controls, validated updates, toasts, and list invalidation.
- `src/frontend/hooks/useComicActions.ts` — async typed update/delete actions with state changes after success only.
- `src/frontend/components/Comics/Edition/CreateComic.tsx`, `EditComic.tsx`, `MergeComic.tsx`, `MergeComic.test.tsx`, and `ScrapeButton.tsx` — endpoint-module calls, consistent toasts, abort-aware polling, and success invalidation.
- `package.json`, `package-lock.json`, `Makefile`, and `.github/workflows/ci.yml` — Playwright script/target/dependency and Chromium CI execution.

**Delete after all imports migrate**

- `src/frontend/util/ServerHelpers.ts`
- `src/frontend/util/AuthHelpers.ts`
- `src/frontend/util/AuthHelpers.test.ts` after its refresh/CSRF/logout cases
  move to `src/frontend/api/auth.test.ts` and `client.test.ts`

**Modify after Workstream 5's rename**

- `src/frontend/components/Comics/Edition/ScrapeButton.test.tsx` — extend the component-level toast and polling tests established by the scrape-reliability workstream.

---

### Task 1: Access-token storage and typed REST transport

**Files:**
- Create: `src/frontend/auth/session.ts`
- Create: `src/frontend/auth/session.test.ts`
- Modify: `src/frontend/api/client.ts`
- Modify: `src/frontend/api/client.test.ts`
- Modify: `src/frontend/util/Config.ts:1-31`
- Modify: `src/frontend/util/Config.test.ts:1-34`

**Interfaces:**
- Produces: `readAccessToken(): string | undefined`
- Produces: `storeAccessToken(token: string): void`
- Produces: `clearAuthStorage(): void`
- Produces: `RestClient.request<T>(request: ApiRequest<T>): Promise<ApiResult<T>>`
- Produces: `RestClient.setUnauthorizedHandler(handler: UnauthorizedHandler | undefined): () => void`
- Produces: `ApiError`, `isApiError(error)`, `isUnauthorizedError(error)`, `isAbortError(error)`, and `apiErrorMessage(error, fallback)`
- Consumes: `config.SERVER`
- Removes: Workstream 1's `AUTH_SESSION_EXPIRED_EVENT` export and
  `window.dispatchEvent` path; the registered handler is the only 401
  notification mechanism.
- Preserves: Workstream 6's `credentials: 'include'` default and
  `X-CSRF-Protection: 1` header on every non-GET browser request.

- [ ] **Step 1: Write failing storage tests and replace the minimal bearer-client tests with the full transport contract**

```ts
import { afterEach, describe, expect, test } from 'vitest';
import {
  ACCESS_TOKEN_KEY,
  clearAuthStorage,
  readAccessToken,
  storeAccessToken,
} from './session';

describe('access-token session storage', () => {
  afterEach(() => sessionStorage.clear());

  test('stores and reads the access token', () => {
    storeAccessToken('admin-token');
    expect(sessionStorage.getItem(ACCESS_TOKEN_KEY)).toBe('admin-token');
    expect(readAccessToken()).toBe('admin-token');
  });

  test('clears the access token', () => {
    storeAccessToken('admin-token');
    clearAuthStorage();
    expect(readAccessToken()).toBeUndefined();
  });
});
```

- [ ] **Step 2: Run the storage test and verify the red state**

Run:

```bash
npm test -- src/frontend/auth/session.test.ts
```

Expected: exit `1` with `Cannot find module './session'`.

- [ ] **Step 3: Add the storage implementation**

```ts
export const ACCESS_TOKEN_KEY = 'comics_access_token';

export const storeAccessToken = (token: string) => {
  sessionStorage.setItem(ACCESS_TOKEN_KEY, token);
};

export const readAccessToken = () => (
  sessionStorage.getItem(ACCESS_TOKEN_KEY) ?? undefined
);

export const clearAuthStorage = () => {
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
};
```

- [ ] **Step 4: Write failing client tests**

Cover these exact cases in `src/frontend/api/client.test.ts`:

```ts
const decodeName = (value: unknown) => {
  if (
    value === null
    || typeof value !== 'object'
    || !('name' in value)
    || typeof value.name !== 'string'
  ) {
    throw new TypeError('name must be a string');
  }
  return value.name;
};

test('keeps public reads anonymous', async () => {
  const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(200, { name: 'Solo' }));
  const client = createRestClient({
    baseUrl: 'https://api.example.com/',
    fetchImpl,
    getAccessToken: () => 'admin-token',
  });

  await client.request({
    path: '/comics',
    auth: 'none',
    credentials: 'omit',
    decode: decodeName,
  });

  const headers = new Headers(fetchImpl.mock.calls[0][1]?.headers);
  expect(headers.has('Authorization')).toBe(false);
  expect(fetchImpl.mock.calls[0][1]?.credentials).toBe('omit');
});

test('injects a bearer token for required requests', async () => {
  const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(200, { name: 'Solo' }));
  const client = createRestClient({
    baseUrl: 'https://api.example.com',
    fetchImpl,
    getAccessToken: () => 'admin-token',
  });

  await client.request({
    path: '/comics/1',
    method: 'PUT',
    auth: 'required',
    body: { track: true },
    decode: decodeName,
  });

  const headers = new Headers(fetchImpl.mock.calls[0][1]?.headers);
  expect(headers.get('Authorization')).toBe('Bearer admin-token');
  expect(headers.get('Content-Type')).toBe('application/json');
  expect(headers.get('X-CSRF-Protection')).toBe('1');
  expect(fetchImpl.mock.calls[0][1]?.credentials).toBe('include');
});

test('normalizes target and legacy error envelopes before decoding success', async () => {
  const decode = vi.fn(decodeName);
  const fetchImpl = vi.fn()
    .mockResolvedValueOnce(jsonResponse(409, {
      error: { code: 'comic_conflict', message: 'Title already belongs to comic 7' },
    }))
    .mockResolvedValueOnce(jsonResponse(400, { message: 'Invalid comic' }));
  const client = createRestClient({
    baseUrl: 'https://api.example.com',
    fetchImpl,
    getAccessToken: () => 'admin-token',
  });

  await expect(client.request({
    path: '/comics',
    method: 'POST',
    auth: 'required',
    body: { titles: ['Solo'] },
    decode,
  })).rejects.toMatchObject({
    status: 409,
    code: 'comic_conflict',
    message: 'Title already belongs to comic 7',
  });
  await expect(client.request({
    path: '/comics',
    method: 'POST',
    auth: 'required',
    body: { titles: [] },
    decode,
  })).rejects.toMatchObject({
    status: 400,
    code: 'HTTP_400',
    message: 'Invalid comic',
  });
  expect(decode).not.toHaveBeenCalled();
});

test('runs centralized unauthorized handling only for authenticated 401 responses', async () => {
  const onUnauthorized = vi.fn();
  const client = createRestClient({
    baseUrl: 'https://api.example.com',
    fetchImpl: vi.fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: 'expired' }))
      .mockResolvedValueOnce(jsonResponse(401, { error: 'bad login' }))
      .mockResolvedValueOnce(jsonResponse(403, { error: 'admin required' })),
    getAccessToken: () => 'expired-token',
  });
  client.setUnauthorizedHandler(onUnauthorized);

  await expect(client.request({
    path: '/comics/1',
    auth: 'required',
    decode: decodeName,
  })).rejects.toMatchObject({ status: 401 });
  await expect(client.request({
    path: '/login',
    auth: 'none',
    decode: decodeName,
  })).rejects.toMatchObject({ status: 401 });
  await expect(client.request({
    path: '/comics/1',
    auth: 'required',
    decode: decodeName,
  })).rejects.toMatchObject({ status: 403 });

  expect(onUnauthorized).toHaveBeenCalledTimes(1);
});

test('forwards AbortSignal and preserves AbortError', async () => {
  const controller = new AbortController();
  const abortError = new DOMException('Aborted', 'AbortError');
  const fetchImpl = vi.fn().mockRejectedValue(abortError);
  const client = createRestClient({
    baseUrl: 'https://api.example.com',
    fetchImpl,
    getAccessToken: () => undefined,
  });

  await expect(client.request({
    path: '/comics',
    auth: 'none',
    signal: controller.signal,
    decode: decodeName,
  })).rejects.toBe(abortError);
  expect(fetchImpl.mock.calls[0][1]?.signal).toBe(controller.signal);
});
```

The test-local helper is exact:

```ts
const jsonResponse = (status: number, body: unknown) => new Response(
  JSON.stringify(body),
  {
    status,
    headers: { 'Content-Type': 'application/json' },
  },
);
```

- [ ] **Step 5: Run the client test and verify the red state**

Run:

```bash
npm test -- src/frontend/api/client.test.ts
```

Expected: exit `1` with `Cannot find module './client'`.

- [ ] **Step 6: Implement the single typed transport**

Use these exact public types in `src/frontend/api/client.ts`:

```ts
import config from '../util/Config';
import { readAccessToken } from '../auth/session';

export type ApiAuthMode = 'none' | 'required';
export type Decoder<T> = (value: unknown) => T;
export type UnauthorizedHandler = (error: ApiError) => void;
export type FetchLike = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Promise<Response>;

export type ApiRequest<T> = {
  path: string;
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  auth: ApiAuthMode;
  body?: unknown;
  headers?: HeadersInit;
  credentials?: RequestCredentials;
  signal?: AbortSignal;
  decode: Decoder<T>;
};

export type ApiResult<T> = {
  data: T;
  status: number;
  headers: Headers;
};

export type ApiErrorKind =
  | 'http'
  | 'network'
  | 'invalid-json'
  | 'invalid-response';

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly kind: ApiErrorKind,
    public readonly payload?: unknown,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
```

Implement `RestClient` with this exact contract and ordering:

```ts
export class RestClient {
  private unauthorizedHandler?: UnauthorizedHandler;

  constructor(
    private readonly baseUrl: string,
    private readonly fetchImpl: FetchLike,
    private readonly getAccessToken: () => string | undefined,
  ) {}

  setUnauthorizedHandler(handler: UnauthorizedHandler | undefined) {
    this.unauthorizedHandler = handler;
    return () => {
      if (this.unauthorizedHandler === handler) {
        this.unauthorizedHandler = undefined;
      }
    };
  }

  async request<T>({
    path,
    method = 'GET',
    auth,
    body,
    headers: initialHeaders,
    credentials = 'include',
    signal,
    decode,
  }: ApiRequest<T>): Promise<ApiResult<T>> {
    const headers = new Headers(initialHeaders);
    headers.set('Accept', 'application/json');
    if (body !== undefined) {
      headers.set('Content-Type', 'application/json');
    }
    if (method !== 'GET') {
      headers.set('X-CSRF-Protection', '1');
    }
    if (auth === 'required') {
      const token = this.getAccessToken();
      if (token) headers.set('Authorization', `Bearer ${token}`);
    }

    let response: Response;
    try {
      response = await this.fetchImpl(
        `${this.baseUrl}${path.startsWith('/') ? path : `/${path}`}`,
        {
          method,
          headers,
          body: body === undefined ? undefined : JSON.stringify(body),
          credentials,
          signal,
        },
      );
    } catch (error) {
      if (isAbortError(error)) throw error;
      throw new ApiError(
        error instanceof Error ? error.message : 'Network request failed',
        0,
        'NETWORK_ERROR',
        'network',
      );
    }

    let payload: unknown;
    const text = await response.text();
    if (text.trim() !== '') {
      try {
        payload = JSON.parse(text) as unknown;
      } catch {
        const error = new ApiError(
          'Server returned invalid JSON',
          response.status,
          'INVALID_JSON',
          'invalid-json',
          text,
        );
        if (auth === 'required' && response.status === 401) {
          this.unauthorizedHandler?.(error);
        }
        throw error;
      }
    }

    if (!response.ok) {
      const normalized = normalizeApiError(response.status, payload);
      if (auth === 'required' && response.status === 401) {
        this.unauthorizedHandler?.(normalized);
      }
      throw normalized;
    }

    try {
      return {
        data: decode(payload),
        status: response.status,
        headers: response.headers,
      };
    } catch (error) {
      throw new ApiError(
        error instanceof Error ? error.message : 'Unexpected response payload',
        response.status,
        'INVALID_RESPONSE',
        'invalid-response',
        payload,
      );
    }
  }
}
```

`normalizeApiError` must prefer nested target fields, preserve the raw payload, and use `HTTP_<status>` as the legacy fallback code. Export these helpers:

```ts
type JsonRecord = Record<string, unknown>;

const isRecord = (value: unknown): value is JsonRecord => (
  value !== null && typeof value === 'object' && !Array.isArray(value)
);

const normalizeApiError = (status: number, payload: unknown) => {
  let code = `HTTP_${status}`;
  let message = `Request failed (${status})`;

  if (isRecord(payload)) {
    if (isRecord(payload.error)) {
      if (typeof payload.error.code === 'string') {
        code = payload.error.code;
      }
      if (typeof payload.error.message === 'string') {
        message = payload.error.message;
      }
    } else if (typeof payload.message === 'string') {
      message = payload.message;
    } else if (typeof payload.error === 'string') {
      message = payload.error;
    }
  }

  return new ApiError(message, status, code, 'http', payload);
};

export const isApiError = (error: unknown): error is ApiError => (
  error instanceof ApiError
);

export const isUnauthorizedError = (error: unknown) => (
  isApiError(error) && error.status === 401
);

export const isAbortError = (error: unknown) => (
  error instanceof DOMException
    ? error.name === 'AbortError'
    : (
      error !== null
      && typeof error === 'object'
      && 'name' in error
      && error.name === 'AbortError'
    )
);

export const apiErrorMessage = (error: unknown, fallback: string) => (
  isApiError(error) && error.message.trim() !== ''
    ? error.message
    : fallback
);

export const createRestClient = ({
  baseUrl,
  fetchImpl = (input, init) => globalThis.fetch(input, init),
  getAccessToken = readAccessToken,
}: {
  baseUrl: string;
  fetchImpl?: FetchLike;
  getAccessToken?: () => string | undefined;
}) => new RestClient(
  baseUrl.replace(/\/+$/, ''),
  fetchImpl,
  getAccessToken,
);

export const apiClient = createRestClient({ baseUrl: config.SERVER });
```

Delete Workstream 1's `AUTH_SESSION_EXPIRED_EVENT` constant and every dispatch
from this module in the same step. Do not keep the event as a second 401 path.

- [ ] **Step 7: Normalize the configured base URL**

Change `cleanURL` in `src/frontend/util/Config.ts` so an explicit server cannot create `//comics`:

```ts
const cleanURL = (value?: string) => {
  const trimmed = value?.trim().replace(/\/+$/, '');
  return trimmed === '' ? undefined : trimmed;
};
```

Add this assertion to `src/frontend/util/Config.test.ts`:

```ts
test('removes trailing slashes from an explicit API server', () => {
  expect(resolveServerURL({
    hostname: 'localhost',
    env: { VITE_API_SERVER: 'https://api.example.com///' },
  })).toBe('https://api.example.com');
});
```

- [ ] **Step 8: Run focused tests and commit**

Run:

```bash
npm test -- src/frontend/auth/session.test.ts src/frontend/api/client.test.ts src/frontend/util/Config.test.ts
```

Expected: exit `0`; all three files report `PASS`.

Commit:

```bash
git add src/frontend/auth/session.ts src/frontend/auth/session.test.ts \
  src/frontend/api/client.ts src/frontend/api/client.test.ts \
  src/frontend/util/Config.ts src/frontend/util/Config.test.ts
git commit -m "Add typed frontend REST transport"
```

### Task 2: Runtime contracts and endpoint modules

**Files:**
- Create: `src/frontend/api/contracts.ts`
- Create: `src/frontend/api/contracts.test.ts`
- Create: `src/frontend/api/comics.ts`
- Create: `src/frontend/api/comics.test.ts`
- Create: `src/frontend/api/auth.ts`
- Create: `src/frontend/api/auth.test.ts`
- Modify: `src/frontend/components/Comics/types.ts:1-77`

**Interfaces:**
- Consumes: `apiClient.request<T>()`, `Decoder<T>`, and `ApiError`
- Produces: `decodeComic(value): Comic`
- Produces: `comicsApi.list(query, signal): Promise<ComicListFetchResult>`
- Produces: `comicsApi.create(input, signal?)`, `update(id, patch, signal?)`, `delete(id, signal?)`, `merge(baseID, mergingID, signal?)`, `setCoverVisibility(id, input, signal?)`
- Produces: `comicsApi.scrapeStatus(signal?)` and `comicsApi.runScrape(signal?)`
- Produces: `authApi.login`, `signup`, deduplicated `refresh`, `logout`,
  `profile`, `updateProfile`, and `googleLoginURL`

- [ ] **Step 1: Tighten the REST model with a failing typecheck**

Replace the permissive `Comic` shape with the exact current REST success shape:

```ts
export type ComicLookupValue = number | string;

export interface Comic {
  id: number;
  titles: string[];
  cover: string;
  cover_visible: boolean;
  author: string;
  current_chap: number;
  viewed_chap: number;
  track: boolean;
  status: number;
  com_type: number;
  genres: number[];
  published_in: number[];
  description: string;
  rating: number;
  deleted: boolean;
  last_update: string;
}

export type ComicUpdate = Partial<Pick<
  Comic,
  | 'titles'
  | 'cover'
  | 'cover_visible'
  | 'author'
  | 'current_chap'
  | 'viewed_chap'
  | 'track'
  | 'status'
  | 'com_type'
  | 'genres'
  | 'published_in'
  | 'description'
  | 'rating'
  | 'deleted'
>>;

export type CreateComicInput = Omit<Comic, 'id' | 'last_update' | 'deleted'>;

export interface ComicListRequest {
  from: number;
  limit: number;
  queryFilter: string;
  onlyTracked: boolean;
  onlyUnchecked: boolean;
  ratingMin?: number;
  ratingMax?: number;
  sortBy?: string;
  sortDir?: string;
}

export interface ComicListFetchResult {
  comics: Comic[];
  pagination: Required<PaginationState>;
}
```

Remove `Comic`'s `[key: string]: unknown` index signature. Update existing test fixtures with `cover_visible`, `rating`, `deleted`, and RFC3339 `last_update`.

Run:

```bash
npm run build
```

Expected before fixture/call-site migration: exit `2` with TypeScript errors naming missing required comic fields. Keep this red state until the endpoint decoders and fixture updates compile.

- [ ] **Step 2: Add failing primitive contract tests**

`src/frontend/api/contracts.test.ts` must verify:

```ts
test('decodes exact primitive fields', () => {
  const value = recordValue({ title: 'Solo', count: 3 }, 'comic');
  expect(stringField(value, 'title', 'comic')).toBe('Solo');
  expect(numberField(value, 'count', 'comic')).toBe(3);
});

test('rejects a wrong primitive with its field path', () => {
  const value = recordValue({ title: 7 }, 'comic');
  expect(() => stringField(value, 'title', 'comic')).toThrow(
    'comic.title must be a string',
  );
});
```

Implement and export `recordValue`, `stringField`, `numberField`, `booleanField`, `stringArrayField`, and `numberArrayField` with this contract:

```ts
export type JsonRecord = Record<string, unknown>;

export const recordValue = (value: unknown, path: string): JsonRecord => {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new TypeError(`${path} must be an object`);
  }
  return value as JsonRecord;
};

export const stringField = (
  value: JsonRecord,
  field: string,
  path: string,
) => {
  if (typeof value[field] !== 'string') {
    throw new TypeError(`${path}.${field} must be a string`);
  }
  return value[field];
};

export const numberField = (
  value: JsonRecord,
  field: string,
  path: string,
) => {
  const item = value[field];
  if (typeof item !== 'number' || !Number.isInteger(item)) {
    throw new TypeError(`${path}.${field} must be an integer`);
  }
  return item;
};

export const booleanField = (
  value: JsonRecord,
  field: string,
  path: string,
) => {
  if (typeof value[field] !== 'boolean') {
    throw new TypeError(`${path}.${field} must be a boolean`);
  }
  return value[field];
};

export const stringArrayField = (
  value: JsonRecord,
  field: string,
  path: string,
) => {
  const items = value[field];
  if (!Array.isArray(items) || !items.every((item) => typeof item === 'string')) {
    throw new TypeError(`${path}.${field} must be a string array`);
  }
  return items;
};

export const numberArrayField = (
  value: JsonRecord,
  field: string,
  path: string,
) => {
  const items = value[field];
  if (
    !Array.isArray(items)
    || !items.every((item) => typeof item === 'number' && Number.isInteger(item))
  ) {
    throw new TypeError(`${path}.${field} must be an integer array`);
  }
  return items;
};
```

- [ ] **Step 3: Add failing catalog endpoint tests**

`src/frontend/api/comics.test.ts` must inject a fake `RestClient` through an exported `createComicsApi(client)` factory and assert:

1. Search titles are path-encoded and all list filters use `URLSearchParams`.
2. `list` uses `auth: 'none'` with `credentials: 'omit'`, forwards the exact signal, decodes `Comic[]`, and parses `total-comics`, `total-pages`, and `current-page`.
3. Missing/non-integer pagination headers reject with `ApiError` code `INVALID_PAGINATION`.
4. `create`, `update`, `delete`, `merge`, cover mutation, `scrapeStatus`, and `runScrape` use `auth: 'required'` with `credentials: 'include'`.
5. Create omits client-generated `last_update`.
6. A wrong `rating` type rejects with `INVALID_RESPONSE`.
7. Delete accepts the current `202` JSON number and a future empty `204` body.
8. Scrape status validates `running` and optional source/completion fields.

Use this complete fixture:

```ts
export const comicFixture: Comic = {
  id: 15816,
  titles: ['Rise from the bottom'],
  cover: 'https://example.com/cover.jpg',
  cover_visible: true,
  author: 'DemonicSca',
  current_chap: 62,
  viewed_chap: 48,
  track: true,
  status: 2,
  com_type: 3,
  genres: [0, 1],
  published_in: [0, 1],
  description: '',
  rating: 4,
  deleted: false,
  last_update: '2026-10-26T12:00:00Z',
};
```

- [ ] **Step 4: Implement the catalog API**

Use this endpoint surface:

```ts
export type CoverVisibilityInput = {
  cover: string;
  cover_visible: boolean;
};

export type ScrapeStatusResponse = {
  running: boolean;
  source?: string;
  last_completed_at?: string;
  last_completed_source?: string;
};

export type ComicsApi = {
  list(
    query: ComicListRequest,
    signal?: AbortSignal,
  ): Promise<ComicListFetchResult>;
  create(input: CreateComicInput, signal?: AbortSignal): Promise<Comic>;
  update(id: number, patch: ComicUpdate, signal?: AbortSignal): Promise<Comic>;
  delete(id: number, signal?: AbortSignal): Promise<void>;
  merge(baseID: number, mergingID: number, signal?: AbortSignal): Promise<Comic>;
  setCoverVisibility(
    id: number,
    input: CoverVisibilityInput,
    signal?: AbortSignal,
  ): Promise<Comic>;
  scrapeStatus(signal?: AbortSignal): Promise<ScrapeStatusResponse>;
  runScrape(signal?: AbortSignal): Promise<void>;
};
```

Decode every required comic field exactly:

```ts
export const decodeComic = (value: unknown): Comic => {
  const comic = recordValue(value, 'comic');
  return {
    id: numberField(comic, 'id', 'comic'),
    titles: stringArrayField(comic, 'titles', 'comic'),
    cover: stringField(comic, 'cover', 'comic'),
    cover_visible: booleanField(comic, 'cover_visible', 'comic'),
    author: stringField(comic, 'author', 'comic'),
    current_chap: numberField(comic, 'current_chap', 'comic'),
    viewed_chap: numberField(comic, 'viewed_chap', 'comic'),
    track: booleanField(comic, 'track', 'comic'),
    status: numberField(comic, 'status', 'comic'),
    com_type: numberField(comic, 'com_type', 'comic'),
    genres: numberArrayField(comic, 'genres', 'comic'),
    published_in: numberArrayField(comic, 'published_in', 'comic'),
    description: stringField(comic, 'description', 'comic'),
    rating: numberField(comic, 'rating', 'comic'),
    deleted: booleanField(comic, 'deleted', 'comic'),
    last_update: stringField(comic, 'last_update', 'comic'),
  };
};

const decodeComicList = (value: unknown) => {
  if (!Array.isArray(value)) {
    throw new TypeError('comics must be an array');
  }
  return value.map(decodeComic);
};

const decodePaginationHeaders = (headers: Headers, status: number) => {
  const integerHeader = (name: string, minimum: number) => {
    const raw = headers.get(name);
    const parsed = raw === null ? Number.NaN : Number(raw);
    if (!Number.isInteger(parsed) || parsed < minimum) {
      throw new ApiError(
        `${name} must be an integer greater than or equal to ${minimum}`,
        status,
        'INVALID_PAGINATION',
        'invalid-response',
      );
    }
    return parsed;
  };

  return {
    total: integerHeader('total-comics', 0),
    totalPages: integerHeader('total-pages', 1),
    currentPage: integerHeader('current-page', 1),
  };
};
```

The list implementation must pass its signal to the client:

```ts
const list = async (
  query: ComicListRequest,
  signal?: AbortSignal,
): Promise<ComicListFetchResult> => {
  const filter = query.queryFilter.trim();
  const path = filter === ''
    ? '/comics'
    : `/comics/search/${encodeURIComponent(filter)}`;
  const params = new URLSearchParams({
    from: String(query.from),
    limit: String(query.limit),
    only_tracked: String(query.onlyTracked),
    only_unchecked: String(query.onlyUnchecked),
  });
  if (query.ratingMin !== undefined) params.set('rating_min', String(query.ratingMin));
  if (query.ratingMax !== undefined) params.set('rating_max', String(query.ratingMax));
  if (query.sortBy) params.set('sort_by', query.sortBy);
  if (query.sortDir) params.set('sort_dir', query.sortDir);

  const result = await client.request({
    path: `${path}?${params.toString()}`,
    auth: 'none',
    credentials: 'omit',
    signal,
    decode: decodeComicList,
  });

  return {
    comics: result.data,
    pagination: decodePaginationHeaders(result.headers, result.status),
  };
};
```

Mutation methods must use the same pattern:

```ts
const update = async (
  id: number,
  patch: ComicUpdate,
  signal?: AbortSignal,
) => (
  await client.request({
    path: `/comics/${id}`,
    method: 'PUT',
    auth: 'required',
    credentials: 'include',
    body: patch,
    signal,
    decode: decodeComic,
  })
).data;
```

Export `createComicsApi` for tests and `comicsApi = createComicsApi(apiClient)` for production.

- [ ] **Step 5: Add failing auth endpoint tests**

`src/frontend/api/auth.test.ts` must cover:

- login/signup/refresh/logout use `auth: 'none'`;
- profile/update use `auth: 'required'`;
- all auth endpoints use `credentials: 'include'` for Workstream 6's hardened cookie flow;
- login/signup/refresh/logout POSTs carry `X-CSRF-Protection: 1` through
  `RestClient`;
- login requires `data.access_token`;
- refresh requires a new `data.access_token`, exposes no refresh token, and two
  concurrent calls share one request (plus one Web Lock when available);
- logout accepts an empty `204` response;
- Google login URL resolves from the same trailing-slash-free API base without
  calling `fetch`;
- profile accepts only `role: 'user' | 'admin'`;
- profile accepts boolean `active`, rejects malformed booleans, and derives admin UX only when `active === true`;
- non-2xx auth envelopes are handled by `RestClient`, not by checking a body `status` in React.

- [ ] **Step 6: Implement the auth API**

Use these exact types:

```ts
export type UserRole = 'user' | 'admin';

export type AuthData = {
  user_id: string;
  access_token: string;
};

export type AuthUser = {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type ProfileUpdate = {
  email?: string;
  username?: string;
  password?: string;
};

export type AuthApi = {
  login(input: { email: string; password: string }): Promise<AuthData>;
  signup(input: {
    email: string;
    username: string;
    password: string;
  }): Promise<AuthData>;
  refresh(): Promise<AuthData>;
  logout(): Promise<void>;
  googleLoginURL(): string;
  profile(signal?: AbortSignal): Promise<AuthUser>;
  updateProfile(input: ProfileUpdate, signal?: AbortSignal): Promise<void>;
};
```

Decode the existing success envelope `{ status, data, message }`. Require the
expected HTTP/body status (`200` login/refresh/update, `201` signup), require
`access_token` for login/signup/refresh, decode profile as a direct `AuthUser`,
and accept an empty `204` from logout. Port Workstream 6's module-level
refresh-promise deduplication and `navigator.locks.request("comics-auth-refresh",
...)` fallback into `createAuthApi`; JavaScript never reads a refresh token.
Send no `Role` header.
`googleLoginURL()` returns `${config.SERVER}/auth/google` and performs no
request.

- [ ] **Step 7: Run endpoint tests and typecheck**

Run:

```bash
npm test -- src/frontend/api/contracts.test.ts src/frontend/api/comics.test.ts src/frontend/api/auth.test.ts
npm run build
```

Expected: both commands exit `0`; the endpoint files report `PASS` and TypeScript reports no missing-field or endpoint-module errors.

- [ ] **Step 8: Commit**

```bash
git add src/frontend/api/contracts.ts src/frontend/api/contracts.test.ts \
  src/frontend/api/comics.ts src/frontend/api/comics.test.ts \
  src/frontend/api/auth.ts src/frontend/api/auth.test.ts \
  src/frontend/components/Comics/types.ts \
  src/frontend/components/Comics/Card/ComicCard.test.tsx
git commit -m "Define typed frontend API contracts"
```

### Task 3: Global auth, unauthorized handling, and app-level toasts

**Files:**
- Create: `src/frontend/context/AuthContext.test.tsx`
- Modify: `src/frontend/context/AuthContext.tsx:1-99`
- Modify: `src/index.tsx:1-32`
- Modify: `src/frontend/components/Auth/LoginPage.tsx`
- Modify: `src/frontend/components/Auth/AuthNav.tsx`
- Modify: `src/frontend/components/Auth/ProfilePage.tsx:1-79`

**Interfaces:**
- Consumes: `authApi`, `apiClient.setUnauthorizedHandler`, session helpers, and `useToast`
- Produces: `AuthContextValue` with `user`, `status`, `isAuthenticated`,
  `isAdmin`, derived `authReady`, `login`, `signup`, async `logout`, and
  `refreshProfile`
- Guarantees: one authenticated 401 clears the token and user once; a 403 or network failure does not clear the token

- [ ] **Step 1: Write failing provider tests**

Use a `ContextProbe` rendered under `MemoryRouter`, `ToastProvider`, and `AuthProvider`. Mock `authApi` and invoke the registered unauthorized callback captured from `apiClient.setUnauthorizedHandler`.

Required assertions:

```ts
expect(screen.getByTestId('is-admin')).toHaveTextContent('true');
expect(screen.getByTestId('auth-ready')).toHaveTextContent('true');
expect(readAccessToken()).toBe('admin-token');
```

For the unauthorized case:

```ts
act(() => unauthorizedHandler?.(
  new ApiError('Authentication expired', 401, 'unauthorized', 'http'),
));

expect(readAccessToken()).toBeUndefined();
expect(screen.getByTestId('is-authenticated')).toHaveTextContent('false');
expect(screen.getByRole('alert')).toHaveTextContent('Session expired');
```

Call the same callback twice and assert only one `Session expired` toast exists. Trigger a `403` rejection from `authApi.profile()` and assert the stored token remains.

Also preserve Workstream 6's browser-session tests: no stored token performs
one deduplicated `authApi.refresh()` then loads profile; an invalid stored token
does the same once; React Strict Mode does not rotate twice; logout clears
local state before the pending `authApi.logout()` resolves.

- [ ] **Step 2: Run the provider test and verify the red state**

Run:

```bash
npm test -- src/frontend/context/AuthContext.test.tsx
```

Expected: exit `1`; `isAdmin`/`authReady` are absent and the provider has no centralized unauthorized registration.

- [ ] **Step 3: Mount toasts above auth and routes**

Change the provider order in `src/index.tsx`:

```tsx
<StrictMode>
  <Router basename={appBasename}>
    <ToastProvider>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<ComicsMainPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Routes>
      </AuthProvider>
    </ToastProvider>
  </Router>
</StrictMode>
```

Remove the page-local `ToastProvider` from `ComicsMainPage` in Task 4. This provider order lets auth failures notify every route, including profile and catalog mutations.

- [ ] **Step 4: Replace legacy auth helpers in the context**

Expose this exact context shape:

```ts
export type AuthStatus = 'loading' | 'authenticated' | 'anonymous';

type AuthContextValue = {
  user: AuthUser | null;
  status: AuthStatus;
  isAuthenticated: boolean;
  isAdmin: boolean;
  authReady: boolean;
  login: (email: string, password: string) => Promise<string | null>;
  signup: (
    email: string,
    username: string,
    password: string,
  ) => Promise<string | null>;
  logout: () => Promise<void>;
  refreshProfile: () => Promise<void>;
};
```

Implementation rules:

1. Preserve Workstream 6's `status` state, derive
   `authReady = status !== 'loading'`, and derive `isAdmin` exactly as
   `user?.active === true && user.role === 'admin'`.
2. Register `apiClient.setUnauthorizedHandler` after bootstrap completes and
   unregister it in cleanup, so an expected stale access token can refresh
   without first showing a session-expired toast.
3. Remove Workstream 1's `window.addEventListener` /
   `window.removeEventListener` session-expired listener in the same change.
4. In the handler, return immediately if storage is already empty; otherwise clear token/user and show:

```ts
toast.error({
  title: 'Session expired',
  description: 'Sign in again before making catalog changes.',
});
```

5. On mount, try profile with the stored access token. If the token is missing
   or profile returns `401`, clear local token state, call the deduplicated
   `authApi.refresh()` once, store its access token, and load profile once.
   Mark `authReady` in `finally`; a failed refresh leaves the context anonymous.
6. `login` and `signup` call `authApi`, store only `access_token`, then await `refreshProfile`.
7. Catch `ApiError` and return `apiErrorMessage(error, 'Login failed')` or the signup equivalent.
8. After bootstrap, a profile `401` is handled centrally. Other profile
   failures leave the token intact, set `user` to `null`, and complete
   initialization.
9. `logout` clears storage and state synchronously, then awaits
   `authApi.logout()`. Swallow its network failure so local logout remains
   reliable and do not show an unauthorized toast.

- [ ] **Step 5: Migrate auth components to the final context/API**

`ProfilePage` must no longer receive/pass `accessToken`. Replace its submit call with:

```ts
try {
  await authApi.updateProfile({
    email,
    username,
    password: password || undefined,
  });
  setPassword('');
  setMessage('Profile updated');
  await refreshProfile();
} catch (error) {
  setMessage(apiErrorMessage(error, 'Profile update failed'));
}
```

`LoginPage` imports `authApi.googleLoginURL` instead of
`AuthHelpers.googleLoginURL` and keeps Workstream 6's OAuth error UX.
`AuthNav` consumes `status`/`logout` only from context and launches logout with
`void logout()`; neither component imports `AuthHelpers`.

- [ ] **Step 6: Verify global toast behavior**

Run the existing `ToastProvider.test.tsx` stacking/timer tests after moving the provider, plus the AuthContext assertion that duplicate unauthorized callbacks create one toast. Do not add deduplication to general success/error toasts.

- [ ] **Step 7: Run focused tests and commit**

Run:

```bash
npm test -- src/frontend/context/AuthContext.test.tsx \
  src/frontend/components/Toast/ToastProvider.test.tsx
```

Expected: exit `0`; both files report `PASS`.

Commit:

```bash
git add src/frontend/context/AuthContext.tsx \
  src/frontend/context/AuthContext.test.tsx \
  src/frontend/components/Auth/LoginPage.tsx \
  src/frontend/components/Auth/AuthNav.tsx \
  src/frontend/components/Auth/ProfilePage.tsx \
  src/index.tsx
git commit -m "Centralize frontend session expiry handling"
```

### Task 4: Public list reads, aborts, invalidation, and resize pagination

**Files:**
- Modify: `src/frontend/components/Comics/MainPage/MainPage.tsx:1-266`
- Modify: `src/frontend/components/Comics/MainPage/MainPage.test.tsx:1-173`
- Modify: `src/frontend/components/Comics/Card/ComicsList.tsx:1-46`

**Interfaces:**
- Consumes: `comicsApi.list(query, signal)`, `isAbortError`, and `useAuth().isAdmin`
- Produces: one stable `invalidateCatalog(): void` callback passed to all mutation children
- Produces: `ComicsList` props `canMutate: boolean` and `onMutationSuccess: () => void`

- [ ] **Step 1: Rewrite MainPage mocks and add failing tests**

Mock `comicsApi.list`, not `ServerHelpers.dataFetch`. Keep existing debounce, visibility, and non-overlap tests.

Add these tests:

```ts
test('passes an AbortSignal to the public list client and aborts stale reads', async () => {
  let firstSignal: AbortSignal | undefined;
  listMock
    .mockImplementationOnce((_query, signal) => {
      firstSignal = signal;
      return new Promise<never>(() => undefined);
    })
    .mockResolvedValue(emptyComicsResult);

  renderPage();
  fireEvent.click(screen.getByRole('button', { name: /tracked/i }));
  await flushPromises();

  expect(firstSignal?.aborted).toBe(true);
  expect(listMock).toHaveBeenCalledTimes(2);
});

test('resets offset when resize changes the page limit', async () => {
  setViewport(1700, 900);
  renderPage('/?from=18');
  await flushPromises();
  expect(listMock).toHaveBeenCalledWith(
    expect.objectContaining({ from: 18, limit: 9 }),
    expect.any(AbortSignal),
  );

  setViewport(1200, 900);
  act(() => {
    window.dispatchEvent(new Event('resize'));
    vi.runOnlyPendingTimers();
  });
  await flushPromises();

  expect(listMock).toHaveBeenLastCalledWith(
    expect.objectContaining({ from: 0, limit: 6 }),
    expect.any(AbortSignal),
  );
});
```

`setViewport` must redefine `window.innerWidth` and `window.innerHeight` as configurable values. Update `renderPage(initialEntry = '/')` so the memory router can start at a nonzero offset.

Because `AuthProvider` now consumes the global toast context, use this exact test wrapper:

```tsx
const renderPage = (initialEntry = '/') => render(
  <MemoryRouter
    basename={appBasename}
    initialEntries={[
      appBasename ? `${appBasename}${initialEntry}` : initialEntry,
    ]}
  >
    <ToastProvider>
      <AuthProvider>
        <ComicsMainPage />
      </AuthProvider>
    </ToastProvider>
  </MemoryRouter>
);
```

Add a test that invokes a mocked child `onMutationSuccess` and asserts `listMock` runs again regardless of filter combination.

- [ ] **Step 2: Run MainPage tests and verify the red state**

Run:

```bash
npm test -- src/frontend/components/Comics/MainPage/MainPage.test.tsx
```

Expected: exit `1`; the page still imports `dataFetch`, mutation invalidation is filter-conditional, and resize preserves `from`.

- [ ] **Step 3: Migrate list reads and remove debug logging**

Replace `dataFetch` with:

```ts
const result = await comicsApi.list({
  from,
  limit,
  queryFilter,
  onlyTracked,
  onlyUnchecked,
  ratingMin,
  ratingMax,
  sortBy,
  sortDir,
}, controller.signal);
```

Keep the request-id stale-response guard. In `catch`, return for aborts and stale requests, map `ApiError.kind === 'http' || 'invalid-json' || 'invalid-response'` to the server load message, and map network failures to the network load message. Remove `console.debug`.

- [ ] **Step 4: Make invalidation unconditional**

Delete `handleFilteredMutationSuccess`. Rename `requestRefresh` to:

```ts
const invalidateCatalog = useCallback(() => {
  startTransition(() => {
    setRefreshTick((value) => value + 1);
  });
}, []);
```

Pass `invalidateCatalog` to create, merge, scrape, and `ComicsList`. A successful checkout/delete/track/rate/edit/cover update must use the same callback in Tasks 5–6.

- [ ] **Step 5: Reset offset atomically when the limit changes**

Use a ref so the native resize handler compares against the latest limit:

```ts
const limitRef = useRef(limit);

const applyPageLimit = useCallback((nextLimit: number) => {
  if (limitRef.current === nextLimit) return;
  limitRef.current = nextLimit;
  setLimit(nextLimit);
  setSearchParams((current) => {
    const next = new URLSearchParams(current);
    next.delete('from');
    return next;
  }, { replace: true });
}, [setSearchParams]);
```

Call `applyPageLimit(calculatePageLimit(window.innerWidth, window.innerHeight))` inside the existing animation-frame resize and nav-height handler. Do not reset offset when the computed limit is unchanged.

- [ ] **Step 6: Gate list and rail mutation UI by validated admin role**

At the top of `ComicsMainPage`, read:

```ts
const { isAdmin } = useAuth();
```

Pass `canMutate={isAdmin}` to `ComicsList`. Render the floating rail only for admins:

```tsx
{isAdmin ? (
  <FloatingActionRail>
    <ScrapeButton onSuccess={invalidateCatalog} />
    <MergeComic onSuccess={invalidateCatalog} />
    <CreateComic onSuccess={invalidateCatalog} />
  </FloatingActionRail>
) : null}
```

Remove the page-local `ToastProvider` wrapper because `src/index.tsx` now owns it.

- [ ] **Step 7: Run MainPage tests and commit**

Run:

```bash
npm test -- src/frontend/components/Comics/MainPage/MainPage.test.tsx \
  src/frontend/components/Comics/utils.test.ts
```

Expected: exit `0`; stale requests abort, mutation callbacks refetch, and the resize case ends at `from: 0`.

Commit:

```bash
git add src/frontend/components/Comics/MainPage/MainPage.tsx \
  src/frontend/components/Comics/MainPage/MainPage.test.tsx \
  src/frontend/components/Comics/Card/ComicsList.tsx
git commit -m "Reconcile catalog reads and pagination"
```

### Task 5: Card mutations and list/card consistency

**Files:**
- Create: `src/frontend/hooks/useComicActions.test.tsx`
- Modify: `src/frontend/hooks/useComicActions.ts:1-50`
- Modify: `src/frontend/components/Comics/Card/ComicCard.tsx:1-220`
- Modify: `src/frontend/components/Comics/Card/ComicCard.test.tsx:1-122`
- Modify: `src/frontend/components/Comics/Card/ComicCardContext.tsx:1-38`
- Modify: `src/frontend/components/Comics/Card/ComicCover.tsx:1-90`
- Modify: `src/frontend/components/Comics/Card/ComicsList.tsx:1-46`
- Modify: `src/frontend/components/Comics/Edition/EditComic.tsx:1-103`

**Interfaces:**
- Consumes: `comicsApi.update/delete/setCoverVisibility`, `apiErrorMessage`, `isAbortError`, and `useToast`
- Produces: `ComicCardProps { comic, canMutate, onMutationSuccess }`
- Produces: `ComicCardContextValue.onMutationSuccess`
- Guarantees: card state changes only from decoded successful comic payloads; every success invalidates the parent list

- [ ] **Step 1: Add failing action-order tests**

In `useComicActions.test.tsx`, render a hook harness and mock `comicsApi`.

Required failure assertion:

```ts
updateMock.mockRejectedValue(
  new ApiError('Update denied', 403, 'forbidden', 'http'),
);

await act(async () => {
  await result.current.handleTrackToggle();
});

expect(setComic).not.toHaveBeenCalled();
expect(onMutationSuccess).not.toHaveBeenCalled();
expect(screen.getByRole('alert')).toHaveTextContent('Update denied');
```

Required success ordering:

```ts
updateMock.mockResolvedValue({ ...comicFixture, track: false });

await act(async () => {
  await result.current.handleTrackToggle();
});

expect(setComic).toHaveBeenCalledWith({ ...comicFixture, track: false });
expect(onMutationSuccess).toHaveBeenCalledTimes(1);
expect(screen.getByText('Tracking updated')).toBeInTheDocument();
```

Repeat the state-before-success assertion for checkout and delete. Delete calls `setDel(true)` only after the accepted response.

- [ ] **Step 2: Add failing card permission and invalidation tests**

Update `ComicCard.test.tsx` to assert:

1. `canMutate={false}` renders title/metadata but no edit, delete, rating buttons, checkout, or track button.
2. A public cover image failure shows the fallback and does not call `comicsApi.setCoverVisibility`.
3. `canMutate={true}` shows controls.
4. A successful rating response replaces local card state, shows `Rating updated`, and calls `onMutationSuccess`.
5. A failed rating response leaves the pressed rating unchanged and shows the server message.
6. Rerendering with refreshed parent props still replaces local card state.

- [ ] **Step 3: Run card/action tests and verify the red state**

Run:

```bash
npm test -- src/frontend/hooks/useComicActions.test.tsx \
  src/frontend/components/Comics/Card/ComicCard.test.tsx
```

Expected: exit `1`; card actions still use legacy helpers and controls are not permission-gated.

- [ ] **Step 4: Make the action hook async and typed**

Use this argument shape:

```ts
type UseComicActionsArgs = {
  comic: Comic;
  setComic: Dispatch<SetStateAction<Comic>>;
  setViewedChap: Dispatch<SetStateAction<number>>;
  setCheck: Dispatch<SetStateAction<boolean>>;
  setDel: Dispatch<SetStateAction<boolean>>;
  onMutationSuccess: () => void;
};
```

Each handler follows this exact order:

```ts
const handleTrackToggle = useCallback(async () => {
  try {
    const updated = await comicsApi.update(comic.id, {
      track: !comic.track,
    });
    setComic(updated);
    onMutationSuccess();
    toast.success({
      title: 'Tracking updated',
      description: updated.track
        ? `${updated.titles[0]} is now tracked.`
        : `${updated.titles[0]} is no longer tracked.`,
    });
  } catch (error) {
    if (isAbortError(error) || isUnauthorizedError(error)) return;
    toast.error({
      title: 'Tracking update failed',
      description: apiErrorMessage(error, 'Unable to change tracking.'),
    });
  }
}, [comic.id, comic.track, onMutationSuccess, setComic, toast]);
```

Checkout uses `{ viewed_chap: comic.current_chap }`, then sets the returned comic, `viewedChap`, and `check`. Delete waits for `comicsApi.delete`, then sets `del`, invalidates, and toasts.

- [ ] **Step 5: Gate and reconcile ComicCard**

Adopt these props:

```ts
interface ComicCardProps {
  comic: Comic;
  canMutate: boolean;
  onMutationSuccess: () => void;
}
```

Keep local card state only to reflect the decoded mutation response immediately. Continue the existing `useEffect` that replaces it whenever parent props refresh.

Rating must use:

```ts
const handleRate = async (next: number) => {
  if (!canMutate || comic.rating === next) return;
  try {
    const updated = await comicsApi.update(id, { rating: next });
    setComic(updated);
    onMutationSuccess();
    toast.success({
      title: 'Rating updated',
      description: `${updated.titles[0]} is rated ${updated.rating}/5.`,
    });
  } catch (error) {
    if (isAbortError(error) || isUnauthorizedError(error)) return;
    toast.error({
      title: 'Rating update failed',
      description: apiErrorMessage(error, 'Unable to save the rating.'),
    });
  }
};
```

Render edit/delete/rating buttons and footer actions only when `canMutate`. Public users still see a non-interactive rating display.

- [ ] **Step 6: Propagate mutation permission and invalidation**

`ComicsList` passes both props to every card. `ComicCardContext` adds:

```ts
onMutationSuccess: () => void;
```

`EditComic` calls `comicsApi.update`, then and only then:

```ts
setComic(updated);
setViewedChap(updated.viewed_chap);
handleCloseEditComicModal();
toast.success({
  title: 'Comic updated',
  description: `${db_classes.com_type[updated.com_type]} ${updated.titles[0]} is saved.`,
});
onMutationSuccess();
return true;
```

On failure, keep the modal open, preserve the draft, show an error toast, and return `false`.

- [ ] **Step 7: Make cover reporting admin-only and typed**

Add `canReportFailure: boolean` and `onMutationSuccess` to `ComicCover`. Always switch to the local fallback on image error, but call `comicsApi.setCoverVisibility` only when `canReportFailure` is true. On success replace the card with the returned comic through a context callback and invalidate the list. On non-401 failure show `Cover status not saved`; remove `console.debug`.

Use this exact prop contract so the public card can omit overlay children and the updated payload reaches the card owner:

```ts
interface ComicCoverProps {
  comic: Comic;
  children?: ReactNode;
  canReportFailure: boolean;
  onComicUpdated: (comic: Comic) => void;
  onMutationSuccess: () => void;
}
```

Pass `onComicUpdated={setComic}` from `ComicCard`. Extend `comicCardPropsEqual` to compare `canMutate` and `onMutationSuccess`; otherwise login/logout or a changed invalidation callback can be hidden by the custom memo comparator.

- [ ] **Step 8: Run card tests and commit**

Run:

```bash
npm test -- src/frontend/hooks/useComicActions.test.tsx \
  src/frontend/components/Comics/Card/ComicCard.test.tsx \
  src/frontend/components/Comics/Edition/EditComic.test.tsx
```

Expected: exit `0`; failed responses do not update state or invalidate, while every decoded success does both.

Commit:

```bash
git add src/frontend/hooks/useComicActions.ts \
  src/frontend/hooks/useComicActions.test.tsx \
  src/frontend/components/Comics/Card/ComicCard.tsx \
  src/frontend/components/Comics/Card/ComicCard.test.tsx \
  src/frontend/components/Comics/Card/ComicCardContext.tsx \
  src/frontend/components/Comics/Card/ComicCover.tsx \
  src/frontend/components/Comics/Card/ComicsList.tsx \
  src/frontend/components/Comics/Edition/EditComic.tsx \
  src/frontend/components/Comics/Edition/EditComic.test.tsx
git commit -m "Reconcile card mutations with the catalog"
```

### Task 6: Create, merge, and scrape feedback

**Files:**
- Create: `src/frontend/components/Comics/Edition/CreateComic.test.tsx`
- Modify: `src/frontend/components/Comics/Edition/CreateComic.tsx:1-96`
- Modify: `src/frontend/components/Comics/Edition/MergeComic.tsx:1-118`
- Modify: `src/frontend/components/Comics/Edition/MergeComic.test.tsx:1-32`
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.tsx:1-150`
- Modify: `src/frontend/components/Comics/Edition/ScrapeButton.test.tsx`

**Interfaces:**
- Consumes: `comicsApi.create/merge/scrapeStatus/runScrape`
- Produces: success toasts + `onSuccess` invalidation for all three workflows
- Guarantees: failed forms remain open with their values; polling is abortable; aborts do not toast

- [ ] **Step 1: Add failing create tests**

`CreateComic.test.tsx` must assert:

- the request maps `title` to `titles: [title]`;
- the request omits `last_update`;
- success closes the modal, clears the form, shows `Comic created`, and calls `onSuccess`;
- `403` keeps the modal and draft open, shows `Create failed`, and does not call `onSuccess`;
- malformed successful JSON is presented as a create failure.

- [ ] **Step 2: Expand merge tests before implementation**

Keep the existing draft-reopen test and add:

```ts
test('keeps the merge draft and toasts when the API rejects it', async () => {
  mergeMock.mockRejectedValue(
    new ApiError('Comic 22 does not exist', 404, 'not_found', 'http'),
  );
  const onSuccess = vi.fn();
  renderMerge({ onSuccess });

  fireEvent.change(screen.getByLabelText(/baseid/i), {
    target: { value: '11' },
  });
  fireEvent.change(screen.getByLabelText(/mergingid/i), {
    target: { value: '22' },
  });
  fireEvent.click(screen.getByRole('button', { name: /^merge$/i }));

  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Comic 22 does not exist',
  );
  expect(onSuccess).not.toHaveBeenCalled();
  expect(screen.getByLabelText(/baseid/i)).toHaveValue(11);
});
```

On success, assert `Comics merged`, modal close, draft reset, and one invalidation.

- [ ] **Step 3: Rewrite scrape tests around typed errors and aborts**

Test:

1. status returns the validated response;
2. `409` from `runScrape` shows `Scrape already running` and source from `ApiError.payload`;
3. `500` shows `Scrape failed` with the normalized message;
4. success shows `Catalog refreshed` and invalidates once;
5. status polling passes a signal and cleanup aborts it;
6. an `AbortError` produces no error toast.

- [ ] **Step 4: Run edition tests and verify the red state**

Run:

```bash
npm test -- src/frontend/components/Comics/Edition/CreateComic.test.tsx \
  src/frontend/components/Comics/Edition/MergeComic.test.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
```

Expected: exit `1`; raw component fetches and legacy string-return helpers remain.

- [ ] **Step 5: Migrate create**

Build the request without a client timestamp:

```ts
const input: CreateComicInput = {
  titles: [data.title],
  track: data.track,
  current_chap: data.current_chap,
  viewed_chap: data.viewed_chap,
  cover: data.cover,
  cover_visible: true,
  description: data.description,
  author: data.author,
  com_type: data.com_type,
  status: data.status,
  published_in: data.published_in,
  genres: data.genres,
  rating: 0,
};
```

Await `comicsApi.create(input)`. Only after it returns: close, toast, invalidate, and return `true`. Return `false` silently for aborts and authenticated `401` failures; the latter already emits the centralized session-expired toast. For other failures use `apiErrorMessage`, preserve the modal/draft, and return `false`. Remove all request/payload logging.

- [ ] **Step 6: Migrate merge**

Replace `mergeComic` with `comicsApi.merge(baseID, mergingID)`. On a successful decoded comic, reset and close before success toast/invalidation. On failure, set the existing inline `submissionError` and also show:

```ts
toast.error({
  title: 'Merge failed',
  description: apiErrorMessage(
    error,
    `Unable to merge ${data.baseID} and ${data.mergingID}.`,
  ),
});
```

Preserve the draft and keep the modal open for non-401 failures. Return `false` without an additional merge toast for authenticated `401`; admin-control unmounting closes the modal after centralized logout.

- [ ] **Step 7: Migrate scrape and abort status polling**

Replace both exported raw helpers with `comicsApi`. Every mount/status poll creates an `AbortController`, passes its signal, and aborts it in effect cleanup. Maintain one controller ref for the current status request so a new poll aborts the previous poll.

For a busy response:

```ts
if (isApiError(error) && error.status === 409) {
  const source = (
    error.payload !== null
    && typeof error.payload === 'object'
    && 'source' in error.payload
    && typeof error.payload.source === 'string'
  )
    ? error.payload.source
    : undefined;
  setRemoteRunning(true);
  setShowLoader(true);
  toast.info({
    title: 'Scrape already running',
    description: source
      ? `A ${source} scrape is in progress. Try again later.`
      : 'Another scrape is in progress. Try again later.',
  });
  return;
}
```

On success, show `Catalog refreshed` and invalidate. On abort or authenticated `401`, return silently after the centralized handler runs. On other errors, use the normalized message in the failure toast.

- [ ] **Step 8: Run edition tests and commit**

Run:

```bash
npm test -- src/frontend/components/Comics/Edition/CreateComic.test.tsx \
  src/frontend/components/Comics/Edition/EditComic.test.tsx \
  src/frontend/components/Comics/Edition/MergeComic.test.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
```

Expected: exit `0`; all four workflow files report `PASS`.

Commit:

```bash
git add src/frontend/components/Comics/Edition/CreateComic.tsx \
  src/frontend/components/Comics/Edition/CreateComic.test.tsx \
  src/frontend/components/Comics/Edition/MergeComic.tsx \
  src/frontend/components/Comics/Edition/MergeComic.test.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.tsx \
  src/frontend/components/Comics/Edition/ScrapeButton.test.tsx
git commit -m "Standardize catalog mutation feedback"
```

### Task 7: Remove parallel clients and debug logging

**Files:**
- Modify: every migrated import listed in Tasks 3–6
- Delete: `src/frontend/util/ServerHelpers.ts`
- Delete: `src/frontend/util/AuthHelpers.ts`
- Delete: `src/frontend/util/AuthHelpers.test.ts`

**Interfaces:**
- Consumes: only `api/*`, `auth/session`, and `AuthContext` for REST/session behavior
- Produces: invariant that `src/frontend/api/client.ts` is the only production file containing `fetch(`

- [ ] **Step 1: Prove legacy dependencies remain before cleanup**

Run:

```bash
rg "ServerHelpers|AuthHelpers|console\.debug|\bfetch\(" src/frontend \
  --glob '*.{ts,tsx}' --glob '!**/*.test.*'
```

Expected before deletion: matches identify the remaining migration points. No match may be ignored.

- [ ] **Step 2: Confirm auth page boundaries**

`LoginPage` and `SignupPage` continue calling context methods; they must not import API/session modules directly. Keep their current user-visible inline error behavior, sourced from normalized context messages.

- [ ] **Step 3: Delete the legacy helpers**

Delete `ServerHelpers.ts` and `AuthHelpers.ts` after every call site uses
`comicsApi`, `authApi`, or `auth/session`. Before deleting
`AuthHelpers.test.ts`, port every Workstream 6 refresh-deduplication, Web Lock,
credential, CSRF, cookie-only refresh, and server-logout assertion into
`api/auth.test.ts` or `api/client.test.ts`. Do not leave compatibility
re-exports; that would preserve two client surfaces.

- [ ] **Step 4: Remove request debug logs**

Remove all `console.debug` calls from MainPage, card/cover, and edition workflows. Do not replace them with `console.log`. Errors are represented by state/toasts and backend logs.

- [ ] **Step 5: Verify the one-client invariant**

Run:

```bash
test "$(rg -l '\bfetch\(' src/frontend --glob '*.{ts,tsx}' \
  --glob '!**/*.test.*' | sort)" = "src/frontend/api/client.ts"
! rg "ServerHelpers|AuthHelpers|console\.debug" src/frontend \
  --glob '*.{ts,tsx}' --glob '!**/*.test.*'
```

Expected: both commands exit `0`; the only production fetch owner is `src/frontend/api/client.ts`, and there are no legacy-helper/debug matches.

- [ ] **Step 6: Run the complete frontend regression gate**

Run:

```bash
make check-front
```

Expected:

```text
npm run test
Test Files  passed
Tests       passed
npm run build
build completed successfully
```

Both commands must exit `0` with no TypeScript errors or React test-environment warnings about updates outside `act`.

- [ ] **Step 7: Commit**

```bash
git add -u src/frontend/util/ServerHelpers.ts \
  src/frontend/util/AuthHelpers.ts \
  src/frontend/util/AuthHelpers.test.ts
git commit -m "Remove legacy frontend request helpers"
```

### Task 8: Browser E2E, live smoke, and CI rollout

**Files:**
- Create: `playwright.config.ts`
- Create: `tests/e2e/frontend-api-consistency.spec.ts`
- Create: `scripts/smoke/frontend-api-consistency.mjs`
- Modify: `package.json:15-39`
- Modify: `package-lock.json`
- Modify: `Makefile:1-10,126-145`
- Modify: `.github/workflows/ci.yml:32-45`

**Interfaces:**
- Consumes: production Vite app and merged Workstream 1/6 API contracts
- Produces: `npm run test:e2e`, `npm run smoke:frontend-api`, `make front-e2e`
- Guarantees: deterministic mocked browser coverage in CI and an opt-in real admin mutation round trip

- [ ] **Step 1: Install and pin Playwright**

Run:

```bash
npm install --save-dev @playwright/test@1.64.1
```

Expected: exit `0`; `package.json` contains `"@playwright/test": "^1.64.1"` and `package-lock.json` is updated.

Add scripts:

```json
{
  "test:e2e": "playwright test",
  "smoke:frontend-api": "node scripts/smoke/frontend-api-consistency.mjs"
}
```

- [ ] **Step 2: Add the Playwright configuration**

```ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: 'http://127.0.0.1:3000/comics/',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run start',
    url: 'http://127.0.0.1:3000/comics/',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
```

- [ ] **Step 3: Add deterministic browser scenarios**

Use this fixture and route setup at the top of `frontend-api-consistency.spec.ts`:

```ts
import { expect, test, type Page, type Route } from '@playwright/test';

const API_ORIGIN = 'https://localhost:8081';
const ACCESS_TOKEN_KEY = 'comics_access_token';

const comicFixture = {
  id: 15816,
  titles: ['Rise from the bottom'],
  cover: 'https://example.com/cover.jpg',
  cover_visible: true,
  author: 'DemonicSca',
  current_chap: 62,
  viewed_chap: 48,
  track: true,
  status: 2,
  com_type: 3,
  genres: [0, 1],
  published_in: [0, 1],
  description: '',
  rating: 4,
  deleted: false,
  last_update: '2026-10-26T12:00:00Z',
};

const adminProfile = {
  id: '11111111-1111-4111-8111-111111111111',
  username: 'admin',
  email: 'admin@example.com',
  role: 'admin',
  active: true,
  created_at: '2026-10-26T12:00:00Z',
  updated_at: '2026-10-26T12:00:00Z',
};

const fulfillList = (route: Route, comics = [comicFixture]) => route.fulfill({
  status: 200,
  contentType: 'application/json',
  headers: {
    'total-comics': String(comics.length),
    'total-pages': '1',
    'current-page': '1',
    'access-control-expose-headers': 'total-comics,total-pages,current-page',
  },
  body: JSON.stringify(comics),
});

const installAdminSession = async (page: Page) => {
  await page.addInitScript(({ key }) => {
    sessionStorage.setItem(key, 'admin-token');
  }, { key: ACCESS_TOKEN_KEY });
  await page.route(`${API_ORIGIN}/protected/profile`, async (route) => {
    expect(route.request().headers().authorization).toBe('Bearer admin-token');
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(adminProfile),
    });
  });
};
```

Implement the five browser tests exactly:

```ts
test('public catalog reads stay bearer-free and read-only', async ({ page }) => {
  await page.route('**/comics?**', async (route) => {
    expect(route.request().headers().authorization).toBeUndefined();
    expect(route.request().headers().cookie).toBeUndefined();
    await fulfillList(route);
  });

  await page.goto('/comics/');

  await expect(page.getByText('Rise from the bottom')).toBeVisible();
  await expect(page.getByRole('button', { name: /delete rise from the bottom/i })).toHaveCount(0);
  await expect(page.getByRole('button', { name: /edit/i })).toHaveCount(0);
  await expect(page.getByRole('button', { name: /open comic utilities/i })).toHaveCount(0);
});

test('admin create injects bearer and invalidates the list', async ({ page }) => {
  await installAdminSession(page);
  let listReads = 0;
  await page.route('**/comics?**', async (route) => {
    expect(route.request().headers().authorization).toBeUndefined();
    expect(route.request().headers().cookie).toBeUndefined();
    listReads += 1;
    await fulfillList(route);
  });
  await page.route(`${API_ORIGIN}/comics`, async (route) => {
    expect(route.request().method()).toBe('POST');
    expect(route.request().headers().authorization).toBe('Bearer admin-token');
    const body = route.request().postDataJSON();
    expect(body.titles).toEqual(['Smoke comic']);
    expect(body.last_update).toBeUndefined();
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        ...comicFixture,
        id: 15817,
        titles: ['Smoke comic'],
      }),
    });
  });

  await page.goto('/comics/');
  await page.getByRole('button', { name: /open comic utilities/i }).click();
  await page.getByRole('button', { name: /create comic/i }).click();
  await page.getByLabel('Title').fill('Smoke comic');
  await page.getByRole('button', { name: 'Create', exact: true }).click();

  await expect(page.getByText('Comic created')).toBeVisible();
  await expect.poll(() => listReads).toBe(2);
});

test('authenticated 401 clears the session once', async ({ page }) => {
  await installAdminSession(page);
  await page.route('**/comics?**', (route) => fulfillList(route));
  await page.route(`${API_ORIGIN}/comics`, async (route) => {
    expect(route.request().headers().authorization).toBe('Bearer admin-token');
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({
        error: {
          code: 'unauthorized',
          message: 'Authentication expired',
        },
      }),
    });
  });

  await page.goto('/comics/');
  await page.getByRole('button', { name: /open comic utilities/i }).click();
  await page.getByRole('button', { name: /create comic/i }).click();
  await page.getByLabel('Title').fill('Smoke comic');
  await page.getByRole('button', { name: 'Create', exact: true }).click();

  await expect(page.getByText('Session expired')).toHaveCount(1);
  await expect(page.getByText('Create failed')).toHaveCount(0);
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: /open comic utilities/i })).toHaveCount(0);
  await expect.poll(() => page.evaluate(
    (key) => sessionStorage.getItem(key),
    ACCESS_TOKEN_KEY,
  )).toBeNull();
});

test('failed card update keeps card and list state unchanged', async ({ page }) => {
  await installAdminSession(page);
  let listReads = 0;
  await page.route('**/comics?**', async (route) => {
    listReads += 1;
    await fulfillList(route);
  });
  await page.route(`${API_ORIGIN}/comics/15816`, async (route) => {
    expect(route.request().method()).toBe('PUT');
    expect(route.request().headers().authorization).toBe('Bearer admin-token');
    await route.fulfill({
      status: 403,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'Admin access required' }),
    });
  });

  await page.goto('/comics/');
  const ratingFive = page.getByRole('button', { name: 'Set rating to 5 of 5' });
  await expect(ratingFive).toHaveAttribute('aria-pressed', 'false');
  await ratingFive.click();

  await expect(page.getByText('Admin access required')).toBeVisible();
  await expect(ratingFive).toHaveAttribute('aria-pressed', 'false');
  expect(listReads).toBe(1);
});

test('resize resets offset when the page limit changes', async ({ page }) => {
  await page.setViewportSize({ width: 1700, height: 900 });
  const listURLs: string[] = [];
  await page.route('**/comics?**', async (route) => {
    expect(route.request().headers().authorization).toBeUndefined();
    expect(route.request().headers().cookie).toBeUndefined();
    listURLs.push(route.request().url());
    await fulfillList(route);
  });

  await page.goto('/comics/?from=18');
  await expect.poll(() => listURLs.some(
    (url) => url.includes('from=18') && url.includes('limit=9'),
  )).toBe(true);

  await page.setViewportSize({ width: 1200, height: 900 });
  await expect.poll(() => listURLs.some(
    (url) => url.includes('from=0') && url.includes('limit=6'),
  )).toBe(true);
});
```

- [ ] **Step 4: Run browser tests locally**

Run:

```bash
npx playwright install chromium
npm run test:e2e
```

Expected:

```text
Running 5 tests using 1 worker
5 passed
```

- [ ] **Step 5: Add the live API smoke script**

The script requires `COMICS_SMOKE_API`, `COMICS_SMOKE_ADMIN_EMAIL`, and `COMICS_SMOKE_ADMIN_PASSWORD`; it exits before any request when one is missing. Use this implementation:

```js
const requireEnv = (name) => {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
};

const baseURL = requireEnv('COMICS_SMOKE_API').replace(/\/+$/, '');
const email = requireEnv('COMICS_SMOKE_ADMIN_EMAIL');
const password = requireEnv('COMICS_SMOKE_ADMIN_PASSWORD');

const isRecord = (value) => (
  value !== null && typeof value === 'object' && !Array.isArray(value)
);

const payloadMessage = (payload) => {
  if (!isRecord(payload)) return 'empty or non-object response';
  if (
    isRecord(payload.error)
    && typeof payload.error.message === 'string'
  ) {
    return payload.error.message;
  }
  if (typeof payload.message === 'string') return payload.message;
  if (typeof payload.error === 'string') return payload.error;
  return 'request failed';
};

const request = async (path, { token, ...init } = {}) => {
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (init.body !== undefined) headers.set('Content-Type', 'application/json');
  if (token) headers.set('Authorization', `Bearer ${token}`);

  const response = await fetch(`${baseURL}${path}`, { ...init, headers });
  const text = await response.text();
  let payload;
  if (text.trim() !== '') {
    try {
      payload = JSON.parse(text);
    } catch {
      throw new Error(`${init.method ?? 'GET'} ${path}: invalid JSON`);
    }
  }
  return { response, payload };
};

const expectStatus = ({ response, payload }, allowed, label) => {
  if (!allowed.includes(response.status)) {
    throw new Error(
      `${label}: expected ${allowed.join(' or ')}, got ${response.status}: ${payloadMessage(payload)}`,
    );
  }
};

const publicRead = await request('/comics?from=0&limit=1');
expectStatus(publicRead, [200], 'public catalog read');
if (!Array.isArray(publicRead.payload)) {
  throw new Error('public catalog read: expected an array');
}
console.log('PASS public catalog read');

const anonymousWrite = await request('/comics', {
  method: 'POST',
  body: JSON.stringify({ titles: [] }),
});
expectStatus(anonymousWrite, [401], 'anonymous catalog write');
console.log('PASS anonymous catalog write denied');

const login = await request('/login', {
  method: 'POST',
  body: JSON.stringify({ email, password }),
});
expectStatus(login, [200], 'admin login');
const token = (
  isRecord(login.payload)
  && isRecord(login.payload.data)
  && typeof login.payload.data.access_token === 'string'
)
  ? login.payload.data.access_token
  : undefined;
if (!token) throw new Error('admin login: access token missing');
console.log('PASS admin login');

const title = `Frontend smoke ${Date.now()}`;
const createBody = {
  titles: [title],
  current_chap: 1,
  viewed_chap: 0,
  cover: 'https://example.invalid/frontend-smoke.jpg',
  cover_visible: true,
  com_type: 3,
  status: 2,
  published_in: [0],
  genres: [0],
  description: 'Frontend API consistency smoke record',
  author: 'Smoke test',
  track: false,
  rating: 0,
};

let comicID;
try {
  const created = await request('/comics', {
    method: 'POST',
    token,
    body: JSON.stringify(createBody),
  });
  expectStatus(created, [201], 'admin comic create');
  comicID = isRecord(created.payload) && typeof created.payload.id === 'number'
    ? created.payload.id
    : undefined;
  if (!comicID) throw new Error('admin comic create: numeric id missing');
  console.log('PASS admin comic create');

  const updated = await request(`/comics/${comicID}`, {
    method: 'PUT',
    token,
    body: JSON.stringify({ track: true, rating: 5 }),
  });
  expectStatus(updated, [200], 'admin comic update');
  if (
    !isRecord(updated.payload)
    || updated.payload.track !== true
    || updated.payload.rating !== 5
  ) {
    throw new Error('admin comic update: validated fields do not match');
  }
  console.log('PASS admin comic update');
} finally {
  if (comicID !== undefined) {
    const deleted = await request(`/comics/${comicID}`, {
      method: 'DELETE',
      token,
    });
    expectStatus(deleted, [202, 204], 'admin comic cleanup');
    console.log('PASS admin comic cleanup');
  }
}
```

It performs:

1. anonymous `GET /comics?from=0&limit=1` → `200`;
2. anonymous invalid `POST /comics` → `401`;
3. admin `POST /login` → access token;
4. admin create of the unique Date.now-suffixed smoke comic → `201`;
5. admin update `{ "track": true, "rating": 5 }` → `200` and validated fields;
6. admin delete → `202` or `204`;
7. `finally` cleanup delete if create succeeded but a later assertion failed.

Print exactly:

```text
PASS public catalog read
PASS anonymous catalog write denied
PASS admin login
PASS admin comic create
PASS admin comic update
PASS admin comic cleanup
```

Use `Authorization: Bearer ${token}`, `Accept: application/json`, and JSON content type. Parse every nonempty response as JSON and throw with status plus normalized `error.message`, `message`, or string `error`. Never print the token or password.

- [ ] **Step 6: Run the live smoke after dependency integration**

With a disposable admin in the target environment:

```bash
npm run smoke:frontend-api
```

Expected: all six `PASS` lines and exit `0`. Before Workstream 1 lands, the second assertion intentionally fails because anonymous writes are still public; do not weaken that assertion.

- [ ] **Step 7: Add Make and CI gates**

Add:

```make
## @frontend:front-e2e        Run Playwright frontend API consistency tests
front-e2e:
	npm run test:e2e
```

Add `front-e2e` to `.PHONY`. Keep `check-front: front-test front-build` unchanged so local release checks do not implicitly download a browser; CI invokes `front-e2e` as a separate required gate.

After `npm ci` in the frontend CI job, add:

```yaml
      - run: npx playwright install --with-deps chromium
      - run: npm test
      - run: npm run build
      - run: npm run test:e2e
```

Do not put live admin credentials in CI; mocked Playwright tests are the deterministic PR gate, while the live script is a post-deploy/manual security smoke.

- [ ] **Step 8: Run final verification**

Run:

```bash
make check-front
make front-e2e
```

Expected: Vitest, TypeScript/Vite build, and five Chromium tests all exit `0`.

Run:

```bash
rg "ServerHelpers|AuthHelpers|console\.debug" src/frontend \
  --glob '*.{ts,tsx}' --glob '!**/*.test.*'
```

Expected: exit `1` with no output.

Run:

```bash
git diff --check
git status --short
```

Expected: `git diff --check` exits `0`; status lists only files named in this plan plus pre-existing unrelated work.

- [ ] **Step 9: Commit**

```bash
git add package.json package-lock.json playwright.config.ts \
  tests/e2e/frontend-api-consistency.spec.ts \
  scripts/smoke/frontend-api-consistency.mjs \
  Makefile .github/workflows/ci.yml
git commit -m "Add frontend API consistency smoke coverage"
```

## Migration and Rollout

1. **Land client and decoders first:** Tasks 1–2 are behavior-neutral foundations. Legacy helpers remain until their consumers migrate, so either commit can be reverted without changing server behavior.
2. **Land Workstream 1 and 6 contracts:** Verify the dependency gate with Go route/auth tests before enabling the admin-only UI. Profile must return validated role data; protected routes must return JSON `401/403`.
3. **Move the provider boundary:** Task 3 preserves cookie refresh, Web Lock
   deduplication, and server logout while globally handling session expiry and
   retaining compatibility with both target and legacy error envelopes.
4. **Move public reads:** Task 4 proves anonymous list traffic has no bearer and preserves existing debounce/refresh behavior. This is the safest production canary because it does not mutate data.
5. **Move writes by workflow:** Land card actions, then create/edit/merge/scrape. Each commit has focused tests and can be reverted independently while the endpoint modules remain.
6. **Delete parallel clients only after the invariant check:** Task 7 is the cutoff. After it lands, reject new direct `fetch` calls in review and CI.
7. **Canary deployment:** deploy the frontend against the merged secured backend, sign in as an admin, run the live smoke, then manually confirm a signed-out browser can search/paginate but cannot see mutation controls.
8. **Observe:** during the canary, watch backend counts for `401`, `403`, and mutation success. A spike in public-read `401` indicates a route-policy regression because reads use no bearer.
9. **Rollback:** revert the frontend deployment/commits in reverse order. No database rollback is required. The compatibility error parser permits frontend/backend versions to overlap during rollback, but never roll back Workstream 1 solely to restore public mutations.
10. **Remove compatibility parsing later:** only after all deployed backends emit the target nested error envelope may a follow-up delete legacy `{message}`/string `{error}` parsing. That cleanup is outside this workstream.

## Acceptance Checklist

- [ ] Exactly one production `fetch` owner exists: `src/frontend/api/client.ts`.
- [ ] Success bodies, comic arrays, profile data, pagination headers, and scrape status are validated from `unknown`.
- [ ] Target and legacy error bodies normalize to `ApiError`.
- [ ] Public reads send neither bearer nor auth cookies; every catalog/scrape mutation is marked `auth: 'required'`.
- [ ] Every non-GET browser request sends `X-CSRF-Protection: 1`; auth
  requests include credentials without exposing the refresh token.
- [ ] Missing/stale access tokens perform at most one rotating cookie refresh,
  and local-first logout still calls the idempotent server endpoint.
- [ ] One authenticated `401` clears session state and emits one global session-expired toast.
- [ ] Authenticated `401` failures do not also emit a workflow-specific mutation error toast.
- [ ] `403` keeps the session and shows the server's admin-required message.
- [ ] All mutation state changes happen after `response.ok` and successful decoding.
- [ ] Every mutation success invalidates the catalog list.
- [ ] Card-local state is replaced by server payloads and then reconciled by the list.
- [ ] Superseded list/status requests receive abort signals; aborts remain silent.
- [ ] Resize-driven page-size changes reset `from` to `0`.
- [ ] No production `console.debug`, `ServerHelpers`, or `AuthHelpers` usage remains.
- [ ] Vitest, build, Playwright, and the dependency-gated live smoke pass.
- [ ] No backend schema or data migration is introduced.
