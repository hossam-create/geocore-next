import { Switch, Route, Router as WouterRouter } from "wouter";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import NotFound from "@/pages/not-found";
import { AdminLayout } from "@/components/layout";

// Pages
import Login from "./pages/login";
import Dashboard from "./pages/dashboard";
import Listings from "./pages/listings";
import Auctions from "./pages/auctions";
import Users from "./pages/users";
import Reports from "./pages/reports";
import Payments from "./pages/payments";
import Pricing from "./pages/pricing";
import Categories from "./pages/categories";
import Storefronts from "./pages/storefronts";
import Settings from "./pages/settings";
import ListingDetailPage from "./pages/listing-detail";
import KYC from "./pages/kyc";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
  },
});

function Router() {
  return (
    <Switch>
      <Route path="/login" component={Login} />
      <Route path="/">
        <AdminLayout><Dashboard /></AdminLayout>
      </Route>
      <Route path="/listings">
        <AdminLayout><Listings /></AdminLayout>
      </Route>
      <Route path="/listings/:id">
        <AdminLayout><ListingDetailPage /></AdminLayout>
      </Route>
      <Route path="/auctions">
        <AdminLayout><Auctions /></AdminLayout>
      </Route>
      <Route path="/users">
        <AdminLayout><Users /></AdminLayout>
      </Route>
      <Route path="/reports">
        <AdminLayout><Reports /></AdminLayout>
      </Route>
      <Route path="/payments">
        <AdminLayout><Payments /></AdminLayout>
      </Route>
      <Route path="/pricing">
        <AdminLayout><Pricing /></AdminLayout>
      </Route>
      <Route path="/categories">
        <AdminLayout><Categories /></AdminLayout>
      </Route>
      <Route path="/storefronts">
        <AdminLayout><Storefronts /></AdminLayout>
      </Route>
      <Route path="/settings">
        <AdminLayout><Settings /></AdminLayout>
      </Route>
      <Route path="/kyc">
        <AdminLayout><KYC /></AdminLayout>
      </Route>
      <Route component={NotFound} />
    </Switch>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <WouterRouter base={import.meta.env.BASE_URL.replace(/\/$/, "")}>
          <Router />
        </WouterRouter>
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
