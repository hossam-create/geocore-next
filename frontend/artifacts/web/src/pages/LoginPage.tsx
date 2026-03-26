import { useState } from "react";
import { Link, useLocation, useSearch } from "wouter";
import { useAuthStore } from "@/store/auth";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const { login } = useAuthStore();
  const [, navigate] = useLocation();
  const search = useSearch();
  const params = new URLSearchParams(search);
  const next = params.get("next") || "/";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(email, password);
      navigate(next);
    } catch (err: any) {
      setError(err?.response?.data?.message || "Invalid credentials. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 bg-gray-50">
      <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-extrabold text-[#0071CE]">
            Geo<span className="text-[#FFC220]">Core</span>
          </h1>
          <p className="text-gray-500 mt-2 text-sm">Sign in to buy, sell, and bid</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-3 text-sm">
              {error}
            </div>
          )}
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Email Address</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="you@example.com"
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder="••••••••"
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60 mt-2"
          >
            {loading ? "Signing In..." : "Sign In"}
          </button>
        </form>

        <div className="text-center mt-6 space-y-2">
          <p className="text-sm text-gray-500">
            Don't have an account?{" "}
            <Link href="/register" className="text-[#0071CE] font-semibold hover:underline">
              Create one free
            </Link>
          </p>
          <a href="#" className="text-xs text-gray-400 hover:text-gray-600 block">
            Forgot your password?
          </a>
        </div>
      </div>
    </div>
  );
}
