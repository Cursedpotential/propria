import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "TraceIQ Workspace",
  description: "Dual-use agent-native investigation workspace",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased" suppressHydrationWarning>
      <body className="h-full overflow-hidden bg-bg text-ink">{children}</body>
    </html>
  );
}
