// A dropdown menu in the style of shadcn/ui (MIT), on Radix: keyboard accessible.
import { DropdownMenu } from "radix-ui";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export interface MenuItem {
  label: string;
  icon?: ReactNode;
  onSelect: () => void;
  danger?: boolean;
  disabled?: boolean;
}

interface MenuProps {
  trigger: ReactNode;
  items: (MenuItem | "separator" | { label: string; heading: true })[];
}

export function Menu({ trigger, items }: MenuProps) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>{trigger}</DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="end"
          sideOffset={4}
          className="z-50 min-w-48 rounded-xl border border-zinc-200 bg-white p-1 shadow-xl dark:border-zinc-800 dark:bg-zinc-900"
        >
          {items.map((item, i) =>
            item === "separator" ? (
              <DropdownMenu.Separator key={i} className="my-1 h-px bg-zinc-200 dark:bg-zinc-800" />
            ) : "heading" in item ? (
              <DropdownMenu.Label key={i} className="px-2 py-1 text-xs text-zinc-500">
                {item.label}
              </DropdownMenu.Label>
            ) : (
              <DropdownMenu.Item
                key={i}
                disabled={item.disabled}
                onSelect={item.onSelect}
                className={cn(
                  "flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm outline-none select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40 data-[highlighted]:bg-zinc-100 dark:data-[highlighted]:bg-zinc-800 [&_svg]:size-4",
                  item.danger && "text-red-600 dark:text-red-400",
                )}
              >
                {item.icon}
                {item.label}
              </DropdownMenu.Item>
            ),
          )}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
