import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import { ReactNode } from "react";

interface Action {
  label: string;
  icon?: ReactNode;
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
  onClick: () => void;
  className?: string;
}

interface BulkActionsBarProps {
  count: number;
  actions: Action[];
  onClear: () => void;
}

export function BulkActionsBar({ count, actions, onClear }: BulkActionsBarProps) {
  if (count === 0) return null;

  return (
    <div className="bg-primary/10 border border-primary/20 rounded-xl p-3 flex flex-wrap items-center gap-4 animate-in fade-in slide-in-from-top-2 mb-4">
      <span className="text-primary font-semibold ml-2">{count} selected</span>
      <div className="flex flex-wrap items-center gap-2 flex-1">
        {actions.map((action, i) => (
          <Button 
            key={i}
            size="sm" 
            variant={action.variant || "default"} 
            onClick={action.onClick}
            className={action.className}
          >
            {action.icon && <span className="mr-2">{action.icon}</span>}
            {action.label}
          </Button>
        ))}
      </div>
      <Button 
        size="sm" 
        variant="ghost" 
        onClick={onClear} 
        className="text-muted-foreground hover:text-foreground shrink-0"
      >
        <X className="w-4 h-4 mr-2" />
        Clear Selection
      </Button>
    </div>
  );
}
