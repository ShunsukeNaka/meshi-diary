"use client";

import { useState } from "react";

// R2 の署名付き URL の写真を表示する。読み込めなかったとき（期限切れなど）は、灰色の四角にする。
// 署名付き URL は毎回変わり、外部のドメインなので、next/image の最適化は通さない。
export default function RemoteImage({
  src,
  alt,
  className,
}: {
  src: string;
  alt: string;
  className?: string;
}) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  if (failedSrc === src) {
    return (
      <span
        role="img"
        aria-label={alt}
        className={`block bg-placeholder ${className ?? ""}`}
      />
    );
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={src}
      alt={alt}
      className={`bg-placeholder ${className ?? ""}`}
      onError={() => setFailedSrc(src)}
    />
  );
}
