// 編集画面の写真の状態と、「保存」のときに API へ反映する処理。
import {
  ApiError,
  addPhotos,
  deletePhoto,
  reorderPhotos,
  updatePhotoAlt,
  type Photo,
} from "@/lib/api";

export const maxPhotos = 10;
export const maxPhotoBytes = 10 * 1024 * 1024;
export const acceptedTypes = ["image/jpeg", "image/png", "image/webp"];
// 送る前に縮める大きさ（長辺の px）と、WebP の画質（#109）
export const maxLongSide = 2048;
export const webpQuality = 0.8;

// 画面の写真。保存ずみ（existing）と、選んだだけでまだ送っていない（added）がある
export type PhotoItem =
  | {
      key: string;
      kind: "existing";
      id: string;
      url: string;
      alt: string | null;
      savedAlt: string | null;
    }
  | {
      key: string;
      kind: "added";
      file: File;
      previewUrl: string;
      alt: string | null;
    };

export function fromSaved(photos: Photo[]): PhotoItem[] {
  return photos.map((p) => ({
    key: p.id,
    kind: "existing",
    id: p.id,
    url: p.url,
    alt: p.alt,
    savedAlt: p.alt,
  }));
}

export function itemSrc(item: PhotoItem): string {
  return item.kind === "existing" ? item.url : item.previewUrl;
}

// 写真に変更があるか（× で閉じるときの確認に使う）
export function photosChanged(
  initial: PhotoItem[],
  current: PhotoItem[],
): boolean {
  if (initial.length !== current.length) {
    return true;
  }
  return current.some(
    (item, i) =>
      item.key !== initial[i].key ||
      item.kind === "added" ||
      (item.kind === "existing" && (item.alt ?? "") !== (item.savedAlt ?? "")),
  );
}

// 画面の写真の状態を API に反映する。
// 1. 消した写真を削除 2. 説明を変えた写真を更新 3. 新しい写真を縮めて WebP にし、まとめて追加 4. 画面の順番に並べ替え
export async function savePhotos(
  diaryId: string,
  initial: PhotoItem[],
  current: PhotoItem[],
): Promise<void> {
  const keep = new Set(current.map((i) => i.key));
  for (const item of initial) {
    if (item.kind === "existing" && !keep.has(item.key)) {
      await ignoreNotFound(deletePhoto(diaryId, item.id));
    }
  }
  for (const item of current) {
    if (
      item.kind === "existing" &&
      (item.alt ?? "") !== (item.savedAlt ?? "")
    ) {
      await updatePhotoAlt(diaryId, item.id, emptyToNull(item.alt));
    }
  }

  const added = current.filter((i) => i.kind === "added");
  let saved: Photo[] | null = null;
  if (added.length > 0) {
    // 写真を縮めて WebP にしてから送る。メモリを使いすぎないよう、1枚ずつ変換する
    const photos: { file: File; alt: string | null }[] = [];
    for (const i of added) {
      photos.push({ file: await toWebP(i.file), alt: emptyToNull(i.alt) });
    }
    const res = await addPhotos(diaryId, photos);
    saved = res.photos;
  }

  // 追加した写真は、今ある写真のうしろに送った順で並ぶので、うしろから順に id を割り当てる
  const addedIds = saved
    ? saved.slice(saved.length - added.length).map((p) => p.id)
    : [];
  let n = 0;
  const order = current.map((i) =>
    i.kind === "existing" ? i.id : addedIds[n++],
  );
  // 削除と追加のあとのサーバーの順番。画面の順番と違えば並べ替える
  const serverOrder = saved
    ? saved.map((p) => p.id)
    : initialOrderAfterDelete(initial, keep);
  const sameOrder =
    order.length === serverOrder.length &&
    order.every((id, i) => id === serverOrder[i]);
  if (!sameOrder) {
    await reorderPhotos(diaryId, order);
  }
}

// 削除したあとのサーバーの順番（position を詰めた順）
function initialOrderAfterDelete(
  initial: PhotoItem[],
  keep: Set<string>,
): string[] {
  return initial
    .filter((i) => i.kind === "existing" && keep.has(i.key))
    .map((i) => (i.kind === "existing" ? i.id : ""));
}

// 写真を長辺 maxLongSide px 以下の WebP にする。もっと小さい写真も、大きさは変えずに WebP にする。
// 向き（EXIF）は createImageBitmap が画素に反映するので、縦に撮った写真も正しい向きになる
export async function toWebP(file: File): Promise<File> {
  const bitmap = await createImageBitmap(file, {
    imageOrientation: "from-image",
  });
  try {
    const scale = Math.min(
      1,
      maxLongSide / Math.max(bitmap.width, bitmap.height),
    );
    const width = Math.round(bitmap.width * scale);
    const height = Math.round(bitmap.height * scale);
    const canvas = new OffscreenCanvas(width, height);
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new Error("canvas 2d context is not available");
    }
    ctx.imageSmoothingQuality = "high";
    ctx.drawImage(bitmap, 0, 0, width, height);
    const blob = await canvas.convertToBlob({
      type: "image/webp",
      quality: webpQuality,
    });
    const name = `${file.name.replace(/\.[^.]*$/, "")}.webp`;
    return new File([blob], name, { type: blob.type });
  } finally {
    bitmap.close();
  }
}

function emptyToNull(s: string | null): string | null {
  return s && s.trim() ? s : null;
}

async function ignoreNotFound(p: Promise<unknown>) {
  try {
    await p;
  } catch (e) {
    if (!(e instanceof ApiError && e.status === 404)) {
      throw e;
    }
  }
}

// 選んだファイルを確かめる。追加できないものがあれば、その理由を返す
export function checkFiles(files: File[], currentCount: number): string | null {
  if (currentCount + files.length > maxPhotos) {
    return `写真は1つの日記に${maxPhotos}枚までです`;
  }
  if (files.some((f) => !acceptedTypes.includes(f.type))) {
    return "JPEG・PNG・WebP の写真を選んでください";
  }
  if (files.some((f) => f.size > maxPhotoBytes)) {
    return "10 MB を超える写真は追加できません";
  }
  return null;
}
