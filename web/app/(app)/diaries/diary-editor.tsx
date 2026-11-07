"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import {
  createDiary,
  updateDiary,
  type DiaryInput,
  type MealType,
  type Photo,
} from "@/lib/api";
import {
  calendarHref,
  formatMonthDay,
  mealLabels,
  mealTypes,
} from "@/lib/date";
import {
  CalendarIcon,
  ChevronDownIcon,
  ImagePlusIcon,
  NotebookPenIcon,
  UtensilsIcon,
  XIcon,
} from "@/components/icons";
import ConfirmDialog from "@/components/confirm-dialog";
import {
  acceptedTypes,
  checkFiles,
  fromSaved,
  maxPhotos,
  photosChanged,
  savePhotos,
  type PhotoItem,
} from "./photo-edits";
import PhotoSheet from "./photo-sheet";
import PhotoStrip from "./photo-strip";

// 日記の編集画面（docs/v2.pen の「03 日記の編集」「PC 03 日記の編集」）。
// diaryId がないときは新しく書く。あるときはその日記を編集する。
export default function DiaryEditor({
  diaryId,
  initial,
  initialPhotos = [],
}: {
  diaryId?: string;
  initial: DiaryInput;
  initialPhotos?: Photo[];
}) {
  const router = useRouter();
  const [input, setInput] = useState<DiaryInput>(initial);
  const [initialItems] = useState<PhotoItem[]>(() => fromSaved(initialPhotos));
  const [photos, setPhotos] = useState<PhotoItem[]>(initialItems);
  const [openKey, setOpenKey] = useState<string | null>(null);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState(false);
  const [confirmingDiscard, setConfirmingDiscard] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  // 選んだ写真のプレビュー用の URL は、画面を閉じるときに解放する
  const previews = useRef(new Set<string>());
  useEffect(() => {
    const urls = previews.current;
    return () => urls.forEach((u) => URL.revokeObjectURL(u));
  }, []);

  const dirty =
    input.eaten_on !== initial.eaten_on ||
    input.status !== initial.status ||
    (input.info ?? "") !== (initial.info ?? "") ||
    input.body !== initial.body ||
    photosChanged(initialItems, photos);
  const openIndex = photos.findIndex((p) => p.key === openKey);

  function addFiles(files: File[]) {
    const problem = checkFiles(files, photos.length);
    setPhotoError(problem);
    if (problem) {
      return;
    }
    const added: PhotoItem[] = files.map((file) => {
      const previewUrl = URL.createObjectURL(file);
      previews.current.add(previewUrl);
      return { key: previewUrl, kind: "added", file, previewUrl, alt: null };
    });
    setPhotos((prev) => [...prev, ...added]);
  }

  function updateOpen(change: (item: PhotoItem) => PhotoItem | null) {
    setPhotos((prev) =>
      prev.flatMap((p) => {
        if (p.key !== openKey) {
          return [p];
        }
        const next = change(p);
        return next ? [next] : [];
      }),
    );
  }
  // 本文は空白以外の文字が1文字以上（\s は全角スペースも含む）
  const canSave = /\S/.test(input.body) && !saving;
  const closeHref = diaryId
    ? `/diaries/${diaryId}`
    : calendarHref(initial.eaten_on);

  function set<K extends keyof DiaryInput>(key: K, value: DiaryInput[K]) {
    setInput((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSave() {
    if (!canSave) {
      return;
    }
    setSaving(true);
    setSaveError(false);
    const body: DiaryInput = {
      ...input,
      info: input.info?.trim() ? input.info : null,
    };
    let id: string;
    try {
      const { diary } = diaryId
        ? await updateDiary(diaryId, body)
        : await createDiary(body);
      id = diary.id;
    } catch {
      setSaving(false);
      setSaveError(true);
      return;
    }
    try {
      await savePhotos(id, initialItems, photos);
      router.replace(`/diaries/${id}`);
    } catch {
      // 日記は保存できている。写真は途中まで反映されているかもしれないので、
      // 詳細画面で実際に保存された状態を見せる
      router.replace(`/diaries/${id}?photoError=1`);
    }
  }

  function handleClose() {
    if (dirty) {
      setConfirmingDiscard(true);
      return;
    }
    router.replace(closeHref);
  }

  return (
    <main className="mx-auto flex w-full max-w-[480px] flex-1 flex-col lg:max-w-[720px] lg:py-4">
      <div className="flex h-[52px] items-center justify-between px-2 lg:px-0">
        <button
          type="button"
          aria-label="閉じる"
          onClick={handleClose}
          disabled={saving}
          className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-surface disabled:opacity-40"
        >
          <XIcon />
        </button>
        <button
          type="button"
          onClick={handleSave}
          disabled={!canSave}
          className="h-9 rounded-full bg-text-primary px-[18px] text-[15px] font-bold text-on-primary disabled:opacity-40"
        >
          {saving ? "保存中…" : "保存"}
        </button>
      </div>

      {saveError ? (
        <p role="alert" className="px-6 pb-2 text-sm text-danger lg:px-0">
          保存できませんでした。時間をおいてもう一度お試しください
        </p>
      ) : null}

      <div className="flex flex-1 flex-col gap-3.5 px-6 pt-1 pb-4 lg:px-0">
        <div className="flex items-center gap-2">
          <DateChip
            value={input.eaten_on}
            onChange={(v) => set("eaten_on", v)}
          />
          <MealChip value={input.status} onChange={(v) => set("status", v)} />
        </div>
        <label className="flex items-center gap-2.5 border-b border-border py-2.5">
          <NotebookPenIcon size={18} className="shrink-0 text-text-secondary" />
          <input
            type="text"
            aria-label="一言メモ"
            value={input.info ?? ""}
            onChange={(e) => set("info", e.target.value)}
            className="min-w-0 flex-1 bg-transparent text-[15px] text-text-primary outline-none"
          />
        </label>
        <textarea
          aria-label="本文"
          value={input.body}
          onChange={(e) => set("body", e.target.value)}
          className="min-h-[260px] flex-1 resize-none bg-transparent text-[17px] leading-[1.9] text-text-primary outline-none"
        />
      </div>

      <PhotoStrip items={photos} onReorder={setPhotos} onOpen={setOpenKey} />
      {photoError ? (
        <p role="alert" className="px-6 pb-2 text-sm text-danger lg:px-0">
          {photoError}
        </p>
      ) : null}

      <div className="sticky bottom-0 flex h-12 items-center gap-1 border-t border-border bg-surface px-4">
        <button
          type="button"
          aria-label="写真を追加"
          disabled={photos.length >= maxPhotos}
          onClick={() => fileInput.current?.click()}
          className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-bg disabled:opacity-40"
        >
          <ImagePlusIcon />
        </button>
        <input
          ref={fileInput}
          type="file"
          accept={acceptedTypes.join(",")}
          multiple
          hidden
          onChange={(e) => {
            addFiles(Array.from(e.target.files ?? []));
            e.target.value = "";
          }}
        />
        <span className="flex-1" />
        <span className="text-xs text-text-secondary">
          {countChars(input.body)}文字
        </span>
      </div>

      <PhotoSheet
        item={openIndex >= 0 ? photos[openIndex] : null}
        index={openIndex}
        total={photos.length}
        onChangeAlt={(alt) => updateOpen((p) => ({ ...p, alt }))}
        onDelete={() => {
          updateOpen(() => null);
          setOpenKey(null);
        }}
        onClose={() => setOpenKey(null)}
      />

      <ConfirmDialog
        open={confirmingDiscard}
        title="変更を破棄しますか？"
        description="保存していない変更は消えます。"
        confirmLabel="破棄"
        danger
        onCancel={() => setConfirmingDiscard(false)}
        onConfirm={() => router.replace(closeHref)}
      />
    </main>
  );
}

// 本文の文字数。改行は数えない。絵文字なども1文字と数える
function countChars(body: string): number {
  return [...body.replace(/\r?\n/g, "")].length;
}

const chipClass =
  "relative flex h-[30px] items-center gap-1.5 rounded-full bg-surface px-2.5 text-[13px] text-text-primary";

// 日付のチップ。押すとブラウザの日付の選択を開く
function DateChip({
  value,
  onChange,
}: {
  value: string;
  onChange: (date: string) => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  return (
    <button
      type="button"
      className={chipClass}
      onClick={() => ref.current?.showPicker()}
      aria-label={`食事をした日: ${formatMonthDay(value)}`}
    >
      <CalendarIcon size={14} className="text-text-secondary" />
      {formatMonthDay(value)}
      <ChevronDownIcon size={14} className="text-text-secondary" />
      <input
        ref={ref}
        type="date"
        tabIndex={-1}
        aria-hidden="true"
        value={value}
        onChange={(e) => {
          if (e.target.value) {
            onChange(e.target.value);
          }
        }}
        className="pointer-events-none absolute inset-0 opacity-0"
      />
    </button>
  );
}

// 食事の区分のチップ。押すとブラウザの選択肢を開く
function MealChip({
  value,
  onChange,
}: {
  value: MealType;
  onChange: (m: MealType) => void;
}) {
  return (
    <label className={chipClass}>
      <UtensilsIcon size={14} className="text-text-secondary" />
      {mealLabels[value]}
      <ChevronDownIcon size={14} className="text-text-secondary" />
      <select
        aria-label="食事の区分"
        value={value}
        onChange={(e) => onChange(e.target.value as MealType)}
        className="absolute inset-0 cursor-pointer opacity-0"
      >
        {mealTypes.map((m) => (
          <option key={m} value={m}>
            {mealLabels[m]}
          </option>
        ))}
      </select>
    </label>
  );
}
