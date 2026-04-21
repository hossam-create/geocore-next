"use client";

import { useEffect, useRef } from "react";

interface ModelViewerProps {
  src: string;
  poster?: string;
  alt?: string;
  className?: string;
  autoRotate?: boolean;
  cameraControls?: boolean;
  ar?: boolean;
}

export default function ModelViewer({
  src,
  poster,
  alt = "3D model",
  className = "w-full h-[400px]",
  autoRotate = true,
  cameraControls = true,
  ar = true,
}: ModelViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Dynamically load model-viewer script
    if (typeof window !== "undefined" && !customElements.get("model-viewer")) {
      const script = document.createElement("script");
      script.type = "module";
      script.src = "https://ajax.googleapis.com/ajax/libs/model-viewer/3.4.0/model-viewer.min.js";
      document.head.appendChild(script);
    }
  }, []);

  // Build attributes string
  const attrs: Record<string, string> = { src, alt };
  if (poster) attrs.poster = poster;
  if (autoRotate) attrs["auto-rotate"] = "";
  if (cameraControls) attrs["camera-controls"] = "";
  if (ar) attrs.ar = "";

  const attrStr = Object.entries(attrs)
    .map(([k, v]) => (v === "" ? k : `${k}="${v}"`))
    .join(" ");

  return (
    <div
      ref={containerRef}
      className={className}
      dangerouslySetInnerHTML={{
        __html: `<model-viewer ${attrStr} style="width:100%;height:100%;" shadow-intensity="1" environment-image="neutral"></model-viewer>`,
      }}
    />
  );
}
