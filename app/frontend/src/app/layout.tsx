import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Inter } from "next/font/google";
import "./globals.css";
import { GraphQLProvider } from "./providers";

const inter = Inter({ subsets: ["latin"], display: "swap" });

export const metadata: Metadata = {
  title: "BalkanID Vault",
  description: "Secure file vault with dedupe, sharing, and analytics"
};

export default function RootLayout({
  children
}: {
  children: ReactNode;
}) {
  return (
    <html lang="en" className={inter.className}>
      <body className="min-h-screen bg-slate-950 text-slate-50 antialiased">
        <GraphQLProvider>{children}</GraphQLProvider>
      </body>
    </html>
  );
}
