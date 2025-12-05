const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1'

interface ApiError {
  error: string
}

class ApiClient {
  private baseUrl: string
  private token: string | null = null

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
  }

  setToken(token: string) {
    this.token = token
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      'X-Admin-Token': 'change-me-admin',
      ...(this.token && { Authorization: `Bearer ${this.token}` }),
      ...options?.headers,
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new Error(error.error || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // Nodes
  async getNodes() {
    return this.request<any[]>('/nodes')
  }

  async createNode(data: {
    name: string
    ip: string
    ssh_port?: number
    role: string
    username: string
    password?: string
    private_key?: string
  }) {
    return this.request('/nodes', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteNode(id: string) {
    return this.request(`/nodes/${id}`, { method: 'DELETE' })
  }

  async installAgent(id: string) {
    return this.request(`/nodes/${id}/install-agent`, { method: 'POST' })
  }

  async getTaskStatus(taskId: string) {
    return this.request(`/tasks/${taskId}`)
  }

  // Tunnels
  async getTunnels() {
    return this.request<any[]>('/tunnels')
  }

  async createTunnel(data: {
    name: string
    protocol: string
    source_node_id: number
    dest_node_id: number
    source_port?: number
    dest_port?: number
  }) {
    return this.request('/tunnels', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteTunnel(id: string) {
    return this.request(`/tunnels/${id}`, { method: 'DELETE' })
  }

  // Services
  async getServices() {
    return this.request<any[]>('/services')
  }

  async createService(data: {
    name: string
    protocol: string
    node_id: number
    listen_port: number
    routing_mode: string
    config?: any
  }) {
    return this.request('/services', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteService(id: string) {
    return this.request(`/services/${id}`, { method: 'DELETE' })
  }

  // Timeline
  async getTimeline(params?: { resource_type?: string; resource_id?: string }) {
    const query = new URLSearchParams(params as any).toString()
    return this.request<any[]>(`/timeline${query ? `?${query}` : ''}`)
  }

  // Cleanup
  async cleanupNode(data: {
    node_id: number
    mode: 'soft' | 'hard'
    force?: boolean
    confirm_text?: string
  }) {
    return this.request('/cleanup', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }
}

export const api = new ApiClient(API_BASE_URL)
