// 日付は API と同じ YYYY-MM-DD（月は YYYY-MM）の文字列で扱う。
// 「今日」は日本時間で決める（docs/v2.pen とドメインモデルの「日付: JST」）。

import type { MealType } from "./api";

const weekdays = ["日", "月", "火", "水", "木", "金", "土"];

export const mealLabels: Record<MealType, string> = {
  BREAKFAST: "朝食",
  LUNCH: "昼食",
  DINNER: "夕食",
  OTHER: "その他",
};

export const mealTypes: MealType[] = ["BREAKFAST", "LUNCH", "DINNER", "OTHER"];

// 今日の日付（日本時間）
export function todayJST(): string {
  // en-CA は YYYY-MM-DD の形で書き出す
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Tokyo" }).format(
    new Date(),
  );
}

export function isDate(s: string | null): s is string {
  if (!s || !/^\d{4}-\d{2}-\d{2}$/.test(s)) {
    return false;
  }
  const [y, m, d] = s.split("-").map(Number);
  const t = new Date(Date.UTC(y, m - 1, d));
  return (
    t.getUTCFullYear() === y &&
    t.getUTCMonth() === m - 1 &&
    t.getUTCDate() === d
  );
}

export function isMonth(s: string | null): s is string {
  return !!s && /^\d{4}-(0[1-9]|1[0-2])$/.test(s);
}

function parts(date: string) {
  const [y, m, d] = date.split("-").map(Number);
  return { y, m, d, w: new Date(Date.UTC(y, m - 1, d)).getUTCDay() };
}

// 2026-11-06 → 11月6日（金）
export function formatMonthDay(date: string): string {
  const { m, d, w } = parts(date);
  return `${m}月${d}日（${weekdays[w]}）`;
}

// 2026-11-06 → 2026年11月6日（金）
export function formatFullDate(date: string): string {
  const { y } = parts(date);
  return `${y}年${formatMonthDay(date)}`;
}

// 2026-11 → 2026年11月
export function formatMonth(month: string): string {
  const [y, m] = month.split("-").map(Number);
  return `${y}年${m}月`;
}

export function monthOf(date: string): string {
  return date.slice(0, 7);
}

export function addMonths(month: string, n: number): string {
  const [y, m] = month.split("-").map(Number);
  const t = new Date(Date.UTC(y, m - 1 + n, 1));
  return `${t.getUTCFullYear()}-${String(t.getUTCMonth() + 1).padStart(2, "0")}`;
}

export { weekdays };

// 月のカレンダーを、日曜はじまりの週ごとに返す。月の外の日は null
export function calendarWeeks(month: string): (string | null)[][] {
  const [y, m] = month.split("-").map(Number);
  const first = new Date(Date.UTC(y, m - 1, 1)).getUTCDay();
  const days = new Date(Date.UTC(y, m, 0)).getUTCDate();
  const cells: (string | null)[] = Array(first).fill(null);
  for (let d = 1; d <= days; d++) {
    cells.push(`${month}-${String(d).padStart(2, "0")}`);
  }
  while (cells.length % 7 !== 0) {
    cells.push(null);
  }
  const weeks: (string | null)[][] = [];
  for (let i = 0; i < cells.length; i += 7) {
    weeks.push(cells.slice(i, i + 7));
  }
  return weeks;
}
