import type { UserAvailableChannel } from '@/api/channels'

/**
 * 把「可用渠道」按分组拆成「这把 Key 绑定该分组时能填的模型」。
 * 同一分组出现在多个渠道/平台 section 时取并集。
 */
export function modelsByGroupId(channels: UserAvailableChannel[]): Record<number, string[]> {
  const sets = new Map<number, Set<string>>()

  for (const channel of channels) {
    for (const section of channel.platforms || []) {
      const names = (section.supported_models || [])
        .map((model) => model.name?.trim())
        .filter((name): name is string => Boolean(name))
      if (names.length === 0) continue

      for (const group of section.groups || []) {
        let set = sets.get(group.id)
        if (!set) {
          set = new Set()
          sets.set(group.id, set)
        }
        for (const name of names) {
          set.add(name)
        }
      }
    }
  }

  const out: Record<number, string[]> = {}
  for (const [groupId, set] of sets) {
    out[groupId] = [...set].sort((a, b) => a.localeCompare(b))
  }
  return out
}
