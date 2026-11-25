import type { Metadata, Viewport } from "next";
import { Lexend, Literata } from "next/font/google";
import "./globals.css";

// Fonts are downloaded at build time and served from this app; the browser never calls Google.
const lexend = Lexend({ subsets: ["latin", "vietnamese"], variable: "--font-lexend", display: "swap" });
const literata = Literata({
  subsets: ["latin", "vietnamese"],
  variable: "--font-literata",
  display: "swap",
  style: ["normal", "italic"],
});

export const metadata: Metadata = {
  title: {
    default: "Đại học số Trưng Vương",
    template: "%s · Đại học số Trưng Vương",
  },
  description: "Hệ thống quản lý học tập của Đại học số Trưng Vương",
  robots: { index: false, follow: false },
};

export const viewport: Viewport = {
  themeColor: "#4e2c8e",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="vi" className={`h-full antialiased ${lexend.variable} ${literata.variable}`}>
      <body className="flex min-h-full flex-col font-sans">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded focus:bg-paper focus:px-3 focus:py-2 focus:text-ink-700"
        >
          Bỏ qua tới nội dung chính
        </a>
        {children}
      </body>
    </html>
  );
}
