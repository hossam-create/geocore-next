import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: string;
  size?: "sm" | "md" | "lg";
  className?: string;
}

export function StatusBadge({ status, size = "md", className }: StatusBadgeProps) {
  const sizeClasses = {
    sm: "px-2 py-0.5 text-xs",
    md: "px-2.5 py-1 text-sm",
    lg: "px-3 py-1.5 text-base"
  };

  const getStatusStyles = (s: string) => {
    switch (s.toLowerCase()) {
      case "pending":
        return "bg-yellow-100 text-yellow-800 hover:bg-yellow-200 border-yellow-200";
      case "active":
      case "live":
        return "bg-emerald-100 text-emerald-800 hover:bg-emerald-200 border-emerald-200";
      case "sold":
      case "upcoming":
        return "bg-blue-100 text-blue-800 hover:bg-blue-200 border-blue-200";
      case "expired":
      case "ended":
        return "bg-gray-100 text-gray-800 hover:bg-gray-200 border-gray-200";
      case "rejected":
        return "bg-red-100 text-red-800 hover:bg-red-200 border-red-200";
      default:
        return "bg-gray-100 text-gray-800 border-gray-200 hover:bg-gray-200";
    }
  };

  const isLive = status.toLowerCase() === "live" || status.toLowerCase() === "active";

  return (
    <Badge 
      variant="outline" 
      className={cn(
        "font-medium capitalize", 
        sizeClasses[size], 
        getStatusStyles(status), 
        className
      )}
    >
      {isLive && (
        <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1.5 animate-pulse" />
      )}
      {status}
    </Badge>
  );
}
