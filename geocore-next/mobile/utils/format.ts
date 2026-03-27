export function formatPrice(price: number, currency = "AED"): string {
  if (price === 0) return "Free";
  return new Intl.NumberFormat("en-AE", {
    style: "currency",
    currency,
    maximumFractionDigits: 0,
  }).format(price);
}

export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return date.toLocaleDateString("en-AE", { day: "numeric", month: "short" });
}

export function getAuctionTimeLeft(endsAt: string): string {
  const end = new Date(endsAt);
  const now = new Date();
  const diff = end.getTime() - now.getTime();

  if (diff <= 0) return "Ended";

  const hours = Math.floor(diff / 3600000);
  const minutes = Math.floor((diff % 3600000) / 60000);
  const days = Math.floor(diff / 86400000);

  if (days > 0) return `${days}d ${hours % 24}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function formatAuctionType(type?: string): string {
  switch (type) {
    case "dutch": return "Dutch";
    case "reverse": return "Reverse";
    default: return "Standard";
  }
}

export function getConditionLabel(condition: string): string {
  switch (condition) {
    case "new": return "Brand New";
    case "like-new": return "Like New";
    case "good": return "Good";
    case "fair": return "Fair";
    case "poor": return "For Parts";
    default: return condition;
  }
}

export function getConditionColor(condition: string): string {
  switch (condition) {
    case "new": return "#10B981";
    case "like-new": return "#059669";
    case "good": return "#3B82F6";
    case "fair": return "#F59E0B";
    case "poor": return "#EF4444";
    default: return "#6B7280";
  }
}
