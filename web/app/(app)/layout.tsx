import AuthGuard from "./auth-guard";

// (app) の下の画面は、ログインしていないと開けない
export default function AppLayout({ children }: LayoutProps<"/">) {
  return <AuthGuard>{children}</AuthGuard>;
}
