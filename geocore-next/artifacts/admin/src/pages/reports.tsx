import { useState } from "react";
import { useReports, useReportActions } from "@/hooks/use-reports";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tag, User, CheckCircle2, XCircle } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useToast } from "@/hooks/use-toast";

const TABS = ["pending", "reviewing", "resolved", "all"];

export default function ReportsPage() {
  const [status, setStatus] = useState("pending");
  const { data, isLoading } = useReports(status);
  const actions = useReportActions();
  const { toast } = useToast();

  const handleAction = (id: string, type: 'resolve' | 'ignore') => {
    const mutation = type === 'resolve' ? actions.resolve : actions.ignore;
    mutation.mutate(id, {
      onSuccess: () => toast({ title: `Report marked as ${type}d` })
    });
  };

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold font-display tracking-tight text-foreground flex items-center gap-3">
          Reports Queue
          {data?.pending_count > 0 && (
            <span className="bg-destructive text-destructive-foreground text-sm px-3 py-1 rounded-full font-bold">
              {data.pending_count} pending
            </span>
          )}
        </h1>
      </div>

      <div className="flex gap-2 bg-card p-1.5 rounded-xl border w-fit shadow-sm">
        {TABS.map(s => (
          <button
            key={s}
            onClick={() => setStatus(s)}
            className={`px-5 py-2 rounded-lg text-sm font-semibold capitalize transition-all ${
              status === s 
                ? "bg-primary text-primary-foreground shadow-md" 
                : "text-muted-foreground hover:text-foreground hover:bg-muted"
            }`}
          >
            {s}
          </button>
        ))}
      </div>

      <div className="grid gap-4">
        {isLoading ? (
          <div className="text-center py-12 text-muted-foreground">Loading reports...</div>
        ) : data?.data?.length === 0 ? (
          <Card className="p-12 text-center border-none shadow-sm flex flex-col items-center">
            <CheckCircle2 className="w-16 h-16 text-emerald-500 mb-4 opacity-50" />
            <h3 className="text-xl font-bold font-display">All caught up!</h3>
            <p className="text-muted-foreground">No reports matching this filter.</p>
          </Card>
        ) : data?.data?.map((report: any) => (
          <Card key={report.id} className="p-5 border-none shadow-sm flex flex-col sm:flex-row gap-5 hover:shadow-md transition-shadow">
            <div className={`w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 ${
              report.type === "listing" ? "bg-orange-100 text-orange-600 dark:bg-orange-900/30" : "bg-destructive/10 text-destructive"
            }`}>
              {report.type === "listing" ? <Tag className="w-6 h-6" /> : <User className="w-6 h-6" />}
            </div>

            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <span className="font-bold text-base text-foreground">{report.reason}</span>
                <Badge variant="outline" className="capitalize bg-background text-xs">{report.type}</Badge>
                <Badge variant="secondary" className="capitalize text-xs ml-auto sm:ml-0">{report.status}</Badge>
              </div>
              <p className="text-foreground text-sm bg-muted/50 p-3 rounded-lg border border-border/50 mb-3">{report.description}</p>
              <p className="text-muted-foreground text-xs font-medium">
                Reported by {report.reporter.name} • {formatDistanceToNow(new Date(report.created_at), {addSuffix: true})}
              </p>
            </div>

            {report.status === 'pending' && (
              <div className="flex sm:flex-col gap-2 shrink-0 justify-end sm:justify-start">
                <Button className="bg-emerald-500 hover:bg-emerald-600 text-white w-full sm:w-auto" onClick={() => handleAction(report.id, 'resolve')}>
                  <CheckCircle2 className="w-4 h-4 mr-2" /> Resolve
                </Button>
                <Button variant="outline" className="w-full sm:w-auto" onClick={() => handleAction(report.id, 'ignore')}>
                  <XCircle className="w-4 h-4 mr-2" /> Ignore
                </Button>
              </div>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}
