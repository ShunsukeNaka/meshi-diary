"use client";

import Script from "next/script";
import { useEffect, useRef, useState } from "react";

const clientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;

// GIS のボタンの幅の上限（px）
const maxButtonWidth = 400;

const defaultNote = "初めての方も同じボタンから登録できます";

export default function GoogleLogin() {
  const buttonRef = useRef<HTMLDivElement>(null);
  const [gisReady, setGisReady] = useState(false);

  useEffect(() => {
    const parent = buttonRef.current;
    if (!gisReady || !clientId || !parent) {
      return;
    }

    google.accounts.id.initialize({
      client_id: clientId,
      // 受け取った ID トークンで API を呼ぶ処理は、次の PR で作る
      callback: () => {},
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
  }, [gisReady]);

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
      <p className="text-center text-xs text-text-secondary">
        {clientId
          ? defaultNote
          : "NEXT_PUBLIC_GOOGLE_CLIENT_ID が設定されていません"}
      </p>
    </div>
  );
}
