/**
 * API 错误类
 */
export class APIError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    message?: string
  ) {
    super(message || `HTTP ${status}: ${statusText}`)
    this.name = 'APIError'
  }
}

/**
 * API 请求配置选项
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
   * 通用的 fetch 请求方法，包含统一的错误处理
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

    // 检查 HTTP 状态码
    if (!res.ok) {
      let errorMessage = res.statusText
      try {
        // 尝试从响应体获取错误信息
        const errorData = await res.json()
        errorMessage = errorData.message || errorData.error || errorMessage
      } catch {
        // 如果解析失败，使用默认错误消息
      }
      throw new APIError(res.status, errorMessage)
    }

    // 对于 204 No Content，返回空对象
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
 * API客户端单例实例
 * 使用相对路径通过 Vite 开发服务器代理访问后端 API
 * 在生产环境中，需要在 nginx 或其他反向代理中配置 /api 路由
 */
export const apiClient = new ApiClient('/api');

/**
 * 更新API客户端的基础URL
 * @param baseUrl 新的基础URL
 */
export function updateApiClientUrl(baseUrl: string): void {
  apiClient.setBaseUrl(baseUrl);
}
