"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiError } from "./api";

type Result<T> = { key: string; data?: T; error?: unknown };

// key が変わるたびに load(key) を呼び、結果を返す。key が null のあいだは何もしない。
// load はコンポーネントの外で定義した関数を渡す（毎回作り直すと、そのたびに読み込み直すため）。
// ログインが切れていた（401）ときは、ログイン画面に移動する。
export function useFetch<T>(
  key: string | null,
  load: (key: string) => Promise<T>,
) {
  const router = useRouter();
  const [result, setResult] = useState<Result<T> | null>(null);

  useEffect(() => {
    if (key === null) {
      return;
    }
    let ignore = false;
    load(key).then(
      (data) => {
        if (!ignore) {
          setResult({ key, data });
        }
      },
      (error) => {
        if (ignore) {
          return;
        }
        if (error instanceof ApiError && error.status === 401) {
          router.replace("/login");
          return;
        }
        setResult({ key, error });
      },
    );
    return () => {
      ignore = true;
    };
  }, [key, load, router]);

  const current = result?.key === key ? result : null;
  return {
    data: current?.data,
    error: current?.error,
    loading: key !== null && current === null,
  };
}
