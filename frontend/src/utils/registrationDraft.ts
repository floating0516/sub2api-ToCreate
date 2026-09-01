const FULL_REGISTRATION_DRAFT_KEY = 'sub2api_full_registration_draft'

export interface FullRegistrationDraft {
  email: string
  password: string
}

export function storeFullRegistrationDraft(draft: FullRegistrationDraft): void {
  sessionStorage.setItem(FULL_REGISTRATION_DRAFT_KEY, JSON.stringify(draft))
}

export function consumeFullRegistrationDraft(): FullRegistrationDraft | null {
  const raw = sessionStorage.getItem(FULL_REGISTRATION_DRAFT_KEY)
  sessionStorage.removeItem(FULL_REGISTRATION_DRAFT_KEY)
  if (!raw) return null

  try {
    const draft = JSON.parse(raw) as Partial<FullRegistrationDraft>
    if (typeof draft.email !== 'string' || typeof draft.password !== 'string') return null
    return { email: draft.email, password: draft.password }
  } catch {
    return null
  }
}
