import type { MetadataRoute } from "next";

// ホーム画面に追加したときの名前とアイコン。
// share_target で、写真アプリの「共有」の共有先に出る（Android の Chrome）。
// 共有した写真は POST /share-target に届き、Service Worker（public/sw.js）が受け取って /share を開く
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "meshi-diary",
    short_name: "meshi-diary",
    description: "誰かとご飯を食べたときにつける日記",
    start_url: "/",
    display: "standalone",
    background_color: "#ffffff",
    theme_color: "#ffffff",
    icons: [
      { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
      {
        src: "/icon-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "maskable",
      },
    ],
    share_target: {
      action: "/share-target",
      method: "POST",
      enctype: "multipart/form-data",
      params: {
        files: [
          {
            name: "photos",
            accept: ["image/jpeg", "image/png", "image/webp"],
          },
        ],
      },
    },
  };
}
