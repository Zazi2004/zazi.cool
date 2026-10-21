import { LogOut } from "lucide-react";
import type { ReactNode } from "react";
import { useLocation } from "wouter";
import { api } from "../api";
import { Button } from "./Button";

/** The bar at the top of the admin screens. */
export function TopBar({ children }: { children?: ReactNode }) {
  const [, navigate] = useLocation();
  async function logout() {
    try {
      await api.logout();
    } finally {
      navigate("/login", { replace: true });
    }
  }
  return (
    <header className="sticky top-0 z-30 border-b border-zinc-200 bg-white/80 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/80">
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-3 px-4">
        {children}
        <Button
          variant="ghost"
          size="icon"
          onClick={logout}
          aria-label="Se déconnecter"
          title="Se déconnecter"
        >
          <LogOut />
        </Button>
      </div>
    </header>
  );
}
