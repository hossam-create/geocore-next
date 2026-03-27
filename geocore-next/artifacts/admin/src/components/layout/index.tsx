import { ReactNode } from "react";
import { Link, useLocation } from "wouter";
import { useAuth, useLogout } from "@/hooks/use-auth";
import { 
  LayoutDashboard, Tag, Hammer, Store, FolderOpen, 
  Users, ShieldAlert, CreditCard, DollarSign, 
  Settings, Bell, Globe, Search, ChevronRight
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useListings } from "@/hooks/use-listings";
import { useReports } from "@/hooks/use-reports";

const navigation = [
  {
    section: null,
    items: [
      { icon: LayoutDashboard, label: "Dashboard", path: "/" },
    ]
  },
  {
    section: "CATALOG",
    items: [
      { icon: Tag, label: "Listings", path: "/listings", badge: "listings" },
      { icon: Hammer, label: "Auctions", path: "/auctions", badge: "auctions" },
      { icon: FolderOpen, label: "Categories", path: "/categories" },
      { icon: Store, label: "Storefronts", path: "/storefronts" },
    ]
  },
  {
    section: "USERS",
    items: [
      { icon: Users, label: "Customers", path: "/users" },
      { icon: ShieldAlert, label: "Reports", path: "/reports", badge: "reports" },
    ]
  },
  {
    section: "SALES",
    items: [
      { icon: CreditCard, label: "Payments", path: "/payments" },
      { icon: DollarSign, label: "Price Plans", path: "/pricing" },
    ]
  },
  {
    section: "CONFIGURATION",
    items: [
      { icon: Settings, label: "Settings", path: "/settings" },
    ]
  }
];

function Sidebar() {
  const [location] = useLocation();
  const { data: user } = useAuth();
  const logout = useLogout();
  
  const { data: listingsData } = useListings("pending", "", 1);
  const { data: reportsData } = useReports("pending");

  return (
    <aside className="w-64 fixed inset-y-0 left-0 bg-sidebar flex flex-col z-20">
      <div className="p-6">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 text-sidebar-primary-foreground flex items-center justify-center">
            <Globe className="w-8 h-8" />
          </div>
          <div>
            <p className="text-sidebar-primary-foreground font-bold text-lg tracking-tight font-display leading-tight">GeoCore</p>
            <p className="text-sidebar-foreground text-[10px] uppercase tracking-wider font-semibold">Admin Panel</p>
          </div>
        </div>
      </div>

      <nav className="flex-1 px-4 pb-4 space-y-6 overflow-y-auto">
        {navigation.map((section, idx) => (
          <div key={idx}>
            {section.section && (
              <p className="text-sidebar-foreground/70 text-xs font-semibold uppercase tracking-wider mb-3 px-3">
                {section.section}
              </p>
            )}
            <div className="space-y-1">
              {section.items.map((item) => {
                const isActive = location === item.path || (item.path !== "/" && location.startsWith(item.path));
                let badgeCount = 0;
                if (item.badge === "listings") badgeCount = listingsData?.pending_count || 0;
                if (item.badge === "reports") badgeCount = reportsData?.pending_count || 0;
                // mock live auctions count
                if (item.badge === "auctions") badgeCount = 12;

                return (
                  <Link
                    key={item.path}
                    href={item.path}
                    className={`flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors ${
                      isActive
                        ? "bg-sidebar-primary text-sidebar-primary-foreground"
                        : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                    }`}
                  >
                    <item.icon className="w-4 h-4 shrink-0" />
                    <span className="flex-1">{item.label}</span>
                    {badgeCount > 0 && (
                      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                        isActive ? "bg-sidebar-primary-foreground text-sidebar-primary" : "bg-destructive text-destructive-foreground"
                      }`}>
                        {badgeCount}
                      </span>
                    )}
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="p-4 border-t border-sidebar-border">
        <div className="flex items-center gap-3 px-2 py-2">
          <div className="w-9 h-9 rounded-full bg-sidebar-accent flex items-center justify-center shrink-0 font-bold text-sidebar-accent-foreground">
            {user?.name?.charAt(0) || "A"}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sidebar-primary-foreground text-sm font-medium truncate">{user?.name || "Admin User"}</p>
            <p className="text-sidebar-foreground text-xs truncate">Administrator</p>
          </div>
          <button 
            onClick={() => logout()} 
            className="text-sidebar-foreground hover:text-sidebar-primary-foreground transition-colors shrink-0"
            title="Log out"
          >
            <Settings className="w-5 h-5" />
          </button>
        </div>
      </div>
    </aside>
  );
}

export function PageLayout({ 
  title, 
  subtitle, 
  breadcrumbs, 
  actions, 
  children 
}: { 
  title: string;
  subtitle?: string;
  breadcrumbs?: {label: string, path?: string}[];
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex-1 flex flex-col min-h-0 bg-background">
      <header className="h-16 bg-card border-b border-border flex items-center justify-between px-6 shrink-0 z-10">
        <div className="flex items-center gap-4">
          {breadcrumbs && breadcrumbs.length > 0 ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              {breadcrumbs.map((crumb, i) => (
                <div key={i} className="flex items-center gap-2">
                  {i > 0 && <ChevronRight className="w-4 h-4" />}
                  {crumb.path ? (
                    <Link href={crumb.path} className="hover:text-foreground transition-colors">{crumb.label}</Link>
                  ) : (
                    <span className="text-foreground font-medium">{crumb.label}</span>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <h1 className="text-lg font-semibold text-foreground font-display">{title}</h1>
          )}
        </div>
        
        <div className="flex items-center gap-4">
          <div className="relative w-64 hidden md:block">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input 
              placeholder="Search..." 
              className="pl-9 h-9 bg-muted/50 border-none"
            />
          </div>
          <Button variant="ghost" size="icon" className="rounded-full text-muted-foreground hover:text-foreground">
            <Bell className="w-5 h-5" />
          </Button>
          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold text-sm">
            A
          </div>
        </div>
      </header>

      <main className="flex-1 p-6 overflow-y-auto">
        <div className="max-w-7xl mx-auto space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-foreground">{title}</h1>
              {subtitle && <p className="text-muted-foreground text-sm mt-1">{subtitle}</p>}
            </div>
            {actions && <div className="flex items-center gap-3">{actions}</div>}
          </div>
          {children}
        </div>
      </main>
    </div>
  );
}

export function AdminLayout({ children }: { children: ReactNode }) {
  const { data: user, isLoading } = useAuth();
  const [location] = useLocation();

  if (isLoading) return <div className="min-h-screen bg-background flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div></div>;
  
  if (!user && location !== "/login") {
    window.location.href = "/admin/login";
    return null;
  }

  if (location === "/login") return <>{children}</>;

  return (
    <div className="h-screen flex overflow-hidden">
      <Sidebar />
      <div className="ml-64 flex-1 flex flex-col min-w-0 h-full">
        {children}
      </div>
    </div>
  );
}
