import type { NextConfig } from "next";

// API へのリクエストを中継する先（docs/api.md の「ベース URL」）。
// フロント（Vercel）と API は同じサイトに置けないので、/meshi-diary-api/* を
// Vercel が API に中継し、ブラウザからは同じオリジンへのリクエストにする。
// 本番は Apache が Cookie の Path を /meshi-diary-api/ に書き換えるので、中継のパスもそろえる。
function apiBaseUrl(): string | undefined {
  const url = process.env.API_BASE_URL?.replace(/\/+$/, "");
  if (url) {
    return url;
  }
  if (process.env.NODE_ENV !== "production") {
    return "http://localhost:8080";
  }
  console.warn(
    "API_BASE_URL が設定されていないので、/meshi-diary-api/* を中継しません",
  );
  return undefined;
}

const nextConfig: NextConfig = {
  /* config options here */
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  async rewrites() {
    const base = apiBaseUrl();
    if (!base) {
      return [];
    }
    return [
      {
        source: "/meshi-diary-api/:path*",
        destination: `${base}/:path*`,
      },
    ];
  },
};

export default nextConfig;
