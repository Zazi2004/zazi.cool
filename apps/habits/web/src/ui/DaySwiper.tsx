import { AnimatePresence, motion, useDragControls } from "motion/react";
import { useState, type PointerEvent, type ReactNode } from "react";
import { swipeTarget } from "../today";

interface Props {
  date: string;
  today: string;
  onChange: (date: string) => void;
  children: ReactNode;
}

const slide = {
  enter: (direction: number) => ({ x: direction > 0 ? "60%" : "-60%", opacity: 0 }),
  centre: { x: 0, opacity: 1 },
  exit: (direction: number) => ({ x: direction > 0 ? "-60%" : "60%", opacity: 0 }),
};

/** The day follows the finger; released far enough, it slides out for the next one. */
export function DaySwiper({ date, today, onChange, children }: Props) {
  const controls = useDragControls();
  const [seen, setSeen] = useState({ date, direction: 0 });
  if (seen.date !== date) setSeen({ date, direction: date > seen.date ? 1 : -1 });

  function start(e: PointerEvent) {
    // Dragging inside the note selects text, it must not change the day.
    if (!(e.target as HTMLElement).closest("textarea, input")) controls.start(e);
  }

  return (
    <div className="swiper">
      <AnimatePresence initial={false} custom={seen.direction}>
        <motion.div
          key={date}
          custom={seen.direction}
          variants={slide}
          initial="enter"
          animate="centre"
          exit="exit"
          transition={{ type: "spring", stiffness: 320, damping: 34 }}
          drag="x"
          dragListener={false}
          dragControls={controls}
          dragDirectionLock
          dragConstraints={{ left: 0, right: 0 }}
          dragElastic={0.3}
          style={{ touchAction: "pan-y" }}
          onPointerDown={start}
          onDragEnd={(_, info) => {
            const target = swipeTarget(date, today, info.offset.x, info.offset.y);
            if (target) onChange(target);
          }}
        >
          {children}
        </motion.div>
      </AnimatePresence>
    </div>
  );
}
