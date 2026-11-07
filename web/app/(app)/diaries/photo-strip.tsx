"use client";

import { useRef, useState } from "react";
import RemoteImage from "@/components/remote-image";
import { itemSrc, maxPhotos, type PhotoItem } from "./photo-edits";

// 写真の並び（docs/v2.pen の「03 日記の編集」「03 日記の編集（写真の並べ替え）」）。
// 押すと onOpen。マウスはそのまま動かす、タッチは長押ししてから動かすと、並べ替えられる。
export default function PhotoStrip({
  items,
  onReorder,
  onOpen,
}: {
  items: PhotoItem[];
  onReorder: (items: PhotoItem[]) => void;
  onOpen: (key: string) => void;
}) {
  const [dragKey, setDragKey] = useState<string | null>(null);
  const press = useRef<{
    key: string;
    x: number;
    y: number;
    dragging: boolean;
    timer?: ReturnType<typeof setTimeout>;
  } | null>(null);
  const cells = useRef(new Map<string, HTMLElement>());

  if (items.length === 0) {
    return null;
  }

  function startDrag() {
    if (press.current) {
      press.current.dragging = true;
      setDragKey(press.current.key);
    }
  }

  function handleDown(e: React.PointerEvent, key: string) {
    e.currentTarget.setPointerCapture(e.pointerId);
    press.current = { key, x: e.clientX, y: e.clientY, dragging: false };
    if (e.pointerType === "touch") {
      press.current.timer = setTimeout(startDrag, 250);
    }
  }

  function handleMove(e: React.PointerEvent) {
    const p = press.current;
    if (!p) {
      return;
    }
    const moved = Math.hypot(e.clientX - p.x, e.clientY - p.y);
    if (!p.dragging) {
      if (e.pointerType !== "touch" && moved > 4) {
        startDrag();
      } else if (e.pointerType === "touch" && moved > 8) {
        clearTimeout(p.timer); // 長押しの前に動いたら、並べ替えにしない
        press.current = null;
      }
      return;
    }
    // 指の位置にいちばん近い写真の場所へ移す
    let target = -1;
    let best = Infinity;
    items.forEach((item, i) => {
      const r = cells.current.get(item.key)?.getBoundingClientRect();
      if (!r) {
        return;
      }
      const d = Math.hypot(
        e.clientX - (r.left + r.width / 2),
        e.clientY - (r.top + r.height / 2),
      );
      if (d < best) {
        best = d;
        target = i;
      }
    });
    const from = items.findIndex((i) => i.key === p.key);
    if (target >= 0 && target !== from) {
      const next = [...items];
      const [moving] = next.splice(from, 1);
      next.splice(target, 0, moving);
      onReorder(next);
    }
  }

  function handleUp() {
    const p = press.current;
    press.current = null;
    setDragKey(null);
    if (!p) {
      return;
    }
    clearTimeout(p.timer);
    if (!p.dragging) {
      onOpen(p.key);
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2 px-6 pb-2.5 lg:px-0">
      {items.map((item, i) => (
        <button
          key={item.key}
          type="button"
          ref={(el) => {
            if (el) {
              cells.current.set(item.key, el);
            } else {
              cells.current.delete(item.key);
            }
          }}
          aria-label={`写真 ${i + 1}`}
          onPointerDown={(e) => handleDown(e, item.key)}
          onPointerMove={handleMove}
          onPointerUp={handleUp}
          onPointerCancel={handleUp}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              onOpen(item.key);
            }
          }}
          onContextMenu={(e) => e.preventDefault()}
          className={`size-12 touch-none overflow-hidden rounded-lg select-none ${
            dragKey === item.key
              ? "z-10 scale-110 -rotate-6 shadow-[0_6px_14px_#00000040] outline-2 outline-text-primary outline-solid"
              : ""
          }`}
        >
          <RemoteImage
            src={itemSrc(item)}
            alt=""
            className="pointer-events-none size-full object-cover"
          />
        </button>
      ))}
      <span className="text-xs text-text-secondary">
        {items.length} / {maxPhotos}
      </span>
    </div>
  );
}
