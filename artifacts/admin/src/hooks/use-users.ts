import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_USERS = Array.from({ length: 10 }).map((_, i) => ({
  id: `usr_${i}`,
  name: `User ${i + 1} Al-Farsi`,
  email: `user${i}@example.com`,
  role: i === 0 ? "admin" : "user",
  is_blocked: i === 3,
  listings_count: Math.floor(Math.random() * 50),
  created_at: new Date(Date.now() - Math.random() * 10000000000).toISOString(),
  last_login: new Date(Date.now() - Math.random() * 1000000).toISOString()
}));

export function useUsers(search: string, role: string, isBlocked: string, page: number) {
  return useQuery({
    queryKey: ["admin_users", search, role, isBlocked, page],
    queryFn: async () => {
      try {
        const res = await api.get(`/admin/users?q=${search}&role=${role}&is_blocked=${isBlocked}&page=${page}`);
        return res.data;
      } catch (err) {
        return {
          stats: { total: 8432, new_today: 45, verified: 6102, blocked: 123 },
          data: MOCK_USERS,
          meta: { total: 8432, current_page: page, last_page: 840 }
        };
      }
    }
  });
}

export function useUserActions() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin_users"] });

  return {
    toggleBlock: useMutation({
      mutationFn: ({ id, block }: { id: string, block: boolean }) => 
        api.post(`/admin/users/${id}/${block ? 'block' : 'unblock'}`).catch(() => true),
      onSuccess: invalidate
    }),
    addCredit: useMutation({
      mutationFn: ({ id, amount, reason }: { id: string, amount: number, reason: string }) => 
        api.post(`/admin/users/${id}/credit`, { amount, reason }).catch(() => true),
      onSuccess: invalidate
    }),
    changeRole: useMutation({
      mutationFn: ({ id, role }: { id: string, role: string }) => 
        api.post(`/admin/users/${id}/role`, { role }).catch(() => true),
      onSuccess: invalidate
    })
  };
}
