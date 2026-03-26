import { useEffect, useState } from "react";
import { getCountdown } from "@/lib/utils";

export function CountdownTimer({ endsAt, className = "" }: { endsAt: string; className?: string }) {
  const [timeLeft, setTimeLeft] = useState(getCountdown(endsAt));
  const isUrgent = new Date(endsAt).getTime() - Date.now() < 300_000;
  const isEnded = timeLeft === "Ended";

  useEffect(() => {
    const t = setInterval(() => setTimeLeft(getCountdown(endsAt)), 1000);
    return () => clearInterval(t);
  }, [endsAt]);

  return (
    <p className={`text-xs font-semibold mt-1 ${isEnded ? "text-gray-400" : isUrgent ? "text-red-500" : "text-blue-600"} ${className}`}>
      ⏰ {timeLeft}
    </p>
  );
}
