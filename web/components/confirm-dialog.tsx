"use client";

import { useEffect, useRef } from "react";

// 確認のダイアログ（docs/v2.pen の「04 日記の詳細（削除の確認）」）。
// <dialog> の showModal を使うので、うしろは操作できず、Esc で閉じる（onCancel）。
export default function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  danger = false,
  pending = false,
  onCancel,
  onConfirm,
}: {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  danger?: boolean;
  pending?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) {
      return;
    }
    if (open && !dialog.open) {
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  return (
    <dialog
      ref={ref}
      onCancel={(e) => {
        e.preventDefault();
        onCancel();
      }}
      className="m-auto w-[calc(100%-64px)] max-w-[360px] rounded-2xl bg-bg p-6 backdrop:bg-black/40"
    >
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <h2 className="text-[17px] font-bold text-text-primary">{title}</h2>
          <p className="text-sm leading-[1.6] text-text-secondary">
            {description}
          </p>
        </div>
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={pending}
            className="rounded-[10px] bg-surface px-[18px] py-2.5 text-[15px] text-text-primary disabled:opacity-50"
          >
            キャンセル
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={pending}
            className={`rounded-[10px] px-[18px] py-2.5 text-[15px] font-bold text-on-primary disabled:opacity-50 ${
              danger ? "bg-danger" : "bg-text-primary"
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  );
}
