"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { ApiError, getDiary } from "@/lib/api";
import { useFetch } from "@/lib/use-fetch";
import DiaryEditor from "../../diary-editor";

export default function EditDiary() {
  const { id } = useParams<{ id: string }>();
  const { data, error } = useFetch(id, getDiary);

  if (error) {
    const notFound = error instanceof ApiError && error.status === 404;
    return (
      <main className="flex flex-1 flex-col items-center gap-4 px-6 pt-16 text-center">
        <p role="alert" className="text-sm text-text-secondary">
          {notFound
            ? "日記が見つかりませんでした"
            : "日記を読み込めませんでした。時間をおいてもう一度お試しください"}
        </p>
        <Link href="/" className="text-sm text-text-primary underline">
          カレンダーに戻る
        </Link>
      </main>
    );
  }
  if (!data) {
    return null;
  }
  const { diary } = data;
  return (
    <DiaryEditor
      key={diary.id}
      diaryId={diary.id}
      initial={{
        eaten_on: diary.eaten_on,
        status: diary.status,
        info: diary.info,
        body: diary.body,
      }}
      initialPhotos={diary.photos}
    />
  );
}
