import { useState } from "react";
import { useStorefronts, useStorefrontActions } from "@/hooks/use-storefronts";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Search, Store, Star, Ban, ExternalLink, CheckCircle } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

export default function StorefrontsPage() {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const { data, isLoading } = useStorefronts(search, page);
  const actions = useStorefrontActions();
  const { toast } = useToast();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold font-display tracking-tight text-foreground flex items-center gap-3">
          <Store className="w-8 h-8 text-primary" /> Storefronts
        </h1>
      </div>

      <Card className="p-4 border-none shadow-sm">
        <div className="relative w-full max-w-md">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input placeholder="Search stores..." value={search} onChange={e => setSearch(e.target.value)} className="pl-9 bg-muted/30 border-none" />
        </div>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {isLoading ? (
          <div>Loading...</div>
        ) : data?.data?.map((store: any) => (
          <Card key={store.id} className="p-0 border-none shadow-sm overflow-hidden flex flex-col group hover:shadow-md transition-all">
            <div className="h-24 bg-gradient-to-r from-primary/80 to-primary relative">
              {store.is_featured && (
                <Badge className="absolute top-3 right-3 bg-amber-500 text-white border-none shadow-sm"><Star className="w-3 h-3 mr-1 fill-current" /> Featured</Badge>
              )}
            </div>
            <div className="px-6 pb-6 pt-0 relative flex-1 flex flex-col">
              <div className="w-16 h-16 rounded-xl bg-card border-4 border-card shadow-sm absolute -top-8 flex items-center justify-center text-2xl overflow-hidden">
                <img src={`https://picsum.photos/seed/${store.id}/100`} className="w-full h-full object-cover" />
              </div>
              
              <div className="mt-10 flex-1">
                <h3 className="text-xl font-bold font-display text-foreground line-clamp-1">{store.name}</h3>
                <p className="text-sm text-primary font-mono bg-primary/10 w-fit px-2 py-0.5 rounded mt-1 mb-3">@{store.slug}</p>
                
                <div className="space-y-2 text-sm text-muted-foreground">
                  <p>Owner: <span className="text-foreground font-medium">{store.owner.name}</span></p>
                  <p>Location: <span className="text-foreground">{store.location}</span></p>
                  <div className="flex gap-4 mt-2">
                    <div><span className="font-bold text-foreground">{store.listings_count}</span> items</div>
                    <div><span className="font-bold text-foreground flex items-center gap-1"><Star className="w-3 h-3 fill-amber-400 text-amber-400" /> {store.rating}</span></div>
                  </div>
                </div>
              </div>

              <div className="mt-6 flex gap-2">
                <Button className="flex-1" variant={store.status === 'suspended' ? 'outline' : 'default'} onClick={() => {
                  actions.toggleStatus.mutate({id: store.id, suspend: store.status !== 'suspended'});
                  toast({title: `Store ${store.status === 'suspended' ? 'activated' : 'suspended'}`});
                }}>
                  {store.status === 'suspended' ? <><CheckCircle className="w-4 h-4 mr-2 text-emerald-500"/> Activate</> : <><Ban className="w-4 h-4 mr-2"/> Suspend</>}
                </Button>
                <Button variant="outline" size="icon" className="shrink-0" onClick={() => {
                  actions.toggleFeature.mutate(store.id);
                  toast({title: "Featured status toggled"});
                }}>
                  <Star className={`w-4 h-4 ${store.is_featured ? 'fill-amber-400 text-amber-400' : ''}`} />
                </Button>
                <Button variant="secondary" size="icon" className="shrink-0 bg-muted/50 hover:bg-muted">
                  <ExternalLink className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
