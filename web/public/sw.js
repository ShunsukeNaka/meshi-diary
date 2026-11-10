// 写真アプリの「共有」で届いた写真を受け取る（app/manifest.ts の share_target）。
// 写真は POST /share-target に multipart で届くので、Cache Storage に入れてから /share を開く。
// /share の画面が lib/shared-photos.ts で取り出して消す。
// それ以外のリクエストには何もしない（ブラウザがそのまま取りに行く）。

const shareCache = "shared-photos";

self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);
  if (
    event.request.method === "POST" &&
    url.origin === self.location.origin &&
    url.pathname === "/share-target"
  ) {
    event.respondWith(receiveShared(event.request));
  }
});

async function receiveShared(request) {
  const form = await request.formData();
  const files = form.getAll("photos").filter((f) => f instanceof File);
  // 前に共有して、取り出されなかった写真は捨てる
  await caches.delete(shareCache);
  const cache = await caches.open(shareCache);
  await Promise.all(
    files.map((file, i) =>
      cache.put(
        `/shared-photos/${i}`,
        new Response(file, {
          headers: {
            "Content-Type": file.type,
            "X-File-Name": encodeURIComponent(file.name),
          },
        }),
      ),
    ),
  );
  return Response.redirect(new URL("/share", self.location.origin).href, 303);
}
