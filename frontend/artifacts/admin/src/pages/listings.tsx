import { useState } from "react";
import { useListings, useListingActions } from "@/hooks/use-listings";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Search, Download, CheckCircle, XCircle, Eye, Star, Check, X, Tag, MoreHorizontal, ExternalLink, Trash2 } from "lucide-react";
import { format } from "date-fns";
import { useToast } from "@/hooks/use-toast";
import { PageLayout } from "@/components/layout";
import { EmptyState } from "@/components/ui/EmptyState";
import { BulkActionsBar } from "@/components/ui/BulkActionsBar";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { useLocation } from "wouter";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";

const TABS = ["pending", "active", "sold", "expired", "rejected"];

export default function ListingsPage() {
  const [status, setStatus] = useState("pending");
  const [search, setSearch] = useState("");
  const [type, setType] = useState("all");
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<string[]>([]);
  const { toast } = useToast();
  const [, setLocation] = useLocation();

  const { data, isLoading } = useListings(status, search, page);
  const actions = useListingActions();

  const handleBulkAction = (action: 'approve' | 'reject') => {
    const mutation = action === 'approve' ? actions.bulkApprove : actions.bulkReject;
    mutation.mutate(selected, {
      onSuccess: () => {
        toast({ title: `Successfully ${action}d ${selected.length} listings` });
        setSelected([]);
      }
    });
  };

  const handleSingleAction = (e: React.MouseEvent, id: string, action: 'approve' | 'reject') => {
    e.stopPropagation();
    const mutation = action === 'approve' ? actions.approve : actions.reject;
    mutation.mutate(id, {
      onSuccess: () => toast({ title: `Listing ${action}d` })
    });
  };

  const handleRowClick = (id: string) => {
    setLocation(`/listings/${id}`);
  };

  const toggleSelectAll = (checked: boolean) => {
    if (checked && data?.data) {
      setSelected(data.data.map((l: any) => l.id));
    } else {
      setSelected([]);
    }
  };

  const toggleSelectOne = (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    if (selected.includes(id)) {
      setSelected(selected.filter(item => item !== id));
    } else {
      setSelected([...selected, id]);
    }
  };

  return (
    <PageLayout 
      title="Listings" 
      subtitle={`${data?.meta?.total?.toLocaleString() || 0} listings`}
      actions={
        <Button variant="outline" className="bg-background shadow-sm">
          <Download className="w-4 h-4 mr-2" /> Export CSV
        </Button>
      }
    >
      <Card className="p-4 border-border shadow-sm flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
        <div className="flex bg-muted/50 p-1 rounded-lg overflow-x-auto w-full sm:w-auto">
          {TABS.map(s => (
            <button
              key={s}
              onClick={() => { setStatus(s); setSelected([]); setPage(1); }}
              className={`px-4 py-1.5 rounded-md text-sm font-medium capitalize whitespace-nowrap transition-all duration-200 ${
                status === s 
                  ? "bg-background text-foreground shadow-sm" 
                  : "text-muted-foreground hover:text-foreground hover:bg-background/50"
              }`}
            >
              {s}
              {s === "pending" && data?.pending_count > 0 && (
                <span className="ml-2 bg-destructive text-destructive-foreground text-[10px] px-2 py-0.5 rounded-full">
                  {data.pending_count}
                </span>
              )}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-3 w-full sm:w-auto">
          <Select value={type} onValueChange={setType}>
            <SelectTrigger className="w-[130px] h-9 bg-background">
              <SelectValue placeholder="All types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              <SelectItem value="standard">Standard</SelectItem>
              <SelectItem value="auction">Auction</SelectItem>
            </SelectContent>
          </Select>
          <div className="relative w-full sm:w-64">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search listings..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="pl-9 h-9 bg-background"
            />
          </div>
        </div>
      </Card>

      <BulkActionsBar 
        count={selected.length}
        actions={[
          {
            label: "Approve All",
            icon: <CheckCircle className="w-4 h-4" />,
            variant: "default",
            className: "bg-emerald-600 hover:bg-emerald-700",
            onClick: () => handleBulkAction('approve')
          },
          {
            label: "Reject All",
            icon: <XCircle className="w-4 h-4" />,
            variant: "destructive",
            onClick: () => handleBulkAction('reject')
          },
          {
            label: "Feature",
            icon: <Star className="w-4 h-4" />,
            variant: "secondary",
            onClick: () => toast({title: "Featured selected listings"})
          }
        ]}
        onClear={() => setSelected([])}
      />

      <Card className="border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground font-medium border-b border-border">
              <tr>
                <th className="px-4 py-3 w-12 text-center">
                  <input 
                    type="checkbox"
                    className="rounded border-input text-primary focus:ring-primary w-4 h-4 align-middle"
                    checked={data?.data?.length > 0 && selected.length === data?.data?.length}
                    onChange={e => toggleSelectAll(e.target.checked)}
                  />
                </th>
                <th className="px-4 py-3">Listing</th>
                <th className="px-4 py-3">Seller</th>
                <th className="px-4 py-3">Category</th>
                <th className="px-4 py-3">Price</th>
                <th className="px-4 py-3">Type</th>
                <th className="px-4 py-3">Location</th>
                <th className="px-4 py-3">Date</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {isLoading ? (
                Array.from({length: 5}).map((_, i) => (
                  <tr key={i}>
                    <td className="px-4 py-4"><Skeleton className="w-4 h-4 rounded" /></td>
                    <td className="px-4 py-4 flex gap-3">
                      <Skeleton className="w-12 h-12 rounded-lg" />
                      <div className="space-y-2 py-1">
                        <Skeleton className="h-4 w-32" />
                        <Skeleton className="h-3 w-20" />
                      </div>
                    </td>
                    <td className="px-4 py-4"><Skeleton className="h-4 w-24" /></td>
                    <td className="px-4 py-4"><Skeleton className="h-5 w-20 rounded-full" /></td>
                    <td className="px-4 py-4"><Skeleton className="h-4 w-16" /></td>
                    <td className="px-4 py-4"><Skeleton className="h-5 w-16 rounded-full" /></td>
                    <td className="px-4 py-4"><Skeleton className="h-4 w-24" /></td>
                    <td className="px-4 py-4"><Skeleton className="h-4 w-24" /></td>
                    <td className="px-4 py-4"><Skeleton className="h-8 w-8 rounded-md ml-auto" /></td>
                  </tr>
                ))
              ) : data?.data?.length === 0 ? (
                <tr>
                  <td colSpan={9}>
                    <EmptyState 
                      icon={<Tag className="w-8 h-8" />}
                      title="No listings found"
                      description={`There are no ${status} listings matching your current filters.`}
                      actionLabel="Clear Filters"
                      onAction={() => {setSearch(""); setType("all");}}
                    />
                  </td>
                </tr>
              ) : data?.data?.map((listing: any) => (
                <tr 
                  key={listing.id} 
                  className="hover:bg-muted/30 transition-colors group cursor-pointer"
                  onClick={() => handleRowClick(listing.id)}
                >
                  <td className="px-4 py-4 text-center" onClick={e => e.stopPropagation()}>
                    <input 
                      type="checkbox"
                      className="rounded border-input text-primary focus:ring-primary w-4 h-4 align-middle"
                      checked={selected.includes(listing.id)}
                      onChange={e => toggleSelectOne(e as any, listing.id)}
                    />
                  </td>
                  <td className="px-4 py-4">
                    <div className="flex items-center gap-3">
                      <img src={listing.images[0]?.url} className="w-12 h-12 rounded-lg object-cover bg-muted border border-border" alt="" />
                      <div>
                        <p className="font-medium text-foreground line-clamp-1">{listing.title}</p>
                        <p className="text-xs text-muted-foreground font-mono mt-0.5">{listing.id}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-4 text-foreground">{listing.user.name}</td>
                  <td className="px-4 py-4"><Badge variant="outline" className="bg-muted/50 font-normal">{listing.category.name_en}</Badge></td>
                  <td className="px-4 py-4 font-semibold">{listing.currency} {listing.price.toLocaleString()}</td>
                  <td className="px-4 py-4">
                    <Badge variant={listing.type === "auction" ? "secondary" : "default"} className="capitalize font-medium">
                      {listing.type}
                    </Badge>
                  </td>
                  <td className="px-4 py-4 text-muted-foreground">{listing.city}, {listing.country}</td>
                  <td className="px-4 py-4 text-muted-foreground">{format(new Date(listing.created_at), "MMM d, yyyy")}</td>
                  <td className="px-4 py-4">
                    <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      {listing.status === "pending" && (
                        <>
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50" onClick={(e) => handleSingleAction(e, listing.id, 'approve')} title="Approve">
                            <Check className="w-4 h-4" />
                          </Button>
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10" onClick={(e) => handleSingleAction(e, listing.id, 'reject')} title="Reject">
                            <X className="w-4 h-4" />
                          </Button>
                        </>
                      )}
                      <Button size="icon" variant="ghost" className="h-8 w-8 text-amber-500 hover:text-amber-600 hover:bg-amber-50" onClick={(e) => {e.stopPropagation(); toast({title: "Listing featured"});}} title="Feature">
                        <Star className="w-4 h-4" />
                      </Button>
                      
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild onClick={e => e.stopPropagation()}>
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-muted-foreground">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={(e) => {e.stopPropagation(); handleRowClick(listing.id);}}>
                            <Eye className="w-4 h-4 mr-2" /> View Details
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={(e) => {e.stopPropagation();}}>
                            <ExternalLink className="w-4 h-4 mr-2" /> View on Site
                          </DropdownMenuItem>
                          <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={(e) => {e.stopPropagation();}}>
                            <Trash2 className="w-4 h-4 mr-2" /> Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        
        {data?.meta?.total > 0 && (
          <div className="p-4 border-t border-border flex items-center justify-between text-sm text-muted-foreground bg-background">
            <span>Showing {(page - 1) * 25 + 1} to {Math.min(page * 25, data.meta.total)} of {data.meta.total} entries</span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage(p => p - 1)} className="bg-background">Previous</Button>
              <Button variant="outline" size="sm" disabled={page >= (data?.meta?.last_page || 1)} onClick={() => setPage(p => p + 1)} className="bg-background">Next</Button>
            </div>
          </div>
        )}
      </Card>
    </PageLayout>
  );
}
