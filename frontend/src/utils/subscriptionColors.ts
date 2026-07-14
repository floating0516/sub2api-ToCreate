import {
  platformAccentBarClass,
  platformBadgeClass,
  platformBadgeLightClass,
  platformBorderClass,
  platformButtonClass,
  platformDiscountClass,
  platformIconClass,
  platformTextClass,
} from '@/utils/platformColors'

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
    accentBar: 'bg-gradient-to-r from-yellow-400 to-amber-500',
    badge: 'border-yellow-500/30 bg-yellow-500/10 text-amber-700 dark:text-yellow-300',
    badgeLight: 'bg-yellow-500/10 text-amber-700 dark:bg-yellow-500/10 dark:text-yellow-300',
    border: 'border-yellow-500/30 dark:border-yellow-500/30',
    button: 'bg-yellow-400 text-gray-950 hover:bg-yellow-500 active:bg-amber-500 dark:bg-yellow-400 dark:hover:bg-yellow-300',
    discount: 'bg-yellow-100 text-amber-800 dark:bg-yellow-900/40 dark:text-yellow-300',
    icon: 'text-amber-500 dark:text-yellow-400',
    text: 'text-amber-600 dark:text-yellow-400',
  },
  standard: {
    accentBar: 'bg-gradient-to-r from-green-400 to-emerald-500',
    badge: 'border-green-500/30 bg-green-500/10 text-green-700 dark:text-green-300',
    badgeLight: 'bg-green-500/10 text-green-700 dark:bg-green-500/10 dark:text-green-300',
    border: 'border-green-500/30 dark:border-green-500/30',
    button: 'bg-green-600 text-white hover:bg-green-700 active:bg-green-800 dark:bg-green-600 dark:hover:bg-green-500',
    discount: 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
    icon: 'text-green-500 dark:text-green-400',
    text: 'text-green-600 dark:text-green-400',
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
  if (name.includes('标准') || /\bstandard\b/.test(name)) return 'standard'
  return null
}

function tierClasses(context: SubscriptionColorContext): SubscriptionTierClasses | null {
  const tier = detectSubscriptionTier(context)
  return tier ? TIER_CLASSES[tier] : null
}

function platform(context: SubscriptionColorContext): string {
  return context.platform || ''
}

export function subscriptionAccentBarClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.accentBar || platformAccentBarClass(platform(context))
}

export function subscriptionBadgeClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.badge || platformBadgeClass(platform(context))
}

export function subscriptionBadgeLightClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.badgeLight || platformBadgeLightClass(platform(context))
}

export function subscriptionBorderClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.border || platformBorderClass(platform(context))
}

export function subscriptionButtonClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.button || platformButtonClass(platform(context))
}

export function subscriptionDiscountClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.discount || platformDiscountClass(platform(context))
}

export function subscriptionIconClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.icon || platformIconClass(platform(context))
}

export function subscriptionTextClass(context: SubscriptionColorContext): string {
  return tierClasses(context)?.text || platformTextClass(platform(context))
}
