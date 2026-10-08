import { describe, expect, it } from 'vitest'
import { indicatorFrames, springAt } from '../modelGatewayMotion'

describe('model gateway spring', () => {
  it('settles exactly and permits only a small overshoot', () => {
    const values = Array.from({ length: 101 }, (_, i) => springAt(i / 100))
    expect(values[0]).toBe(0)
    expect(values.at(-1)).toBe(1)
    expect(Math.max(...values)).toBeLessThan(1.01)
    expect(Math.min(...values)).toBeGreaterThanOrEqual(0)
  })
  it('stretches in both directions without a negative width and finishes on target', () => {
    for (const [a,b,c,d] of [[4,100,204,300],[204,300,4,100]]) {
      const frames = indicatorFrames(a,b,c,d)
      expect(frames.some(frame => parseFloat(frame.width) > b-a+20)).toBe(true)
      expect(frames.every(frame => parseFloat(frame.width) > 0)).toBe(true)
      expect(frames.at(-1)?.left).toBe(`${c}px`)
      expect(frames.at(-1)?.width).toBe(`${d-c}px`)
    }
  })
})
