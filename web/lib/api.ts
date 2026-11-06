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
  const res = await fetch(`${basePath}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init.headers,
    },
  });

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
