export interface ApiErrorBody {
  error: string;
  details?: string[];
}

export class ApiError extends Error {
  readonly status: number;
  readonly details?: string[];

  constructor(status: number, body: ApiErrorBody) {
    super(body.error);
    this.status = status;
    this.details = body.details;
  }
}

interface ApiFetchOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  body?: unknown;
  token?: string;
}

function apiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
}

/**
 * Thin fetch wrapper for the browser → API boundary: resolves the URL against
 * NEXT_PUBLIC_API_URL, attaches a bearer token when given one, and normalizes
 * both transport failures and `{error, details?}` API responses into ApiError
 * so callers can render a single error shape regardless of cause.
 */
export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }
  if (options.token) {
    headers['Authorization'] = `Bearer ${options.token}`;
  }

  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}${path}`, {
      method: options.method ?? 'GET',
      headers,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    });
  } catch {
    throw new ApiError(0, { error: 'network error' });
  }

  const text = await res.text();
  const data = text ? (JSON.parse(text) as unknown) : null;

  if (!res.ok) {
    if (data && typeof data === 'object' && 'error' in data) {
      const body = data as { error: unknown; details?: unknown };
      throw new ApiError(res.status, {
        error: String(body.error),
        details: Array.isArray(body.details) ? body.details.map(String) : undefined,
      });
    }
    throw new ApiError(res.status, { error: 'request failed' });
  }

  return data as T;
}
