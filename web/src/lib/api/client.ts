export class APIError extends Error {
  public status: number;
  public code?: string;
  public details?: string;
  public handled?: boolean;

  constructor(
    status: number,
    message: string,
    code?: string,
    details?: string
  ) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

/**
 * API request options
 */
interface RequestInit extends globalThis.RequestInit {
  signal?: AbortSignal
}

export class ApiClient {
  private baseUrl: string
  constructor(baseUrl: string) {
    this.baseUrl = baseUrl.replace(/\/+$/, '')
  }

  getBaseUrl(): string {
    return this.baseUrl
  }

  setBaseUrl(baseUrl: string): void {
    this.baseUrl = baseUrl.replace(/\/+$/, '')
  }

  /**
   * Generic fetch with unified error handling
   */
  private async fetchRequest<T>(
    url: string,
    init: RequestInit = {}
  ): Promise<T> {
    const res = await fetch(url, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        ...init.headers,
      },
    })

    if (!res.ok) {
      let errorMessage = res.statusText
      let errorCode: string | undefined
      let errorDetails: string | undefined
      try {
        const errorData = await res.json()
        if (errorData.error && typeof errorData.error === 'object') {
          errorMessage = errorData.error.message || errorMessage
          errorCode = errorData.error.code
          errorDetails = errorData.error.details
        } else {
          errorMessage = errorData.message || errorData.error || errorMessage
        }
      } catch {
        // JSON parse failed, use default
      }
      throw new APIError(res.status, errorMessage, errorCode, errorDetails)
    }

    // Return empty object for 204 No Content
    if (res.status === 204) {
      return {} as T
    }

    return res.json() as Promise<T>
  }

  async get<T>(path: string, params?: Record<string, unknown>, signal?: AbortSignal): Promise<T> {
    const url = new URL(`${this.baseUrl}${path}`, window.location.origin)
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value))
        }
      })
    }
    return this.fetchRequest<T>(url.toString(), {
      method: 'GET',
      signal,
    })
  }

  async post<T>(path: string, body?: unknown): Promise<T> {
    return this.fetchRequest<T>(`${this.baseUrl}${path}`, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  async put<T>(path: string, body: unknown): Promise<T> {
    return this.fetchRequest<T>(`${this.baseUrl}${path}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  }

  async delete<T>(path: string): Promise<T> {
    return this.fetchRequest<T>(`${this.baseUrl}${path}`, {
      method: 'DELETE',
    })
  }
}

/**
 * API client singleton.
 * Uses relative path proxied through Vite dev server.
 * In production, configure /api route in nginx or other reverse proxy.
 */
export const apiClient = new ApiClient('/api');

/**
 * Update the API client base URL.
 * @param baseUrl New base URL
 */
export function updateApiClientUrl(baseUrl: string): void {
  apiClient.setBaseUrl(baseUrl);
}
