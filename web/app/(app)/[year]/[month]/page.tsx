import type { Metadata } from "next";
import { Suspense } from "react";
import CalendarView from "./calendar-view";

// (app) の画面は、AuthGuard が GET /me で確かめ終わるまで表示しない。
// そのため instant navigation の検査ではページを描けないので、検査から外す
export const instant = false;

export const metadata: Metadata = {
  title: "カレンダー | meshi-diary",
};

// デザインは docs/v2.pen の「02 カレンダー」「PC 02 カレンダー」
export default function CalendarPage() {
  return (
    <Suspense>
      <CalendarView />
    </Suspense>
  );
}
