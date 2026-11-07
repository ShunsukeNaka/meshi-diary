import type { Metadata } from "next";
import { Suspense } from "react";
import NewDiary from "./new-diary";

// AuthGuard が確かめ終わるまで描かないので、instant navigation の検査から外す（カレンダーと同じ）
export const instant = false;

export const metadata: Metadata = {
  title: "日記を書く | meshi-diary",
};

export default function NewDiaryPage() {
  return (
    <Suspense>
      <NewDiary />
    </Suspense>
  );
}
