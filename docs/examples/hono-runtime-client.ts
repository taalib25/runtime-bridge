export type RuntimePlan = {
  cpuMillis: number
  memoryMi: number
}

export type RuntimeStorage = {
  sizeGi: number
  storageClassName: string
}

export type RuntimeNetwork = {
  mode: 'subdomain' | 'path'
  host: string
  path?: string
  subdomain?: string
  ingressClassName?: string
}

export type RuntimeEnvVar = {
  name: string
  value: string
}

export type RuntimeSecretReference = {
  name: string
}

export type CreateRuntimeRequest = {
  tenantId: string
  runtimeId: string
  image: string
  plan: RuntimePlan
  storage: RuntimeStorage
  network: RuntimeNetwork
  config?: Record<string, unknown>
  soul?: string
  env?: RuntimeEnvVar[]
  envFromSecrets?: RuntimeSecretReference[]
  requestId?: string
  actorId?: string
}

export type RuntimeResponse = {
  name: string
  runtimeId: string
  tenantId: string
  phase?: 'creating' | 'running' | 'failed' | 'deleting' | 'deleted'
  url?: string
  namespace?: string
  deploymentName?: string
  serviceName?: string
  ingressName?: string
  readyReplicas?: number
  desiredReplicas?: number
  message?: string
  observedGeneration?: number
  createdAt?: string
}

export type RuntimeHealthResponse = {
  runtimeId: string
  phase?: 'creating' | 'running' | 'failed' | 'deleting' | 'deleted'
  healthy: boolean
  namespace?: string
  serviceUrl?: string
  readyReplicas?: number
  desiredReplicas?: number
  appStatusCode?: number
  message?: string
}

export type RuntimeSecretsResponse = {
  runtimeId: string
  secretName: string
  exists: boolean
  keys: string[]
  envFromAttached: boolean
}

export type UpsertRuntimeSecretsRequest = {
  data: Record<string, string>
  requestId?: string
  actorId?: string
}

export class HermesOperatorClient {
  constructor(
    private readonly baseUrl: string,
    private readonly sharedSecret: string,
    private readonly fetchImpl: typeof fetch = fetch,
  ) {}

  private headers(init?: HeadersInit): HeadersInit {
    return {
      'Content-Type': 'application/json',
      'X-Controller-Secret': this.sharedSecret,
      ...init,
    }
  }

  private async handle<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const body = await response.json().catch(() => ({ error: `HTTP ${response.status}` }))
      throw new Error(body.error ?? `HTTP ${response.status}`)
    }
    return response.json() as Promise<T>
  }

  async createRuntime(payload: CreateRuntimeRequest): Promise<RuntimeResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify(payload),
    })
    return this.handle<RuntimeResponse>(response)
  }

  async updateRuntime(runtimeId: string, payload: CreateRuntimeRequest): Promise<RuntimeResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}`, {
      method: 'PUT',
      headers: this.headers(),
      body: JSON.stringify(payload),
    })
    return this.handle<RuntimeResponse>(response)
  }

  async listRuntimes(tenantId?: string): Promise<{ runtimes: RuntimeResponse[] }> {
    const url = new URL(`${this.baseUrl}/runtimes`)
    if (tenantId) {
      url.searchParams.set('tenantId', tenantId)
    }
    const response = await this.fetchImpl(url.toString(), {
      method: 'GET',
      headers: this.headers(),
    })
    return this.handle<{ runtimes: RuntimeResponse[] }>(response)
  }

  async getRuntime(runtimeId: string): Promise<RuntimeResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}`, {
      method: 'GET',
      headers: this.headers(),
    })
    return this.handle<RuntimeResponse>(response)
  }

  async getRuntimeHealth(runtimeId: string): Promise<RuntimeHealthResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}/health`, {
      method: 'GET',
      headers: this.headers(),
    })
    return this.handle<RuntimeHealthResponse>(response)
  }

  async deleteRuntime(runtimeId: string): Promise<{ runtimeId: string; phase: string; message: string }> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}`, {
      method: 'DELETE',
      headers: this.headers(),
    })
    return this.handle<{ runtimeId: string; phase: string; message: string }>(response)
  }

  async getRuntimeSecrets(runtimeId: string): Promise<RuntimeSecretsResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}/secrets`, {
      method: 'GET',
      headers: this.headers(),
    })
    return this.handle<RuntimeSecretsResponse>(response)
  }

  async upsertRuntimeSecrets(runtimeId: string, payload: UpsertRuntimeSecretsRequest): Promise<RuntimeSecretsResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}/secrets`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify(payload),
    })
    return this.handle<RuntimeSecretsResponse>(response)
  }

  async deleteRuntimeSecretKey(runtimeId: string, key: string): Promise<RuntimeSecretsResponse> {
    const response = await this.fetchImpl(`${this.baseUrl}/runtimes/${encodeURIComponent(runtimeId)}/secrets/${encodeURIComponent(key)}`, {
      method: 'DELETE',
      headers: this.headers(),
    })
    return this.handle<RuntimeSecretsResponse>(response)
  }
}
