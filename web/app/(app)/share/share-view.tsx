"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { getDiary, listDiaries, type DayDiary } from "@/lib/api";
import { formatMonthDay, mealLabels, todayJST } from "@/lib/date";
import { takeSharedPhotos } from "@/lib/shared-photos";
import { useFetch } from "@/lib/use-fetch";
import { ImagePlusIcon, XIcon } from "@/components/icons";
import RemoteImage from "@/components/remote-image";
import DiaryEditor from "../diaries/diary-editor";
import {
  checkFiles,
  toAddedItem,
  type PhotoItem,
} from "../diaries/photo-edits";

type AddedItem = PhotoItem & { kind: "added" };

// 共有した写真を受け取り、今日の日記がなければ「日記を書く」画面を開く。
// 今日の日記があれば、新しく書くか、どの日記に追加するかを選んでもらう
export default function ShareView() {
  const router = useRouter();
  const [today] = useState(todayJST);
  const [items, setItems] = useState<AddedItem[] | null>(null);
  // 選んだもの。"new" は新しく書く、それ以外は追加する日記の id
  const [choice, setChoice] = useState<string | null>(null);

  useEffect(() => {
    let ignore = false;
    let created: AddedItem[] = [];
    takeSharedPhotos().then((files) => {
      if (ignore) {
        return;
      }
      if (files.length === 0) {
        // 写真がない（開き直したなど）ときは、カレンダーに戻る
        router.replace("/");
        return;
      }
      created = files.map(toAddedItem);
      setItems(created);
    });
    return () => {
      ignore = true;
      created.forEach((i) => URL.revokeObjectURL(i.previewUrl));
    };
  }, [router]);

  const { data, error } = useFetch(items ? today : null, listDiaries);

  if (error) {
    return (
      <main className="flex flex-1 flex-col items-center gap-4 px-6 pt-16 text-center">
        <p role="alert" className="text-sm text-text-secondary">
          日記を読み込めませんでした。時間をおいてもう一度お試しください
        </p>
        <Link href="/" className="text-sm text-text-primary underline">
          カレンダーに戻る
        </Link>
      </main>
    );
  }
  if (!items || !data) {
    return null;
  }

  const files = items.map((i) => i.file);
  if (choice === "new" || (choice === null && data.diaries.length === 0)) {
    const problem = checkFiles(files, 0);
    return (
      <DiaryEditor
        initial={{ eaten_on: today, status: "OTHER", info: null, body: "" }}
        initialAdded={problem ? [] : items}
        initialPhotoError={problem}
      />
    );
  }
  if (choice !== null) {
    return <AddToDiary diaryId={choice} items={items} />;
  }
  return (
    <ChooseDiary
      date={today}
      items={items}
      diaries={data.diaries}
      onChoose={setChoice}
      onCancel={() => router.replace("/")}
    />
  );
}

// 選んだ日記の編集画面を、共有した写真を今ある写真のうしろに足した状態で開く
function AddToDiary({
  diaryId,
  items,
}: {
  diaryId: string;
  items: AddedItem[];
}) {
  const { data, error } = useFetch(diaryId, getDiary);
  if (error) {
    return (
      <main className="flex flex-1 flex-col items-center gap-4 px-6 pt-16 text-center">
        <p role="alert" className="text-sm text-text-secondary">
          日記を読み込めませんでした。時間をおいてもう一度お試しください
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
  const problem = checkFiles(
    items.map((i) => i.file),
    diary.photos.length,
  );
  return (
    <DiaryEditor
      diaryId={diary.id}
      initial={{
        eaten_on: diary.eaten_on,
        status: diary.status,
        info: diary.info,
        body: diary.body,
      }}
      initialPhotos={diary.photos}
      initialAdded={problem ? [] : items}
      initialPhotoError={problem}
    />
  );
}

// 新しく書くか、どの日記に追加するかを選ぶ画面
function ChooseDiary({
  date,
  items,
  diaries,
  onChoose,
  onCancel,
}: {
  date: string;
  items: AddedItem[];
  diaries: DayDiary[];
  onChoose: (choice: string) => void;
  onCancel: () => void;
}) {
  return (
    <main className="mx-auto flex w-full max-w-[480px] flex-1 flex-col">
      <div className="flex h-[52px] items-center justify-between px-2">
        <button
          type="button"
          aria-label="共有をやめる"
          onClick={onCancel}
          className="flex size-10 items-center justify-center rounded-full text-text-primary hover:bg-surface"
        >
          <XIcon />
        </button>
        <h1 className="text-base font-bold text-text-primary">写真を追加</h1>
        <span className="size-10" />
      </div>

      <div className="flex flex-1 flex-col gap-7 px-4 pt-2 pb-6">
        <section className="flex flex-col gap-2.5">
          <p className="text-[13px] text-text-secondary">
            共有した写真（{items.length}枚）
          </p>
          <ul className="flex gap-2 overflow-x-auto">
            {items.map((i) => (
              <li key={i.key} className="shrink-0">
                <RemoteImage
                  src={i.previewUrl}
                  alt=""
                  className="size-20 rounded-lg object-cover"
                />
              </li>
            ))}
          </ul>
        </section>

        <section className="flex flex-col gap-3">
          <h2 className="text-base font-bold text-text-primary">
            {formatMonthDay(date)}の日記に追加する
          </h2>
          <ul className="flex flex-col gap-3">
            {diaries.map((d) => (
              <li key={d.id}>
                <button
                  type="button"
                  onClick={() => onChoose(d.id)}
                  className="flex w-full items-center gap-3 rounded-app bg-surface p-3 text-left"
                >
                  {d.thumbnail_url ? (
                    <RemoteImage
                      src={d.thumbnail_url}
                      alt=""
                      className="size-14 shrink-0 rounded-lg object-cover"
                    />
                  ) : (
                    <span className="size-14 shrink-0 rounded-lg bg-placeholder" />
                  )}
                  <span className="flex min-w-0 flex-1 flex-col gap-1">
                    <span className="truncate text-sm font-bold text-text-primary">
                      {mealLabels[d.status]}
                      {d.info ? ` · ${d.info}` : ""}
                    </span>
                    <span className="truncate text-[13px] text-text-secondary">
                      {d.body.split("\n")[0]}
                    </span>
                  </span>
                  <ImagePlusIcon
                    size={20}
                    className="shrink-0 text-text-secondary"
                  />
                </button>
              </li>
            ))}
          </ul>
        </section>
      </div>

      <div className="sticky bottom-0 bg-bg px-4 pb-6">
        <button
          type="button"
          onClick={() => onChoose("new")}
          className="flex h-[52px] w-full items-center justify-center rounded-app bg-text-primary px-5 text-base font-bold text-on-primary"
        >
          新しく書く
        </button>
      </div>
    </main>
  );
}
