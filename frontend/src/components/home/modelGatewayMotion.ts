/** Closed-form spring sampled into WAAPI keyframes. No idle animation loop.
 * Based on the one-shape reference's step response (omega 24, damping .88).
 * The settling window makes the last frame exact without a visible snap.
 */
export function springAt(progress: number, omega = 24, damping = 0.88): number {
  if (progress <= 0) return 0
  if (progress >= 1) return 1
  const t = progress * 0.4
  const d = Math.sqrt(1 - damping * damping)
  const raw = 1 - Math.exp(-damping * omega * t) *
    (Math.cos(omega * d * t) + damping / d * Math.sin(omega * d * t))
  const x = Math.max(0, Math.min(1, (progress - 0.7) / 0.3))
  const settle = x * x * x * (x * (x * 6 - 15) + 10)
  return raw + (1 - raw) * settle
}

export function springFrames(from: Record<string, number>, to: Record<string, number>, units: Record<string, string> = {}) {
  return Array.from({ length: 41 }, (_, i) => {
    const p = springAt(i / 40)
    const frame: Keyframe = { offset: i / 40 }
    for (const key of Object.keys(to)) frame[key] = `${from[key] + (to[key] - from[key]) * p}${units[key] ?? 'px'}`
    return frame
  })
}

/** Two springs: whichever edge leads moves faster; the trailing edge follows. */
export function indicatorFrames(left: number, right: number, nextLeft: number, nextRight: number) {
  const movingRight = nextLeft > left
  return Array.from({ length: 41 }, (_, i) => {
    const l = left + (nextLeft - left) * springAt(i / 40, movingRight ? 21 : 34)
    const r = right + (nextRight - right) * springAt(i / 40, movingRight ? 34 : 21)
    return { offset: i / 40, left: `${l}px`, width: `${Math.max(0, r - l)}px` }
  })
}
