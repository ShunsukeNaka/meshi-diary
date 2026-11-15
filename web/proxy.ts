import { NextResponse, type NextRequest } from "next/server";
import {
  calendarHref,
  calendarPath,
  isDate,
  isMonth,
  monthOf,
  todayJST,
} from "@/lib/date";

// / を開いたら、今月のカレンダー（/YYYY/MM）に転送する。
// 前の形の URL（/?month=YYYY-MM&date=YYYY-MM-DD）は、同じ月と日の新しい URL に転送する。
// 「今月」はリクエストのたびに決める必要があり、next.config の redirects では書けない。
// ページで転送すると、cacheComponents で作られたページの作りが込み入る（サーバー側）か、
// 一度 / を描いてから移ることになる（ブラウザ側）ので、ページを描く前の proxy で転送する。
// / には page.tsx を置いていない（カレンダーは app/(app)/[year]/[month]/）
export function proxy(request: NextRequest) {
  const q = request.nextUrl.searchParams;
  const qMonth = q.get("month");
  const qDate = q.get("date");
  const month = isMonth(qMonth)
    ? qMonth
    : monthOf(isDate(qDate) ? qDate : todayJST());
  const target =
    isDate(qDate) && monthOf(qDate) === month
      ? calendarHref(qDate)
      : calendarPath(month);
  return NextResponse.redirect(new URL(target, request.url));
}

export const config = {
  matcher: "/",
};
