"use client";

import { useRouter } from "next/navigation";
import { createContext, useContext, useEffect, useState } from "react";
import { ApiError, me, type User } from "@/lib/api";

const UserContext = createContext<User | null>(null);

// ログイン中のユーザー。AuthGuard の中でだけ使える
export function useUser(): User {
  const user = useContext(UserContext);
  if (!user) {
    throw new Error("useUser は AuthGuard の中で使う");
  }
  return user;
}

type State =
  | { kind: "checking" }
  | { kind: "ok"; user: User }
  | { kind: "error" };

// 画面を開くたびに GET /me でログイン状態を確かめ、未ログインならログイン画面に移動する。
// セッション Cookie の Path は /meshi-diary-api/ なので、ページのリクエストには付かない。
// そのためサーバー側では確かめられず、ブラウザで /me を呼んで確かめる。
export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [state, setState] = useState<State>({ kind: "checking" });

  // cacheComponents が有効なので、戻るボタンで戻ったときも Activity でこの effect が作り直され、もう一度確かめる
  useEffect(() => {
    let ignore = false;
    me()
      .then(({ user }) => {
        if (!ignore) {
          setState({ kind: "ok", user });
        }
      })
      .catch((e) => {
        if (ignore) {
          return;
        }
        if (e instanceof ApiError && e.status === 401) {
          router.replace("/login");
        } else {
          setState({ kind: "error" });
        }
      });
    return () => {
      ignore = true;
    };
  }, [router]);

  if (state.kind === "checking") {
    return null;
  }
  if (state.kind === "error") {
    return (
      <p role="alert" className="m-auto px-6 text-center text-sm text-text-secondary">
        読み込めませんでした。時間をおいてもう一度お試しください
      </p>
    );
  }
  return (
    <UserContext.Provider value={state.user}>{children}</UserContext.Provider>
  );
}
