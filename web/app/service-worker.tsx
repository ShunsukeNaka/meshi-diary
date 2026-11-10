"use client";

import { useEffect } from "react";

// 写真の共有を受け取る Service Worker（public/sw.js）を登録する
export default function ServiceWorker() {
  useEffect(() => {
    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.register("/sw.js").catch(() => {
        // 登録できなくても、共有のほかは今までどおり使える
      });
    }
  }, []);
  return null;
}
