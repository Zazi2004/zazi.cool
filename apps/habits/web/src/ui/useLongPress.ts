import { useCallback, useRef, type PointerEvent } from "react";
import { cancelsHold, holdDelay } from "../today";

/**
 * Pointer handlers that call `onLongPress` after a ~450 ms press. Moving more than 10 px
 * cancels it so a swipe or a vertical scroll never opens the sheet. `consumeClick` tells
 * the click that follows a long press to stand down.
 */
export function useLongPress(onLongPress: () => void, enabled: boolean) {
  const timer = useRef<number | undefined>(undefined);
  const origin = useRef({ x: 0, y: 0 });
  const fired = useRef(false);

  const cancel = useCallback(() => window.clearTimeout(timer.current), []);

  return {
    consumeClick(): boolean {
      const was = fired.current;
      fired.current = false;
      return was;
    },
    handlers: {
      onPointerDown(e: PointerEvent) {
        if (!enabled || (e.pointerType === "mouse" && e.button !== 0)) return;
        fired.current = false;
        origin.current = { x: e.clientX, y: e.clientY };
        cancel();
        timer.current = window.setTimeout(() => {
          fired.current = true;
          onLongPress();
        }, holdDelay);
      },
      onPointerMove(e: PointerEvent) {
        if (cancelsHold(e.clientX - origin.current.x, e.clientY - origin.current.y)) cancel();
      },
      onPointerUp: cancel,
      onPointerCancel: cancel,
      onPointerLeave: cancel,
      // The browser's own long-press menu would fight the sheet.
      onContextMenu: (e: { preventDefault: () => void }) => e.preventDefault(),
    },
  };
}
