"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { logout } from "@/lib/api";
import { LogOutIcon } from "@/components/icons";

// POST /logout を呼んでから、ログイン画面に移動する。
// ログアウトしたあとに戻るでカレンダーを開いても、AuthGuard が GET /me で確かめてログイン画面に移動する。
export default function LogoutButton({
  withLabel = false,
}: {
  withLabel?: boolean;
}) {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  async function handleClick() {
    setPending(true);
    try {
      await logout();
    } catch {
      // Cookie を消せなくても、ログイン画面には移動する（API は Cookie がなくても 204 を返す）
    }
    router.replace("/login");
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={pending}
      aria-label="ログアウト"
      className={
        withLabel
          ? "flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-text-secondary hover:bg-surface disabled:opacity-50"
          : "flex size-10 items-center justify-center rounded-full text-text-secondary hover:bg-surface disabled:opacity-50"
      }
    >
      <LogOutIcon size={withLabel ? 18 : 20} />
      {withLabel ? <span>ログアウト</span> : null}
    </button>
  );
}
