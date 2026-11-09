"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { isDate, todayJST } from "@/lib/date";
import DiaryEditor from "../diary-editor";

// ?date=YYYY-MM-DD の日（カレンダーで選んでいた日）を最初から入れておく
export default function NewDiary() {
  const params = useSearchParams();
  const date = params.get("date");
  // cacheComponents では、前に開いたときの入力が残ったまま画面が出る。
  // ボタンやリンクから開いたときは空から書き始められるように作り直す
  // （ブラウザの戻る・進むで来たときは、書きかけを残す）
  const { bfcacheId } = useRouter();
  return (
    <DiaryEditor
      key={bfcacheId}
      initial={{
        eaten_on: isDate(date) ? date : todayJST(),
        status: "OTHER",
        info: null,
        body: "",
      }}
    />
  );
}
