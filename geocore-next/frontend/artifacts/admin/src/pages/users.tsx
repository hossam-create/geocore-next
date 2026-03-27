import { useState } from "react";
import { useUsers, useUserActions } from "@/hooks/use-users";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Search, UserPlus, Users as UsersIcon, ShieldAlert, BadgeCheck, MoreVertical, Ban, Coins, Shield } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { format } from "date-fns";
import { useToast } from "@/hooks/use-toast";
import { PageLayout } from "@/components/layout";

export default function UsersPage() {
  const [search, setSearch] = useState("");
  const { data, isLoading } = useUsers(search, "", "", 1);
  const actions = useUserActions();
  const { toast } = useToast();

  return (
    <PageLayout title="Users Management">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="p-5 border-none shadow-sm flex items-center gap-4">
          <div className="w-12 h-12 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 flex items-center justify-center"><UsersIcon /></div>
          <div><p className="text-sm text-muted-foreground font-medium">Total Users</p><p className="text-2xl font-bold font-display">{data?.stats?.total.toLocaleString() || 0}</p></div>
        </Card>
        <Card className="p-5 border-none shadow-sm flex items-center gap-4">
          <div className="w-12 h-12 rounded-full bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 flex items-center justify-center"><UserPlus /></div>
          <div><p className="text-sm text-muted-foreground font-medium">New Today</p><p className="text-2xl font-bold font-display">{data?.stats?.new_today || 0}</p></div>
        </Card>
        <Card className="p-5 border-none shadow-sm flex items-center gap-4">
          <div className="w-12 h-12 rounded-full bg-purple-100 dark:bg-purple-900/30 text-purple-600 flex items-center justify-center"><BadgeCheck /></div>
          <div><p className="text-sm text-muted-foreground font-medium">Verified</p><p className="text-2xl font-bold font-display">{data?.stats?.verified.toLocaleString() || 0}</p></div>
        </Card>
        <Card className="p-5 border-none shadow-sm flex items-center gap-4">
          <div className="w-12 h-12 rounded-full bg-destructive/10 text-destructive flex items-center justify-center"><ShieldAlert /></div>
          <div><p className="text-sm text-muted-foreground font-medium">Blocked</p><p className="text-2xl font-bold font-display">{data?.stats?.blocked || 0}</p></div>
        </Card>
      </div>

      <Card className="p-4 border-none shadow-sm flex justify-between items-center">
        <div className="relative w-72">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input placeholder="Search by name or email..." value={search} onChange={e => setSearch(e.target.value)} className="pl-9 bg-muted/30 border-none" />
        </div>
      </Card>

      <Card className="border-none shadow-sm overflow-hidden">
        <table className="w-full text-sm text-left">
          <thead className="bg-muted/50 text-muted-foreground uppercase text-xs font-semibold">
            <tr>
              <th className="p-4">User</th>
              <th className="p-4">Role</th>
              <th className="p-4">Listings</th>
              <th className="p-4">Joined</th>
              <th className="p-4">Last Login</th>
              <th className="p-4">Status</th>
              <th className="p-4 w-12"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading ? <tr><td colSpan={7} className="p-8 text-center">Loading...</td></tr> : data?.data?.map((user: any) => (
              <tr key={user.id} className="hover:bg-muted/20">
                <td className="p-4 flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-primary/20 text-primary flex items-center justify-center font-bold">
                    {user.name.charAt(0)}
                  </div>
                  <div>
                    <p className="font-semibold text-foreground">{user.name}</p>
                    <p className="text-xs text-muted-foreground">{user.email}</p>
                  </div>
                </td>
                <td className="p-4">
                  <Badge variant={user.role === 'admin' ? 'default' : 'secondary'} className="capitalize">{user.role}</Badge>
                </td>
                <td className="p-4 font-mono">{user.listings_count}</td>
                <td className="p-4 text-muted-foreground">{format(new Date(user.created_at), "MMM d, yyyy")}</td>
                <td className="p-4 text-muted-foreground">{format(new Date(user.last_login), "MMM d, HH:mm")}</td>
                <td className="p-4">
                  {user.is_blocked ? <Badge variant="destructive">Blocked</Badge> : <Badge className="bg-emerald-500 hover:bg-emerald-600">Active</Badge>}
                </td>
                <td className="p-4 text-right">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8"><MoreVertical className="w-4 h-4" /></Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-48">
                      <DropdownMenuLabel>Actions</DropdownMenuLabel>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onClick={() => toast({description: "View profile not implemented"})}>View Profile</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => {
                        actions.addCredit.mutate({id: user.id, amount: 100, reason: "Admin grant"});
                        toast({title: "Added 100 AED credit"});
                      }}>
                        <Coins className="w-4 h-4 mr-2" /> Add Credit
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => {
                        actions.changeRole.mutate({id: user.id, role: user.role === 'admin' ? 'user' : 'admin'});
                        toast({title: `Role changed to ${user.role === 'admin' ? 'user' : 'admin'}`});
                      }}>
                        <Shield className="w-4 h-4 mr-2" /> Toggle Admin
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem 
                        className={user.is_blocked ? "text-emerald-600" : "text-destructive"}
                        onClick={() => {
                          actions.toggleBlock.mutate({id: user.id, block: !user.is_blocked});
                          toast({title: `User ${user.is_blocked ? 'unblocked' : 'blocked'}`});
                        }}
                      >
                        <Ban className="w-4 h-4 mr-2" /> {user.is_blocked ? "Unblock User" : "Block User"}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </PageLayout>
  );
}
