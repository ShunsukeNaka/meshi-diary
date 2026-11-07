"use client";

import { useRouter } from "next/navigation";
import Script from "next/script";
import { useEffect, useRef, useState } from "react";
import { ApiError, login, me, signup } from "@/lib/api";

const clientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;

// GIS のボタンの幅の上限（px）
const maxButtonWidth = 400;

type Status =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "error"; message: string };

const defaultNote = "初めての方も同じボタンから登録できます";

export default function GoogleLogin() {
  const router = useRouter();
  const buttonRef = useRef<HTMLDivElement>(null);
  const [gisReady, setGisReady] = useState(false);
  const [status, setStatus] = useState<Status>(
    clientId
      ? { kind: "idle" }
      : {
          kind: "error",
          message: "NEXT_PUBLIC_GOOGLE_CLIENT_ID が設定されていません",
        },
  );

  // ログイン中なら、ログイン画面を開いてもカレンダーに移動する
  useEffect(() => {
    let ignore = false;
    me()
      .then(() => {
        if (!ignore) {
          router.replace("/");
        }
      })
      // 未ログイン（401）やエラーのときは、そのままログイン画面を出す
      .catch(() => {});
    return () => {
      ignore = true;
    };
  }, [router]);

  useEffect(() => {
    const parent = buttonRef.current;
    if (!gisReady || !clientId || !parent) {
      return;
    }

    // まず /login を呼び、未登録（404）なら /signup で登録する
    async function handleCredential({
      credential,
    }: google.accounts.id.CredentialResponse) {
      setStatus({ kind: "loading" });
      try {
        try {
          await login(credential);
        } catch (e) {
          if (e instanceof ApiError && e.status === 404) {
            await signup(credential);
          } else {
            throw e;
          }
        }
        router.replace("/");
      } catch (e) {
        setStatus({ kind: "error", message: errorMessage(e) });
      }
    }

    google.accounts.id.initialize({
      client_id: clientId,
      callback: handleCredential,
    });
    google.accounts.id.renderButton(parent, {
      type: "standard",
      theme: "outline",
      size: "large",
      text: "signin_with",
      shape: "rectangular",
      logo_alignment: "center",
      width: Math.min(parent.clientWidth, maxButtonWidth),
      locale: "ja",
    });
  }, [gisReady, router]);

  return (
    <div className="flex flex-col items-center gap-3">
      <Script
        src="https://accounts.google.com/gsi/client"
        strategy="afterInteractive"
        onReady={() => setGisReady(true)}
      />
      <div
        ref={buttonRef}
        className="flex h-[52px] w-full items-center justify-center"
      />
      <p
        role={status.kind === "error" ? "alert" : undefined}
        aria-live="polite"
        className={`text-center text-xs ${
          status.kind === "error" ? "text-text-primary" : "text-text-secondary"
        }`}
      >
        {status.kind === "idle" && defaultNote}
        {status.kind === "loading" && "ログインしています…"}
        {status.kind === "error" && status.message}
      </p>
    </div>
  );
}

function errorMessage(e: unknown): string {
  if (e instanceof ApiError && e.status === 403) {
    return "このアカウントは使えません";
  }
  return "ログインできませんでした。時間をおいてもう一度お試しください";
}
