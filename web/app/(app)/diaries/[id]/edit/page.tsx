import type { Metadata } from "next";
import { Suspense } from "react";
import EditDiary from "./edit-diary";

// AuthGuard が確かめ終わるまで描かないので、instant navigation の検査から外す（カレンダーと同じ）
export const instant = false;

export const metadata: Metadata = {
  title: "日記を編集 | meshi-diary",
};

export default function EditDiaryPage() {
  return (
    <Suspense>
      <EditDiary />
    </Suspense>
  );
}
