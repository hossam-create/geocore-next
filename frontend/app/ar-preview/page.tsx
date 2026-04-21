"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import dynamic from "next/dynamic";
import Link from "next/link";
import { ArrowLeft, Box, Smartphone } from "lucide-react";

const ModelViewer = dynamic(() => import("@/components/ar/ModelViewer"), { ssr: false });

function ARPreviewContent() {
  const searchParams = useSearchParams();
  const src = searchParams.get("src") || "";
  const poster = searchParams.get("poster") || "";
  const title = searchParams.get("title") || "3D Preview";

  if (!src) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-16 text-center">
        <Box className="w-16 h-16 mx-auto mb-4 text-gray-300" />
        <h1 className="text-2xl font-bold mb-2">AR Preview</h1>
        <p className="text-gray-500 mb-6">No 3D model URL provided. Use the <code className="bg-gray-100 px-2 py-0.5 rounded">?src=</code> parameter.</p>
        <div className="bg-gray-50 rounded-xl p-6 text-left text-sm">
          <p className="font-medium mb-2">Example:</p>
          <code className="text-blue-600 break-all">/ar-preview?src=https://modelviewer.dev/shared-assets/models/Astronaut.glb&title=Astronaut</code>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <Link href="/" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back
      </Link>

      <div className="flex items-center gap-3 mb-6">
        <div className="bg-purple-100 p-2 rounded-lg"><Box className="w-6 h-6 text-purple-600" /></div>
        <div>
          <h1 className="text-2xl font-bold">{title}</h1>
          <p className="text-gray-500 text-sm flex items-center gap-1">
            <Smartphone className="w-3 h-3" /> Supports AR on compatible devices
          </p>
        </div>
      </div>

      <div className="bg-white rounded-2xl border overflow-hidden">
        <ModelViewer
          src={src}
          poster={poster || undefined}
          alt={title}
          className="w-full h-[500px]"
          autoRotate
          cameraControls
          ar
        />
      </div>

      <div className="mt-4 flex items-center gap-4 text-sm text-gray-500">
        <span className="flex items-center gap-1"><Box className="w-4 h-4" /> Drag to rotate</span>
        <span>Scroll to zoom</span>
        <span>Tap AR icon on mobile to place in your room</span>
      </div>
    </div>
  );
}

export default function ARPreviewPage() {
  return (
    <Suspense fallback={<div className="text-center py-16 text-gray-400">Loading AR Preview...</div>}>
      <ARPreviewContent />
    </Suspense>
  );
}
