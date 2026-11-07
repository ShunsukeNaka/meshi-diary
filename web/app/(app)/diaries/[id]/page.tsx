import type { Metadata } from "next";
import { Suspense } from "react";
import DiaryDetailView from "./diary-detail-view";

// AuthGuard が確かめ終わるまで描かないので、instant navigation の検査から外す（カレンダーと同じ）
export const instant = false;

export const metadata: Metadata = {
  title: "日記 | meshi-diary",
};

// デザインは docs/v2.pen の「04 日記の詳細」「PC 04 日記の詳細」
export default function DiaryDetailPage() {
  return (
    <Suspense>
      <DiaryDetailView />
    </Suspense>
  );
}
