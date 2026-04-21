"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function ResetSessionPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/login");
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center text-sm" style={{ color: "var(--text-secondary)" }}>
      Resetting session...
    </div>
  );
}
