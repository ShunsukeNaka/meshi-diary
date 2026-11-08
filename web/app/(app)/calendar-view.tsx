"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { getCalendar, type CalendarDay, type CalendarMonth } from "@/lib/api";
import {
  addMonths,
  calendarWeeks,
  formatMonth,
  formatMonthDay,
  isDate,
  isMonth,
  mealLabels,
  monthOf,
  todayJST,
  weekdays,
} from "@/lib/date";
import { useFetch } from "@/lib/use-fetch";
import { ChevronLeftIcon, ChevronRightIcon } from "@/components/icons";
import RemoteImage from "@/components/remote-image";
import LogoutButton from "./logout-button";

// 表示している月と選んだ日は URL（?month=YYYY-MM&date=YYYY-MM-DD）に持つ。
// 詳細画面から戻ったときに、同じ月と日に戻れるようにするため。
export default function CalendarView() {
  const params = useSearchParams();
  const date = isDate(params.get("date")) ? params.get("date")! : todayJST();
  const month = isMonth(params.get("month"))
    ? params.get("month")!
    : monthOf(date);

  // ページを読み込み直さずに URL だけ変える（useSearchParams にも反映される）
  function setView(next: { month?: string; date?: string }) {
    const q = new URLSearchParams({
      month: next.month ?? month,
      date: next.date ?? date,
    });
    window.history.replaceState(null, "", `?${q}`);
  }

  // 日付を選んだときの一覧も、GET /calendar の diaries から出す（日付ごとには問い合わせない）。
  // 月を切り替えても選んだ日は変わらないので、選んだ日が別の月のときは、その月の分も取る。
  const calendar = useFetch(month, getCalendar);
  const dateMonth = monthOf(date);
  const otherMonth = useFetch(
    dateMonth === month ? null : dateMonth,
    getCalendar,
  );
  const dateCalendar = dateMonth === month ? calendar : otherMonth;

  return (
    <>
      {/* PC だけのトップバー。スマホでは月の切り替えの右にログアウトを置く */}
      <header className="hidden w-full items-center justify-between px-8 py-5 lg:flex">
        <span className="font-inter text-lg font-bold text-text-primary">
          meshi-diary
        </span>
        <LogoutButton withLabel />
      </header>
      <main className="mx-auto flex w-full max-w-[480px] flex-1 flex-col gap-5 px-4 pt-2 pb-6 lg:max-w-[1280px] lg:flex-row lg:gap-12 lg:px-8 lg:pt-2">
        <section className="flex flex-col gap-5 lg:flex-1">
          <MonthHeader
            month={month}
            onPrev={() => setView({ month: addMonths(month, -1) })}
            onNext={() => setView({ month: addMonths(month, 1) })}
          />
          <MonthGrid
            month={month}
            calendar={calendar}
            selected={date}
            onSelect={(d) => setView({ date: d })}
          />
        </section>
        <section className="flex flex-col gap-5 lg:w-[380px] lg:pt-[60px]">
          <DayEntries date={date} calendar={dateCalendar} />
          <Link
            href={`/diaries/new?date=${date}`}
            className="flex h-[52px] items-center justify-center rounded-app bg-text-primary px-5 text-base font-bold text-on-primary"
          >
            日記を書く
          </Link>
        </section>
      </main>
    </>
  );
}

function MonthHeader({
  month,
  onPrev,
  onNext,
}: {
  month: string;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex items-center justify-between px-1">
      <h1 className="text-[22px] font-bold text-text-primary">
        {formatMonth(month)}
      </h1>
      <div className="flex items-center gap-1">
        <IconButton label="前の月" onClick={onPrev}>
          <ChevronLeftIcon />
        </IconButton>
        <IconButton label="次の月" onClick={onNext}>
          <ChevronRightIcon />
        </IconButton>
        <span className="mx-1 h-5 w-px bg-border lg:hidden" />
        <span className="lg:hidden">
          <LogoutButton />
        </span>
      </div>
    </div>
  );
}

function IconButton({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-surface"
    >
      {children}
    </button>
  );
}

type CalendarResult = {
  data?: CalendarMonth;
  error?: unknown;
  loading: boolean;
};

function MonthGrid({
  month,
  calendar,
  selected,
  onSelect,
}: {
  month: string;
  calendar: CalendarResult;
  selected: string;
  onSelect: (date: string) => void;
}) {
  const { data, error } = calendar;
  const days = new Map<string, CalendarDay>(data?.days.map((d) => [d.date, d]));

  return (
    <div className="flex flex-col gap-1">
      <div className="grid grid-cols-7 gap-1">
        {weekdays.map((w) => (
          <div key={w} className="text-center text-xs text-text-secondary">
            {w}
          </div>
        ))}
      </div>
      {calendarWeeks(month).map((week, i) => (
        <div key={i} className="grid grid-cols-7 gap-1">
          {week.map((d, j) =>
            d ? (
              <DayCell
                key={d}
                date={d}
                day={days.get(d)}
                selected={d === selected}
                onSelect={() => onSelect(d)}
              />
            ) : (
              <div key={`empty-${j}`} />
            ),
          )}
        </div>
      ))}
      {error ? (
        <p
          role="alert"
          className="pt-2 text-center text-sm text-text-secondary"
        >
          カレンダーを読み込めませんでした
        </p>
      ) : null}
    </div>
  );
}

function DayCell({
  date,
  day,
  selected,
  onSelect,
}: {
  date: string;
  day: CalendarDay | undefined;
  selected: boolean;
  onSelect: () => void;
}) {
  const n = Number(date.slice(8));
  // 一覧の順（朝食・昼食・夕食・その他）で、最初に写真がある日記のサムネイル
  const thumbnailURL = day?.diaries.find((d) => d.thumbnail_url)?.thumbnail_url;
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-label={`${formatMonthDay(date)}${day ? `、日記${day.diaries.length}件` : ""}`}
      aria-pressed={selected}
      className={`flex h-[68px] flex-col items-center gap-[3px] rounded-lg p-1 lg:h-[100px] ${
        selected ? "outline-[1.5px] outline-text-primary outline-solid" : ""
      }`}
    >
      <span
        className={`text-xs text-text-primary ${selected ? "font-bold" : ""}`}
      >
        {n}
      </span>
      {thumbnailURL ? (
        <RemoteImage
          src={thumbnailURL}
          alt=""
          className="min-h-0 w-full flex-1 rounded-md object-cover"
        />
      ) : day ? (
        <span className="mt-1 size-1.5 rounded-full bg-text-secondary" />
      ) : null}
    </button>
  );
}

function DayEntries({
  date,
  calendar,
}: {
  date: string;
  calendar: CalendarResult;
}) {
  const { data, error, loading } = calendar;
  const diaries = data?.days.find((d) => d.date === date)?.diaries ?? [];

  return (
    <div className="flex flex-col gap-3">
      <h2 className="text-base font-bold text-text-primary">
        {formatMonthDay(date)}
      </h2>
      {error ? (
        <p role="alert" className="text-sm text-text-secondary">
          日記を読み込めませんでした
        </p>
      ) : loading ? null : diaries.length === 0 ? (
        <p className="text-sm text-text-secondary">
          この日の日記はまだありません
        </p>
      ) : (
        <ul className="flex flex-col gap-3">
          {diaries.map((d) => (
            <li key={d.id}>
              <Link
                href={`/diaries/${d.id}`}
                className="flex items-center gap-3 rounded-app bg-surface p-3"
              >
                {d.thumbnail_url ? (
                  <RemoteImage
                    src={d.thumbnail_url}
                    alt=""
                    className="size-14 shrink-0 rounded-lg object-cover"
                  />
                ) : (
                  <span className="size-14 shrink-0 rounded-lg bg-placeholder" />
                )}
                <span className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="truncate text-sm font-bold text-text-primary">
                    {mealLabels[d.status]}
                    {d.info ? ` · ${d.info}` : ""}
                  </span>
                  <span className="truncate text-[13px] text-text-secondary">
                    {d.body_head.split("\n")[0]}
                  </span>
                </span>
                <ChevronRightIcon
                  size={18}
                  className="shrink-0 text-text-secondary"
                />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
