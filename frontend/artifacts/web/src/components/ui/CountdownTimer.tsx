import { useEffect, useState } from "react";
import { getCountdown } from "@/lib/utils";
import { Clock } from "lucide-react";

interface CountdownTimerProps {
  endsAt: string;
  className?: string;
  compact?: boolean;
}

export function CountdownTimer({ endsAt, className = "", compact = false }: CountdownTimerProps) {
  const [timeLeft, setTimeLeft] = useState(getCountdown(endsAt));
  const isUrgent = new Date(endsAt).getTime() - Date.now() < 300_000;
  const isEnded = timeLeft === "Ended";

  useEffect(() => {
    const t = setInterval(() => setTimeLeft(getCountdown(endsAt)), 1000);
    return () => clearInterval(t);
  }, [endsAt]);

  if (compact) {
    return (
      <p className={`text-[10px] font-bold flex items-center gap-1 ${isEnded ? "text-gray-300" : isUrgent ? "text-red-300" : "text-white/90"} ${className}`}>
        <Clock size={10} />
        {isEnded ? "Ended" : timeLeft}
      </p>
    );
  }

  return (
    <p className={`text-xs font-semibold mt-1 flex items-center gap-1 ${isEnded ? "text-gray-400" : isUrgent ? "text-red-500" : "text-[#0071CE]"} ${className}`}>
      <Clock size={11} />
      {timeLeft}
    </p>
  );
}
