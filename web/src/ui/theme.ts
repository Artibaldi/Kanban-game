export type ThemeChoice = 'system' | 'light' | 'dark'
const KEY = 'kg-theme'

export function loadTheme(): ThemeChoice {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'light' || v === 'dark') return v
  } catch { /* storage may be blocked */ }
  return 'system'
}

export function applyTheme(choice: ThemeChoice) {
  const root = document.documentElement
  if (choice === 'system') delete root.dataset.theme
  else root.dataset.theme = choice
  try { localStorage.setItem(KEY, choice) } catch { /* ignore */ }
}
