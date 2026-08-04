/**
 * A small deterministic PRNG (mulberry32) plus the pick/weighted-pick/
 * range helpers every mock dataset generator needs. Extracted once a
 * second dataset (services/devices/deviceDataset.ts) needed the exact
 * same helpers services/customers/customerDataset.ts already had
 * (docs/11-COMPONENT-ARCHITECTURE.md, "Future Evolution": promote a
 * pattern once a second place actually needs it).
 *
 * Each caller creates its own instance with its own seed, so two
 * datasets generated from independent seeds never influence each
 * other's sequence. Only used to keep fixture data stable across
 * reloads -- never security-sensitive.
 */
export interface Prng {
  random: () => number
  pick: <T>(items: readonly T[]) => T
  pickWeighted: <T>(items: readonly (readonly [T, number])[]) => T
  randomInt: (min: number, max: number) => number
  chance: (probability: number) => boolean
}

export function createPrng(seed: number): Prng {
  let state = seed

  function random(): number {
    state |= 0
    state = (state + 0x6d2b79f5) | 0
    let t = Math.imul(state ^ (state >>> 15), 1 | state)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }

  function pick<T>(items: readonly T[]): T {
    return items[Math.floor(random() * items.length)]
  }

  function pickWeighted<T>(items: readonly (readonly [T, number])[]): T {
    const total = items.reduce((sum, [, weight]) => sum + weight, 0)
    let roll = random() * total
    for (const [item, weight] of items) {
      roll -= weight
      if (roll <= 0) return item
    }
    return items[items.length - 1][0]
  }

  function randomInt(min: number, max: number): number {
    return Math.floor(random() * (max - min + 1)) + min
  }

  function chance(probability: number): boolean {
    return random() < probability
  }

  return { random, pick, pickWeighted, randomInt, chance }
}
