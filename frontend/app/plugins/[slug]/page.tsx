"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, Puzzle, Download, Star, CheckCircle } from "lucide-react";

interface Plugin {
  id: string;
  name: string;
  slug: string;
  description: string;
  category: string;
  version: string;
  icon_url: string;
  repo_url: string;
  price: number;
  currency: string;
  is_free: boolean;
  install_count: number;
  avg_rating: number;
  created_at: string;
}

export default function PluginDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const qc = useQueryClient();

  const { data: plugin, isLoading } = useQuery<Plugin>({
    queryKey: ["plugin", slug],
    queryFn: async () => {
      const { data } = await axios.get(`/api/v1/plugins/${slug}`);
      return data.data;
    },
  });

  const installMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/plugins/${slug}/install`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["plugin", slug] }),
  });

  const uninstallMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/plugins/${slug}/uninstall`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["plugin", slug] }),
  });

  if (isLoading) return <div className="text-center py-16 text-gray-400">Loading...</div>;
  if (!plugin) return <div className="text-center py-16 text-gray-400">Plugin not found</div>;

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <Link href="/plugins" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back to Marketplace
      </Link>

      <div className="bg-white rounded-xl border p-6">
        <div className="flex items-start gap-4 mb-6">
          {plugin.icon_url ? (
            <img src={plugin.icon_url} alt="" className="w-16 h-16 rounded-xl object-cover" />
          ) : (
            <div className="w-16 h-16 bg-purple-100 rounded-xl flex items-center justify-center">
              <Puzzle className="w-8 h-8 text-purple-500" />
            </div>
          )}
          <div className="flex-1">
            <h1 className="text-2xl font-bold">{plugin.name}</h1>
            <div className="flex items-center gap-3 mt-1 text-sm text-gray-500">
              <span>v{plugin.version}</span>
              <span className="capitalize bg-gray-100 px-2 py-0.5 rounded">{plugin.category}</span>
              <span className="flex items-center gap-1"><Download className="w-3 h-3" /> {plugin.install_count} installs</span>
              {plugin.avg_rating > 0 && <span className="flex items-center gap-1"><Star className="w-3 h-3 text-yellow-500" /> {plugin.avg_rating}</span>}
            </div>
          </div>
          <span className={`text-lg font-bold ${plugin.is_free ? "text-green-600" : "text-gray-800"}`}>
            {plugin.is_free ? "Free" : `${plugin.currency} ${plugin.price}`}
          </span>
        </div>

        {plugin.description && (
          <div className="mb-6">
            <h2 className="font-semibold mb-2">Description</h2>
            <p className="text-gray-600 text-sm whitespace-pre-line">{plugin.description}</p>
          </div>
        )}

        {plugin.repo_url && (
          <div className="mb-6">
            <h2 className="font-semibold mb-1">Repository</h2>
            <a href={plugin.repo_url} target="_blank" rel="noopener" className="text-blue-600 text-sm hover:underline break-all">{plugin.repo_url}</a>
          </div>
        )}

        <div className="flex gap-3">
          <button
            onClick={() => installMut.mutate()}
            disabled={installMut.isPending}
            className="flex items-center gap-2 bg-purple-600 text-white px-6 py-2.5 rounded-lg hover:bg-purple-700 disabled:opacity-50 transition"
          >
            <CheckCircle className="w-4 h-4" /> Install
          </button>
          <button
            onClick={() => uninstallMut.mutate()}
            disabled={uninstallMut.isPending}
            className="flex items-center gap-2 bg-gray-100 text-gray-700 px-6 py-2.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition"
          >
            Uninstall
          </button>
        </div>
      </div>
    </div>
  );
}
