import { NextRequest, NextResponse } from "next/server";

export async function middleware(request: NextRequest) {
  // Set locale cookie from Accept-Language header if not already set
  const localeCookie = request.cookies.get("NEXT_LOCALE")?.value;
  const response = NextResponse.next();

  if (!localeCookie) {
    const acceptLang = request.headers.get("accept-language") || "";
    if (acceptLang.startsWith("ar")) {
      response.cookies.set("NEXT_LOCALE", "ar", { path: "/" });
    } else {
      response.cookies.set("NEXT_LOCALE", "en", { path: "/" });
    }
  }

  return response;
}

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
