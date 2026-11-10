// 写真アプリの「共有」で届いた写真を取り出す。
// Service Worker（public/sw.js）が Cache Storage の shared-photos に入れたものを、File にして返し、消す。

const shareCache = "shared-photos";

let taking: Promise<File[]> | null = null;

// 何度呼んでも、同じ写真を返す（開発中の Strict Mode で effect が2回動いても、写真がなくならないように）
export function takeSharedPhotos(): Promise<File[]> {
  taking ??= take();
  return taking;
}

async function take(): Promise<File[]> {
  if (!("caches" in window)) {
    return [];
  }
  const cache = await caches.open(shareCache);
  const keys = [...(await cache.keys())];
  const index = (r: Request) =>
    Number(new URL(r.url).pathname.split("/").pop());
  keys.sort((a, b) => index(a) - index(b));
  const files: File[] = [];
  for (const key of keys) {
    const res = await cache.match(key);
    if (!res) {
      continue;
    }
    const blob = await res.blob();
    const name = decodeURIComponent(res.headers.get("X-File-Name") ?? "photo");
    files.push(new File([blob], name, { type: blob.type }));
  }
  await caches.delete(shareCache);
  return files;
}
