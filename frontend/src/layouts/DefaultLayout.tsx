// src/layouts/DefaultLayout.tsx
import React from "react";

export default function DefaultLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col h-dvh overflow-hidden p-0 main-view border-x border-border relative bg-gradient-to-br from-primary/5 via-background to-primary/10">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 right-0 w-96 h-96 bg-primary/5 rounded-full blur-3xl translate-x-1/3 -translate-y-1/3" />
        <div className="absolute bottom-0 left-0 w-72 h-72 bg-primary/5 rounded-full blur-3xl -translate-x-1/3 translate-y-1/3" />
      </div>
      <main className="sm:shadow-2xl flex-1 flex flex-col overflow-hidden relative">
        {children}
      </main>
    </div>
  );
}
