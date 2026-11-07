"use client";

import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ApiError, deleteDiary, getDiary, type DiaryDetail } from "@/lib/api";
import { calendarHref, formatFullDate, mealLabels } from "@/lib/date";
import { useFetch } from "@/lib/use-fetch";
import {
  CalendarIcon,
  ChevronLeftIcon,
  EllipsisVerticalIcon,
  NotebookPenIcon,
  PencilIcon,
  TrashIcon,
  UtensilsIcon,
} from "@/components/icons";
import RemoteImage from "@/components/remote-image";
import ConfirmDialog from "@/components/confirm-dialog";

export default function DiaryDetailView() {
  const { id } = useParams<{ id: string }>();
  const { data, error } = useFetch(id, getDiary);

  if (error) {
    const notFound = error instanceof ApiError && error.status === 404;
    return (
      <Column>
        <TopBar back="/" />
        <p
          role="alert"
          className="px-6 pt-10 text-center text-sm text-text-secondary"
        >
          {notFound
            ? "日記が見つかりませんでした"
            : "日記を読み込めませんでした。時間をおいてもう一度お試しください"}
        </p>
      </Column>
    );
  }
  if (!data) {
    return (
      <Column>
        <TopBar back="/" />
      </Column>
    );
  }
  return <Detail diary={data.diary} />;
}

function Column({ children }: { children: React.ReactNode }) {
  return (
    <main className="mx-auto flex w-full max-w-[480px] flex-1 flex-col lg:max-w-[720px] lg:py-4">
      {children}
    </main>
  );
}

function TopBar({
  back,
  children,
}: {
  back: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex h-[52px] items-center justify-between px-2 lg:px-0">
      <Link
        href={back}
        aria-label="カレンダーに戻る"
        className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-surface"
      >
        <ChevronLeftIcon />
      </Link>
      <div className="flex items-center gap-1">{children}</div>
    </div>
  );
}

function Detail({ diary }: { diary: DiaryDetail }) {
  const router = useRouter();
  // 編集画面で、写真の保存が途中で失敗したときに付く
  const photoError = useSearchParams().get("photoError") === "1";
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState(false);
  const back = calendarHref(diary.eaten_on);

  async function handleDelete() {
    setDeleting(true);
    setDeleteError(false);
    try {
      await deleteDiary(diary.id);
      router.replace(back);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        // もう消えている
        router.replace(back);
        return;
      }
      setDeleting(false);
      setDeleteError(true);
    }
  }

  return (
    <Column>
      <TopBar back={back}>
        <Link
          href={`/diaries/${diary.id}/edit`}
          aria-label="編集する"
          className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-surface"
        >
          <PencilIcon />
        </Link>
        <MoreMenu onDelete={() => setConfirming(true)} />
      </TopBar>

      {photoError ? (
        <p role="alert" className="px-6 pb-2 text-sm text-danger lg:px-0">
          写真の一部を保存できませんでした。編集からもう一度お試しください
        </p>
      ) : null}

      {diary.photos.length > 0 ? (
        <ul className="flex gap-2 overflow-x-auto px-6 py-1 lg:gap-4 lg:px-0 lg:py-2">
          {diary.photos.map((p) => (
            <li key={p.id} className="flex shrink-0 flex-col gap-1">
              <RemoteImage
                src={p.url}
                alt={p.alt ?? ""}
                className="h-[150px] w-[200px] rounded-[10px] object-cover lg:h-[264px] lg:w-[352px]"
              />
              {p.alt ? (
                <span className="w-[200px] text-xs text-text-secondary lg:w-[352px]">
                  {p.alt}
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}

      <article className="flex flex-col gap-3.5 px-6 pt-4 pb-8 lg:px-0 lg:py-6">
        <div className="flex items-center gap-3.5 text-[13px] text-text-secondary">
          <span className="flex items-center gap-[5px]">
            <CalendarIcon size={14} />
            {formatFullDate(diary.eaten_on)}
          </span>
          <span className="flex items-center gap-[5px]">
            <UtensilsIcon size={14} />
            {mealLabels[diary.status]}
          </span>
        </div>
        {diary.info ? (
          <div className="flex items-center gap-2.5 border-b border-border py-2.5">
            <NotebookPenIcon
              size={18}
              className="shrink-0 text-text-secondary"
            />
            <span className="text-[15px] text-text-primary">{diary.info}</span>
          </div>
        ) : null}
        <p className="text-[17px] leading-[1.9] whitespace-pre-wrap text-text-primary">
          {diary.body}
        </p>
      </article>

      <ConfirmDialog
        open={confirming}
        title="この日記を削除しますか？"
        description={
          deleteError
            ? "削除できませんでした。時間をおいてもう一度お試しください"
            : "写真も一緒に削除されます。元には戻せません。"
        }
        confirmLabel="削除"
        danger
        pending={deleting}
        onCancel={() => {
          setConfirming(false);
          setDeleteError(false);
        }}
        onConfirm={handleDelete}
      />
    </Column>
  );
}

// 右上の「⋮」。押すと「削除」を出す。外側を押すか Esc で閉じる
function MoreMenu({ onDelete }: { onDelete: () => void }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    function close(e: MouseEvent | KeyboardEvent) {
      if (
        e instanceof KeyboardEvent
          ? e.key === "Escape"
          : !ref.current?.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", close);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", close);
    };
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        aria-label="メニュー"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-surface"
      >
        <EllipsisVerticalIcon />
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute top-11 right-0 z-10 w-40 rounded-app border border-border bg-bg p-1.5 shadow-[0_4px_16px_#00000022]"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setOpen(false);
              onDelete();
            }}
            className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-[15px] text-danger hover:bg-surface"
          >
            <TrashIcon size={18} />
            削除
          </button>
        </div>
      ) : null}
    </div>
  );
}
