/**
 * Index to focus after Tab (or Shift+Tab) in a focus trap of `count` elements, wrapping
 * around at both ends. `current` is -1 when the focus is outside the trap.
 */
export function nextFocusIndex(count: number, current: number, backwards: boolean): number {
  if (current === -1) return backwards ? count - 1 : 0;
  return (current + (backwards ? -1 : 1) + count) % count;
}
