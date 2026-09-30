const nf = new Intl.NumberFormat('ru-RU')
export const num = (n: number) => nf.format(n)
export const signed = (n: number) => (n > 0 ? '+' : n < 0 ? '−' : '') + nf.format(Math.abs(n))

export function plural(n: number, one: string, few: string, many: string) {
  const a = Math.abs(n) % 100, b = a % 10
  if (a > 10 && a < 20) return many
  if (b === 1) return one
  if (b >= 2 && b <= 4) return few
  return many
}

export function date(v: string | null) {
  if (!v) return '—'
  const d = new Date(v.length === 10 ? v + 'T00:00:00' : v)
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}

export function dateTime(v: string) {
  return new Date(v).toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

export function ago(v: string) {
  const s = Math.round((Date.now() - new Date(v).getTime()) / 1000)
  if (s < 60) return 'только что'
  const m = Math.round(s / 60)
  if (m < 60) return `${m} ${plural(m, 'минуту', 'минуты', 'минут')} назад`
  const h = Math.round(m / 60)
  if (h < 24) return `${h} ${plural(h, 'час', 'часа', 'часов')} назад`
  const d = Math.round(h / 24)
  return `${d} ${plural(d, 'день', 'дня', 'дней')} назад`
}

export function duration(ms: number) {
  const s = Math.max(0, Math.ceil(ms / 1000))
  if (s < 60) return `${s} с`
  const m = Math.floor(s / 60)
  return `${m} мин ${s % 60} с`
}

/** isOverdue: the due date has passed (the whole due day counts as on time). */
export function isOverdue(due: string | null, now = new Date()) {
  if (!due) return false
  return new Date(due + 'T23:59:59') < now
}
