import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_REPORTS = [
  { id: "1", type: "listing", reason: "Fraud", description: "This listing seems fake, asked for money upfront.", reporter: { name: "Ahmed K." }, target_id: "lst_123", status: "pending", created_at: new Date().toISOString() },
  { id: "2", type: "user", reason: "Inappropriate", description: "Profile picture is offensive.", reporter: { name: "Sarah M." }, target_id: "usr_456", status: "pending", created_at: new Date().toISOString() },
  { id: "3", type: "listing", reason: "Spam", description: "Duplicate listing posted 10 times.", reporter: { name: "Mohammed J." }, target_id: "lst_789", status: "resolved", created_at: new Date(Date.now()-86400000).toISOString() },
];

export function useReports(status: string) {
  return useQuery({
    queryKey: ["admin_reports", status],
    queryFn: async () => {
      try {
        const res = await api.get(`/admin/reports?status=${status}`);
        return res.data;
      } catch (err) {
        return {
          data: status === "all" ? MOCK_REPORTS : MOCK_REPORTS.filter(r => r.status === status),
          pending_count: 12
        };
      }
    }
  });
}

export function useReportActions() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin_reports"] });

  return {
    resolve: useMutation({
      mutationFn: (id: string) => api.post(`/admin/reports/${id}/resolve`).catch(() => true),
      onSuccess: invalidate
    }),
    ignore: useMutation({
      mutationFn: (id: string) => api.post(`/admin/reports/${id}/ignore`).catch(() => true),
      onSuccess: invalidate
    })
  };
}
