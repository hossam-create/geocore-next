import { Lock } from "lucide-react"

type ReadOnlyNoticeProps = {
  message?: string
  className?: string
}

export function ReadOnlyNotice({
  message = "Read-only access",
  className = "",
}: ReadOnlyNoticeProps) {
  return (
    <div
      className={`inline-flex items-center gap-1.5 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-1 text-xs text-gray-600 ${className}`}
      role="status"
      aria-live="polite"
    >
      <Lock className="h-3.5 w-3.5" />
      <span>{message}</span>
    </div>
  )
}
