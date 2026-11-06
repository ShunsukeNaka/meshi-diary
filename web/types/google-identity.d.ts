// Google Identity Services（https://accounts.google.com/gsi/client）のうち、使う部分だけの型
// https://developers.google.com/identity/gsi/web/reference/js-reference

declare namespace google.accounts.id {
  type CredentialResponse = {
    // ID トークン（JWT）
    credential: string;
  };

  type IdConfiguration = {
    client_id: string;
    callback: (response: CredentialResponse) => void;
  };

  type GsiButtonConfiguration = {
    type?: "standard" | "icon";
    theme?: "outline" | "filled_blue" | "filled_black";
    size?: "large" | "medium" | "small";
    text?: "signin_with" | "signup_with" | "continue_with" | "signin";
    shape?: "rectangular" | "pill" | "circle" | "square";
    logo_alignment?: "left" | "center";
    // px。最大 400
    width?: number;
    locale?: string;
  };

  function initialize(config: IdConfiguration): void;
  function renderButton(
    parent: HTMLElement,
    options: GsiButtonConfiguration,
  ): void;
}
