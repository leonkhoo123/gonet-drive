// src/layouts/DefaultLayout.tsx
import React from "react";

export default function DefaultLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col h-dvh overflow-hidden p-0 main-view border-x border-border">
      <main className="sm:shadow-2xl flex-1 flex flex-col overflow-hidden">
        {children}
      </main>
    </div>
  );
}
