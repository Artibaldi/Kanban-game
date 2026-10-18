// API client (pattern from Devops-pilot): lazy CSRF token, JSON errors mapped to Russian text,
// and an Idempotency-Key on every state change. A network failure is retried once with the SAME key,
// so a lost response can never apply an action twice.
const API = '/api/v1'
let csrfToken = ''
let unauthorizedHandler: (() => void) | undefined

export interface ApiError extends Error { status: number; code?: string }

export function setUnauthorizedHandler(h: () => void) { unauthorizedHandler = h }
export function setCsrf(token: string) { csrfToken = token }

const messages: Record<string, string> = {
  login_required: 'Сессия закончилась. Войдите снова.',
  csrf_required: 'Обновите страницу и повторите действие.',
  invalid_origin: 'Запрос с этого адреса не разрешён.',
  not_found: 'Запись не найдена или недоступна вашей команде.',
  forbidden: 'Недостаточно прав для этого действия.',
  idempotency_conflict: 'Этот запрос уже был отправлен с другими данными. Обновите страницу.',
  unavailable: 'Сервис временно недоступен. Попробуйте ещё раз.',
  busy: 'Сервер занят. Повторите действие.',
}

async function toError(response: Response): Promise<ApiError> {
  let payload: { code?: string; message?: string } = {}
  try { payload = await response.json() } catch { /* empty body */ }
  const text = payload.message || (payload.code && messages[payload.code]) || `Не удалось выполнить запрос (${response.status}).`
  return Object.assign(new Error(text), { status: response.status, code: payload.code })
}

async function refreshCsrf() {
  const r = await fetch(`${API}/csrf`, { credentials: 'include' })
  if (!r.ok) throw await toError(r)
  csrfToken = (await r.json()).csrf_token
}

export function newIdempotencyKey() {
  return typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export async function request<T>(method: string, path: string, body?: unknown, key?: string): Promise<T> {
  const headers = new Headers()
  if (body !== undefined) headers.set('Content-Type', 'application/json')
  const mutation = method !== 'GET'
  if (mutation) {
    if (!csrfToken) await refreshCsrf()
    headers.set('X-CSRF-Token', csrfToken)
    headers.set('Idempotency-Key', key || newIdempotencyKey())
  }
  const init: RequestInit = { method, headers, credentials: 'include', body: body === undefined ? undefined : JSON.stringify(body) }
  let response: Response
  try {
    response = await fetch(`${API}${path}`, init)
  } catch {
    if (!mutation) throw new Error('Нет связи с сервером. Проверьте соединение.')
    // One retry with the same Idempotency-Key: the server answers from its record if the first attempt landed.
    try { response = await fetch(`${API}${path}`, init) } catch { throw new Error('Нет связи с сервером. Действие не подтверждено — обновите страницу.') }
  }
  if (response.status === 401 && path !== '/me') unauthorizedHandler?.()
  if (response.status === 403) {
    const clone = response.clone()
    try {
      const p = await clone.json()
      if (p.code === 'csrf_required') { csrfToken = '' }
    } catch { /* ignore */ }
  }
  if (!response.ok) throw await toError(response)
  const data = await response.json() as T
  if (data && typeof data === 'object' && 'csrf_token' in (data as object)) csrfToken = (data as unknown as { csrf_token: string }).csrf_token
  return data
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body: unknown = {}, key?: string) => request<T>('POST', path, body, key),
  patch: <T>(path: string, body: unknown, key?: string) => request<T>('PATCH', path, body, key),
  put: <T>(path: string, body: unknown, key?: string) => request<T>('PUT', path, body, key),
  del: <T>(path: string, key?: string) => request<T>('DELETE', path, undefined, key),
}
