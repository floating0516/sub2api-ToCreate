export type SubscriptionTier = 'light' | 'standard'

export interface SubscriptionColorContext {
  planName?: string | null
  groupName?: string | null
  platform?: string | null
}

interface SubscriptionTierClasses {
  accentBar: string
  badge: string
  badgeLight: string
  border: string
  button: string
  discount: string
  icon: string
  text: string
}

const TIER_CLASSES: Record<SubscriptionTier, SubscriptionTierClasses> = {
  light: {
    accentBar: 'bg-gradient-to-r from-primary-200 to-primary-400',
    badge: 'border-primary-300/50 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300',
    badgeLight: 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300',
    border: 'border-primary-200 dark:border-primary-800',
    button: 'bg-primary-400 text-white hover:bg-primary-500 active:bg-primary-600 dark:bg-primary-400 dark:hover:bg-primary-300',
    discount: 'bg-primary-100 text-primary-800 dark:bg-primary-900/40 dark:text-primary-300',
    icon: 'text-primary-500 dark:text-primary-400',
    text: 'text-primary-600 dark:text-primary-400',
  },
  standard: {
    accentBar: 'bg-gradient-to-r from-primary-600 to-primary-800',
    badge: 'border-primary-500/40 bg-primary-100 text-primary-800 dark:bg-primary-900/40 dark:text-primary-200',
    badgeLight: 'bg-primary-200/70 text-primary-800 dark:bg-primary-900/40 dark:text-primary-200',
    border: 'border-primary-400/50 dark:border-primary-700',
    button: 'bg-primary-700 text-white hover:bg-primary-800 active:bg-primary-900 dark:bg-primary-600 dark:hover:bg-primary-500',
    discount: 'bg-primary-200 text-primary-900 dark:bg-primary-900/50 dark:text-primary-200',
    icon: 'text-primary-600 dark:text-primary-300',
    text: 'text-primary-700 dark:text-primary-300',
  },
}

export function detectSubscriptionTier(context: SubscriptionColorContext): SubscriptionTier | null {
  const name = [context.groupName, context.planName]
    .filter((value): value is string => typeof value === 'string')
    .join(' ')
    .trim()
    .toLowerCase()
    .replace(/[_-]+/g, ' ')

  if (name.includes('轻量') || /\blight\b/.test(name)) return 'light'
  if (name.includes('标准') || name.includes('高额度') || /\bstandard\b/.test(name) || /\bhigh\b/.test(name)) return 'standard'
  return null
}

function tierClasses(context: SubscriptionColorContext): SubscriptionTierClasses {
  const tier = detectSubscriptionTier(context)
  return TIER_CLASSES[tier ?? 'light']
}

export function subscriptionAccentBarClass(context: SubscriptionColorContext): string {
  return tierClasses(context).accentBar
}

export function subscriptionBadgeClass(context: SubscriptionColorContext): string {
  return tierClasses(context).badge
}

export function subscriptionBadgeLightClass(context: SubscriptionColorContext): string {
  return tierClasses(context).badgeLight
}

export function subscriptionBorderClass(context: SubscriptionColorContext): string {
  return tierClasses(context).border
}

export function subscriptionButtonClass(context: SubscriptionColorContext): string {
  return tierClasses(context).button
}

export function subscriptionDiscountClass(context: SubscriptionColorContext): string {
  return tierClasses(context).discount
}

export function subscriptionIconClass(context: SubscriptionColorContext): string {
  return tierClasses(context).icon
}

export function subscriptionTextClass(context: SubscriptionColorContext): string {
  return tierClasses(context).text
}
