import type { Metadata } from "next";
import { Suspense } from "react";
import ShareView from "./share-view";

// AuthGuard が確かめ終わるまで描かないので、instant navigation の検査から外す（カレンダーと同じ）
export const instant = false;

export const metadata: Metadata = {
  title: "写真を追加 | meshi-diary",
};

// 写真アプリの「共有」から開く画面（public/sw.js が写真を受け取ってから、ここに移す）。
// デザインは docs/v2.pen の「05 共有（日記を選ぶ）」
export default function SharePage() {
  return (
    <Suspense>
      <ShareView />
    </Suspense>
  );
}
