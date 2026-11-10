// API を呼ぶときの共通処理。仕様は docs/api.md を参照。
// API へのリクエストは Vercel（ローカルでは next dev）が中継するので、同じオリジンの /meshi-diary-api に送る。
// 同じオリジンなので、セッション Cookie は fetch の既定（credentials: "same-origin"）で送られる。

const basePath = "/meshi-diary-api";

// docs/api.md の共通のエラーレスポンス {"error": {"code", "message"}} を表す
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export type User = {
  id: string;
  name: string;
  created_at: string;
};

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  // FormData のときは、ブラウザが multipart の区切りつきの Content-Type を付ける
  const headers: HeadersInit =
    init.body instanceof FormData
      ? { ...init.headers }
      : { "Content-Type": "application/json", ...init.headers };
  const res = await fetch(`${basePath}${path}`, { ...init, headers });

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(
      res.status,
      body?.error?.code ?? "UNKNOWN",
      body?.error?.message ?? res.statusText,
    );
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return res.json();
}

export function login(idToken: string) {
  return request<{ user: User }>("/login", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
}

export function signup(idToken: string) {
  return request<{ user: User }>("/signup", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
}

// ログイン中のユーザーを返す。未ログインなら ApiError（status 401）を投げる
export function me() {
  return request<{ user: User }>("/me");
}

export function logout() {
  return request<void>("/logout", { method: "POST" });
}

export type MealType = "BREAKFAST" | "LUNCH" | "DINNER" | "OTHER";

export type Diary = {
  id: string;
  eaten_on: string;
  status: MealType;
  info: string | null;
  body: string;
  created_at: string;
  updated_at: string;
};

export type Photo = {
  id: string;
  url: string;
  alt: string | null;
  position: number;
};

export type DiaryDetail = Diary & { photos: Photo[] };

// カレンダーの日の日記。一覧に出す分だけ（docs/api.md の GET /calendar の days[].diaries[]）
export type CalendarDiary = Pick<Diary, "id" | "status" | "info"> & {
  body_head: string;
  thumbnail_url: string | null;
};

export type CalendarDay = {
  date: string;
  diaries: CalendarDiary[];
};

export type CalendarMonth = { month: string; days: CalendarDay[] };

// month は YYYY-MM
export function getCalendar(month: string) {
  return request<CalendarMonth>(`/calendar?month=${encodeURIComponent(month)}`);
}

// その日の日記（docs/api.md の GET /diaries の diaries[]）
export type DayDiary = Diary & { thumbnail_url: string | null };

// date は YYYY-MM-DD
export function listDiaries(date: string) {
  return request<{ date: string; diaries: DayDiary[] }>(
    `/diaries?date=${encodeURIComponent(date)}`,
  );
}

export function getDiary(id: string) {
  return request<{ diary: DiaryDetail }>(`/diaries/${encodeURIComponent(id)}`);
}

export type DiaryInput = {
  eaten_on: string;
  status: MealType;
  info: string | null;
  body: string;
};

export function createDiary(input: DiaryInput) {
  return request<{ diary: Diary }>("/diaries", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateDiary(id: string, input: Partial<DiaryInput>) {
  return request<{ diary: Diary }>(`/diaries/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export function deleteDiary(id: string) {
  return request<void>(`/diaries/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

// 写真をまとめて追加する。1枚でもエラーなら1枚も追加されない（docs/api.md）。
// 返るのは、追加したあとの日記の写真すべて（表示順）
export function addPhotos(
  diaryId: string,
  photos: { file: File; alt: string | null }[],
) {
  const form = new FormData();
  for (const p of photos) {
    form.append("photos", p.file);
    form.append("alts", p.alt ?? "");
  }
  return request<{ photos: Photo[] }>(
    `/diaries/${encodeURIComponent(diaryId)}/photos`,
    { method: "POST", body: form },
  );
}

export function updatePhotoAlt(
  diaryId: string,
  photoId: string,
  alt: string | null,
) {
  return request<{ photo: Photo }>(
    `/diaries/${encodeURIComponent(diaryId)}/photos/${encodeURIComponent(photoId)}`,
    { method: "PATCH", body: JSON.stringify({ alt }) },
  );
}

export function deletePhoto(diaryId: string, photoId: string) {
  return request<void>(
    `/diaries/${encodeURIComponent(diaryId)}/photos/${encodeURIComponent(photoId)}`,
    { method: "DELETE" },
  );
}

export function reorderPhotos(diaryId: string, photoIds: string[]) {
  return request<{ photos: Photo[] }>(
    `/diaries/${encodeURIComponent(diaryId)}/photos/order`,
    { method: "PUT", body: JSON.stringify({ photo_ids: photoIds }) },
  );
}
