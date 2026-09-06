import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Navbar } from "@/components/layout/Navbar";

import "./globals.css";
import { MuiProvider } from "@/components/providers/MuiProvider";
import { ToastContainer } from "@/components/ui/ToastContainer";

const geistSans = Geist({ variable: "--font-geist-sans", subsets: ["latin"] });
const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Turtask",
  description: "Project Task board management system",
  icons: {
    icon: "/Turtask.png",
    shortcut: "/Turtask.png",
    apple: "/Turtask.png",
  },
};
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="h-full flex flex-col">
        <MuiProvider>
          <Navbar />
          <div className="flex-1 flex flex-col min-h-0">{children}</div>
          <ToastContainer />
        </MuiProvider>
      </body>
    </html>
  );
}
