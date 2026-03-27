import { useState } from "react";
import { useAuctions, useAuctionActions } from "@/hooks/use-auctions";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Hammer, Trash2, Eye, StopCircle } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useToast } from "@/hooks/use-toast";

const TABS = ["all", "live", "upcoming", "ended"];

export default function AuctionsPage() {
  const [status, setStatus] = useState("live");
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAuctions(status, page);
  const actions = useAuctionActions();
  const { toast } = useToast();

  const handleEnd = (id: string) => {
    actions.endNow.mutate(id, {
      onSuccess: () => toast({ title: "Auction ended successfully" })
    });
  };

  const handleDelete = (id: string) => {
    if(confirm("Are you sure?")) {
      actions.deleteAuction.mutate(id, {
        onSuccess: () => toast({ title: "Auction deleted" })
      });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold font-display tracking-tight flex items-center gap-3">
            <Hammer className="w-8 h-8 text-primary" /> Auctions Monitor
          </h1>
          <p className="text-muted-foreground mt-1">Real-time view of marketplace auctions</p>
        </div>
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

      <Card className="border-none shadow-sm overflow-hidden">
        <table className="w-full text-sm text-left">
          <thead className="bg-muted/50 text-muted-foreground uppercase text-xs font-semibold">
            <tr>
              <th className="p-4">Title & ID</th>
              <th className="p-4">Type</th>
              <th className="p-4">Seller</th>
              <th className="p-4">Start Price</th>
              <th className="p-4">Current Bid</th>
              <th className="p-4">Bids</th>
              <th className="p-4">Status</th>
              <th className="p-4">Time Left</th>
              <th className="p-4 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading ? (
               <tr><td colSpan={9} className="p-8 text-center text-muted-foreground">Loading auctions...</td></tr>
            ) : data?.data?.map((auction: any) => (
              <tr key={auction.id} className="hover:bg-muted/20">
                <td className="p-4">
                  <p className="font-semibold text-foreground">{auction.title}</p>
                  <p className="text-xs text-muted-foreground font-mono">{auction.id}</p>
                </td>
                <td className="p-4">
                  <Badge variant="outline" className="capitalize bg-background">{auction.type}</Badge>
                </td>
                <td className="p-4 font-medium">{auction.seller.name}</td>
                <td className="p-4 text-muted-foreground">{auction.currency} {auction.start_price.toLocaleString()}</td>
                <td className="p-4 font-bold text-primary">{auction.currency} {auction.current_bid.toLocaleString()}</td>
                <td className="p-4">
                  <Badge variant="secondary" className="font-mono">{auction.bids_count}</Badge>
                </td>
                <td className="p-4">
                  {auction.status === 'live' && <Badge className="bg-emerald-500 hover:bg-emerald-600 animate-pulse">Live</Badge>}
                  {auction.status === 'upcoming' && <Badge className="bg-blue-500 hover:bg-blue-600">Upcoming</Badge>}
                  {auction.status === 'ended' && <Badge variant="secondary">Ended</Badge>}
                </td>
                <td className="p-4 text-muted-foreground">
                  {new Date(auction.ends_at) > new Date() 
                    ? formatDistanceToNow(new Date(auction.ends_at), { addSuffix: true }) 
                    : "Ended"}
                </td>
                <td className="p-4">
                  <div className="flex items-center justify-end gap-2">
                    <Button size="icon" variant="ghost" className="h-8 w-8 text-muted-foreground hover:text-primary">
                      <Eye className="w-4 h-4" />
                    </Button>
                    {auction.status === 'live' && (
                      <Button size="icon" variant="ghost" className="h-8 w-8 text-amber-500 hover:text-amber-600 hover:bg-amber-50" onClick={() => handleEnd(auction.id)}>
                        <StopCircle className="w-4 h-4" />
                      </Button>
                    )}
                    <Button size="icon" variant="ghost" className="h-8 w-8 text-destructive hover:bg-destructive/10" onClick={() => handleDelete(auction.id)}>
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
