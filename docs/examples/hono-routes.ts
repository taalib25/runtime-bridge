import { Hono } from 'hono'
import { HermesOperatorClient, type CreateRuntimeRequest } from './hono-runtime-client'

const app = new Hono()

const operator = new HermesOperatorClient(
  process.env.HERMES_OPERATOR_URL ?? 'http://127.0.0.1:18080',
  process.env.HERMES_OPERATOR_SHARED_SECRET ?? '',
)

app.post('/tenants/:tenantId/runtime', async c => {
  const tenantId = c.req.param('tenantId')
  const body = await c.req.json<{
    providerModel: string
    providerBaseUrl: string
    apiKey: string
    runtimeId?: string
  }>()

  const runtimeId = body.runtimeId ?? `rt-${tenantId}`

  const payload: CreateRuntimeRequest = {
    tenantId,
    runtimeId,
    image: 'nousresearch/hermes-agent:latest',
    plan: { cpuMillis: 1000, memoryMi: 2048 },
    storage: { sizeGi: 10, storageClassName: 'local-path' },
    network: {
      mode: 'subdomain',
      host: 'hermeshq.net',
      subdomain: runtimeId,
      ingressClassName: 'traefik',
    },
    config: {
      model: {
        provider: 'auto',
        default: body.providerModel,
        base_url: body.providerBaseUrl,
      },
      agent: {
        max_turns: 90,
        gateway_timeout: 1800,
      },
      terminal: {
        backend: 'docker',
        cwd: '.',
        timeout: 180,
      },
    },
    soul: '# Hermes Runtime\n\nYou are a helpful assistant.',
    env: [{ name: 'GATEWAY_ALLOW_ALL_USERS', value: 'false' }],
  }

  const runtime = await operator.createRuntime(payload)
  await operator.upsertRuntimeSecrets(runtimeId, {
    data: {
      OPENROUTER_API_KEY: body.apiKey,
      API_SERVER_KEY: crypto.randomUUID(),
    },
    actorId: tenantId,
  })

  return c.json(runtime, 201)
})

app.put('/tenants/:tenantId/runtime/provider', async c => {
  const tenantId = c.req.param('tenantId')
  const runtimeId = `rt-${tenantId}`
  const body = await c.req.json<{
    providerModel: string
    providerBaseUrl: string
    apiKey?: string
  }>()

  const current = await operator.getRuntime(runtimeId)

  const updated = await operator.updateRuntime(runtimeId, {
    tenantId,
    runtimeId,
    image: 'nousresearch/hermes-agent:latest',
    plan: { cpuMillis: 1000, memoryMi: 2048 },
    storage: { sizeGi: 10, storageClassName: 'local-path' },
    network: {
      mode: 'subdomain',
      host: 'hermeshq.net',
      subdomain: runtimeId,
      ingressClassName: 'traefik',
    },
    config: {
      model: {
        provider: 'auto',
        default: body.providerModel,
        base_url: body.providerBaseUrl,
      },
    },
    env: [{ name: 'GATEWAY_ALLOW_ALL_USERS', value: 'false' }],
  })

  if (body.apiKey) {
    await operator.upsertRuntimeSecrets(runtimeId, {
      data: { OPENROUTER_API_KEY: body.apiKey },
      actorId: tenantId,
    })
  }

  return c.json({ before: current, after: updated })
})

app.get('/tenants/:tenantId/runtime', async c => {
  const tenantId = c.req.param('tenantId')
  const runtimeId = `rt-${tenantId}`
  const [runtime, health, secrets] = await Promise.all([
    operator.getRuntime(runtimeId),
    operator.getRuntimeHealth(runtimeId),
    operator.getRuntimeSecrets(runtimeId),
  ])
  return c.json({ runtime, health, secrets })
})

app.delete('/tenants/:tenantId/runtime', async c => {
  const tenantId = c.req.param('tenantId')
  const runtimeId = `rt-${tenantId}`
  const result = await operator.deleteRuntime(runtimeId)
  return c.json(result)
})

export default app
