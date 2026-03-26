import { useEffect } from "react";
import { Switch, Route, Router as WouterRouter } from "wouter";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@/store/auth";
import { Header } from "@/components/layout/Header";
import { Footer } from "@/components/layout/Footer";
import HomePage from "@/pages/HomePage";
import ListingsPage from "@/pages/ListingsPage";
import AuctionsPage from "@/pages/AuctionsPage";
import ListingDetailPage from "@/pages/ListingDetailPage";
import LoginPage from "@/pages/LoginPage";
import RegisterPage from "@/pages/RegisterPage";
import SellerPage from "@/pages/SellerPage";
import SellPage from "@/pages/SellPage";
import WalletPage from "@/pages/WalletPage";
import MyStorefrontPage from "@/pages/MyStorefrontPage";
import StoreListPage from "@/pages/StoreListPage";
import BrandOutletPage from "@/pages/BrandOutletPage";
import StorefrontPage from "@/pages/StorefrontPage";
import ProfilePage from "@/pages/ProfilePage";
import DashboardPage from "@/pages/DashboardPage";
import SearchPage from "@/pages/SearchPage";
import AdvancedSearchPage from "@/pages/AdvancedSearchPage";
import AuctionDetailPage from "@/pages/AuctionDetailPage";
import CheckoutPage from "@/pages/CheckoutPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

function NotFound() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center text-center">
      <div>
        <p className="text-6xl mb-4">404</p>
        <h1 className="text-2xl font-bold text-gray-800">Page Not Found</h1>
        <a href="/web/" className="mt-4 text-[#0071CE] hover:underline block">
          ← Back to Home
        </a>
      </div>
    </div>
  );
}

function AppRoutes() {
  const { restoreSession } = useAuthStore();

  useEffect(() => {
    restoreSession();
  }, []);

  return (
    <div className="min-h-screen flex flex-col">
      <Header />
      <main className="flex-1">
        <Switch>
          <Route path="/" component={HomePage} />
          <Route path="/listings" component={ListingsPage} />
          <Route path="/listings/:id" component={ListingDetailPage} />
          <Route path="/auctions" component={AuctionsPage} />
          <Route path="/auctions/:id" component={AuctionDetailPage} />
          <Route path="/checkout" component={CheckoutPage} />
          <Route path="/sell" component={SellPage} />
          <Route path="/login" component={LoginPage} />
          <Route path="/register" component={RegisterPage} />
          <Route path="/profile" component={ProfilePage} />
          <Route path="/wallet" component={WalletPage} />
          <Route path="/my-store" component={MyStorefrontPage} />
          <Route path="/stores" component={StoreListPage} />
          <Route path="/brand-outlet" component={BrandOutletPage} />
          <Route path="/stores/:slug" component={StorefrontPage} />
          <Route path="/sellers/:id" component={SellerPage} />
          <Route path="/dashboard" component={DashboardPage} />
          <Route path="/search" component={SearchPage} />
          <Route path="/advanced-search" component={AdvancedSearchPage} />
          <Route component={NotFound} />
        </Switch>
      </main>
      <Footer />
    </div>
  );
}

function App() {
  const base = import.meta.env.BASE_URL.replace(/\/$/, "");
  return (
    <QueryClientProvider client={queryClient}>
      <WouterRouter base={base}>
        <AppRoutes />
      </WouterRouter>
    </QueryClientProvider>
  );
}

export default App;
