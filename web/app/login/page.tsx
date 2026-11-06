import type { Metadata } from "next";
import GoogleLogin from "./google-login";

export const metadata: Metadata = {
  title: "ログイン | meshi-diary",
};

// デザインは docs/v2.pen の「01 ログイン」
export default function LoginPage() {
  return (
    <main className="mx-auto flex w-full max-w-[390px] flex-1 flex-col justify-between bg-bg px-6 pt-[200px] pb-12">
      <div className="flex flex-col items-center gap-3">
        <UtensilsIcon />
        <h1 className="font-inter text-[28px] font-bold text-text-primary">
          meshi-diary
        </h1>
        <p className="text-sm text-text-secondary">
          誰かとご飯を食べたときの日記
        </p>
      </div>
      <GoogleLogin />
    </main>
  );
}

// lucide の utensils
function UtensilsIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="48"
      height="48"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="text-text-primary"
      aria-hidden="true"
    >
      <path d="M3 2v7c0 1.1.9 2 2 2h4a2 2 0 0 0 2-2V2" />
      <path d="M7 2v20" />
      <path d="M21 15V2a5 5 0 0 0-5 5v6c0 1.1.9 2 2 2h3Zm0 0v7" />
    </svg>
  );
}
