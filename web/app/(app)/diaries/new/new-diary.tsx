"use client";

import { useSearchParams } from "next/navigation";
import { isDate, todayJST } from "@/lib/date";
import DiaryEditor from "../diary-editor";

// ?date=YYYY-MM-DD の日（カレンダーで選んでいた日）を最初から入れておく
export default function NewDiary() {
  const params = useSearchParams();
  const date = params.get("date");
  return (
    <DiaryEditor
      initial={{
        eaten_on: isDate(date) ? date : todayJST(),
        status: "OTHER",
        info: null,
        body: "",
      }}
    />
  );
}
