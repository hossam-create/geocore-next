import { useRoute } from "wouter";
import { PageLayout } from "@/components/layout";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { ExternalLink, Star, CheckCircle, XCircle, MapPin, Calendar, Eye, Shield, Tag, MessageSquare } from "lucide-react";
import { Switch } from "@/components/ui/switch";
import { format } from "date-fns";
import { useToast } from "@/hooks/use-toast";
import { useQuery } from "@tanstack/react-query";

// Mock fetching single listing based on the list items
const fetchListing = async (id: string) => {
  // Simulate network
  await new Promise(r => setTimeout(r, 800));
  
  return {
    id: id,
    title: "Luxury Villa in Palm Jumeirah - Special Edition",
    description: "Experience unparalleled luxury in this exquisite 5-bedroom villa located in the prestigious Palm Jumeirah. Featuring breathtaking sea views, a private beach, infinity pool, and state-of-the-art smart home automation.\n\nThe property boasts 7,500 sq ft of built-up area on a 10,000 sq ft plot. Master suite includes a private terrace, his and hers walk-in closets, and a spa-like en-suite bathroom.\n\nReady to move in. Viewing highly recommended.",
    price: 15500000,
    currency: "AED",
    type: "standard",
    condition: "Brand New",
    status: "active",
    featured: true,
    views: 1245,
    favorites: 84,
    created_at: new Date(Date.now() - 14 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 2 * 86400000).toISOString(),
    expires_at: new Date(Date.now() + 16 * 86400000).toISOString(),
    images: [
      { url: "https://picsum.photos/seed/villa1/800/600", is_cover: true },
      { url: "https://picsum.photos/seed/villa2/800/600", is_cover: false },
      { url: "https://picsum.photos/seed/villa3/800/600", is_cover: false },
      { url: "https://picsum.photos/seed/villa4/800/600", is_cover: false },
    ],
    user: {
      id: "usr_983",
      name: "Ahmed Ali Real Estate",
      email: "ahmed@luxuryestates.ae",
      rating: 4.8,
      joined: new Date(Date.now() - 365 * 86400000).toISOString(),
      avatar: "https://i.pravatar.cc/150?u=ahmed"
    },
    category: {
      id: "cat_re",
      name_en: "Real Estate > Villas"
    },
    location: {
      country: "United Arab Emirates",
      city: "Dubai",
      address: "Frond P, Palm Jumeirah",
      lat: 25.1124,
      lng: 55.1390
    },
    attributes: [
      { name: "Bedrooms", value: "5" },
      { name: "Bathrooms", value: "6" },
      { name: "Size", value: "7,500 sqft" },
      { name: "Furnished", value: "Yes" },
      { name: "Parking", value: "3 Cars" }
    ]
  };
};

export default function ListingDetailPage() {
  const [, params] = useRoute("/listings/:id");
  const { toast } = useToast();
  
  const id = params?.id || "unknown";

  const { data: listing, isLoading } = useQuery({
    queryKey: ["listing", id],
    queryFn: () => fetchListing(id)
  });

  const handleStatusChange = (newStatus: string) => {
    toast({ title: `Listing status updated to ${newStatus}` });
  };

  const breadcrumbs = [
    { label: "Listings", path: "/listings" },
    { label: listing?.title || "Loading..." }
  ];

  if (isLoading) {
    return (
      <PageLayout title="Loading..." breadcrumbs={breadcrumbs}>
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      </PageLayout>
    );
  }

  if (!listing) {
    return (
      <PageLayout title="Not Found" breadcrumbs={breadcrumbs}>
        <div className="p-12 text-center text-muted-foreground">Listing not found</div>
      </PageLayout>
    );
  }

  return (
    <PageLayout 
      title={listing.title}
      breadcrumbs={breadcrumbs}
      actions={
        <>
          <Button variant="outline" className="bg-background">
            <ExternalLink className="w-4 h-4 mr-2" /> View on Site
          </Button>
          {listing.status === "pending" && (
            <>
              <Button onClick={() => handleStatusChange("active")} className="bg-emerald-600 hover:bg-emerald-700">
                <CheckCircle className="w-4 h-4 mr-2" /> Approve
              </Button>
              <Button variant="destructive" onClick={() => handleStatusChange("rejected")}>
                <XCircle className="w-4 h-4 mr-2" /> Reject
              </Button>
            </>
          )}
        </>
      }
    >
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        
        {/* Left Column - Main Details */}
        <div className="xl:col-span-2 space-y-6">
          
          {/* Images Grid */}
          <Card className="p-6 border-border shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">Media</h3>
              <Badge variant="secondary">{listing.images.length} Images</Badge>
            </div>
            <div className="grid grid-cols-4 gap-4">
              {listing.images.map((img, i) => (
                <div key={i} className={`relative rounded-xl overflow-hidden border border-border group ${i === 0 ? "col-span-4 sm:col-span-2 row-span-2 h-64" : "h-32"}`}>
                  <img src={img.url} alt="" className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105" />
                  {img.is_cover && (
                    <Badge className="absolute top-2 left-2 bg-background/80 backdrop-blur-md text-foreground hover:bg-background/90">
                      Cover Image
                    </Badge>
                  )}
                </div>
              ))}
            </div>
          </Card>

          {/* Listing Information */}
          <Card className="p-6 border-border shadow-sm">
            <h3 className="text-lg font-semibold mb-6">Listing Information</h3>
            
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
              <div className="p-4 bg-muted/30 rounded-xl border border-border/50">
                <p className="text-sm text-muted-foreground mb-1">Price</p>
                <p className="text-2xl font-bold font-display">{listing.currency} {listing.price.toLocaleString()}</p>
              </div>
              <div className="p-4 bg-muted/30 rounded-xl border border-border/50">
                <p className="text-sm text-muted-foreground mb-1">Type</p>
                <p className="text-lg font-medium capitalize">{listing.type}</p>
              </div>
              <div className="p-4 bg-muted/30 rounded-xl border border-border/50">
                <p className="text-sm text-muted-foreground mb-1">Condition</p>
                <p className="text-lg font-medium">{listing.condition}</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-2">Description</h4>
                <div className="text-foreground whitespace-pre-wrap leading-relaxed">
                  {listing.description}
                </div>
              </div>
            </div>
          </Card>

          {/* Attributes */}
          {listing.attributes && listing.attributes.length > 0 && (
            <Card className="p-6 border-border shadow-sm">
              <h3 className="text-lg font-semibold mb-4">Attributes</h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-y-4 gap-x-6">
                {listing.attributes.map((attr, i) => (
                  <div key={i} className="flex flex-col border-b border-border/50 pb-3">
                    <span className="text-sm text-muted-foreground mb-1">{attr.name}</span>
                    <span className="font-medium text-foreground">{attr.value}</span>
                  </div>
                ))}
              </div>
            </Card>
          )}

        </div>

        {/* Right Column - Meta & Status */}
        <div className="space-y-6">
          
          {/* Status & Visibility */}
          <Card className="p-6 border-border shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Status & Visibility</h3>
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Current Status</span>
                <StatusBadge status={listing.status} size="lg" />
              </div>
              
              <div className="flex items-center justify-between pt-4 border-t border-border">
                <div>
                  <p className="font-medium">Featured Listing</p>
                  <p className="text-sm text-muted-foreground">Show in featured sections</p>
                </div>
                <Switch checked={listing.featured} onCheckedChange={(v) => toast({title: `Featured status updated to ${v}`})} />
              </div>
            </div>
          </Card>

          {/* Seller Info */}
          <Card className="p-6 border-border shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Seller Information</h3>
            <div className="flex items-center gap-4 mb-6">
              <img src={listing.user.avatar} alt={listing.user.name} className="w-14 h-14 rounded-full bg-muted border border-border" />
              <div className="flex-1 min-w-0">
                <p className="font-semibold text-foreground truncate">{listing.user.name}</p>
                <div className="flex items-center gap-2 mt-1">
                  <Star className="w-4 h-4 fill-amber-400 text-amber-400" />
                  <span className="text-sm font-medium">{listing.user.rating}</span>
                  <span className="text-sm text-muted-foreground">• Member since {new Date(listing.user.joined).getFullYear()}</span>
                </div>
              </div>
            </div>
            
            <div className="space-y-3">
              <Button className="w-full" variant="outline">
                <Shield className="w-4 h-4 mr-2" /> View Full Profile
              </Button>
              <Button className="w-full" variant="outline">
                <MessageSquare className="w-4 h-4 mr-2" /> Message Seller
              </Button>
            </div>
          </Card>

          {/* Location */}
          <Card className="p-6 border-border shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Location</h3>
            <div className="flex items-start gap-3 text-muted-foreground">
              <MapPin className="w-5 h-5 shrink-0 mt-0.5 text-primary" />
              <div>
                <p className="font-medium text-foreground">{listing.location.city}, {listing.location.country}</p>
                <p className="text-sm">{listing.location.address}</p>
              </div>
            </div>
          </Card>

          {/* Performance & Timeline */}
          <Card className="p-6 border-border shadow-sm">
            <h3 className="text-lg font-semibold mb-4">Performance & Timeline</h3>
            
            <div className="grid grid-cols-2 gap-4 mb-6">
              <div className="flex flex-col gap-1 p-3 bg-muted/30 rounded-lg">
                <span className="text-muted-foreground text-sm flex items-center gap-1"><Eye className="w-3 h-3" /> Views</span>
                <span className="font-semibold text-lg">{listing.views.toLocaleString()}</span>
              </div>
              <div className="flex flex-col gap-1 p-3 bg-muted/30 rounded-lg">
                <span className="text-muted-foreground text-sm flex items-center gap-1"><Star className="w-3 h-3" /> Favorites</span>
                <span className="font-semibold text-lg">{listing.favorites.toLocaleString()}</span>
              </div>
            </div>

            <div className="space-y-4 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground flex items-center gap-2"><Calendar className="w-4 h-4" /> Created</span>
                <span className="font-medium">{format(new Date(listing.created_at), "MMM d, yyyy HH:mm")}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground flex items-center gap-2"><Calendar className="w-4 h-4" /> Updated</span>
                <span className="font-medium">{format(new Date(listing.updated_at), "MMM d, yyyy HH:mm")}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground flex items-center gap-2"><Tag className="w-4 h-4" /> Category</span>
                <span className="font-medium truncate max-w-[150px]" title={listing.category.name_en}>{listing.category.name_en}</span>
              </div>
            </div>
          </Card>

        </div>
      </div>
    </PageLayout>
  );
}
