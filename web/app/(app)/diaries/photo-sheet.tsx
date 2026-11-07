"use client";

import { useEffect, useRef } from "react";
import RemoteImage from "@/components/remote-image";
import { TrashIcon } from "@/components/icons";
import { itemSrc, type PhotoItem } from "./photo-edits";

// 写真を押したときに下から出る画面（docs/v2.pen の「03 日記の編集（写真の説明）」）。
// 説明の入力と削除ができる。変更は日記の「保存」でまとめて反映する
export default function PhotoSheet({
  item,
  index,
  total,
  onChangeAlt,
  onDelete,
  onClose,
}: {
  item: PhotoItem | null;
  index: number;
  total: number;
  onChangeAlt: (alt: string) => void;
  onDelete: () => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) {
      return;
    }
    if (item && !dialog.open) {
      dialog.showModal();
    } else if (!item && dialog.open) {
      dialog.close();
    }
  }, [item]);

  return (
    <dialog
      ref={ref}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      onClick={(e) => {
        // うしろの暗いところを押したら閉じる
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
      className="mx-auto mt-auto mb-0 w-full max-w-[480px] rounded-t-[20px] bg-bg p-0 backdrop:bg-black/40"
    >
      {item ? (
        <div className="flex flex-col gap-4 px-6 pt-3 pb-8">
          <div className="flex justify-center">
            <span className="h-1 w-9 rounded-full bg-border" />
          </div>
          <div className="flex items-center justify-between">
            <h2 className="text-base font-bold text-text-primary">
              写真 {index + 1} / {total}
            </h2>
            <button
              type="button"
              onClick={onDelete}
              className="flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-sm text-danger hover:bg-surface"
            >
              <TrashIcon size={18} />
              削除
            </button>
          </div>
          <RemoteImage
            src={itemSrc(item)}
            alt={item.alt ?? ""}
            className="h-[240px] w-full rounded-app object-contain"
          />
          <label className="flex flex-col gap-1.5">
            <span className="text-[13px] text-text-secondary">説明</span>
            <input
              type="text"
              value={item.alt ?? ""}
              onChange={(e) => onChangeAlt(e.target.value)}
              className="rounded-[10px] border border-border px-3.5 py-3 text-[15px] text-text-primary outline-none focus:border-text-primary"
            />
          </label>
          <button
            type="button"
            onClick={onClose}
            className="flex h-[52px] items-center justify-center rounded-app bg-text-primary text-base font-bold text-on-primary"
          >
            完了
          </button>
        </div>
      ) : null}
    </dialog>
  );
}
