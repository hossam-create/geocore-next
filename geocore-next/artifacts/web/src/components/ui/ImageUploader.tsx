import { useState, useRef, useCallback } from "react";
import api from "@/lib/api";
import { Upload, X, Image as ImageIcon, Loader2, AlertCircle, Plus } from "lucide-react";

interface UploadedImage {
  key: string;
  url: string;
  file_name: string;
}

interface ImageUploaderProps {
  images: UploadedImage[];
  onChange: (images: UploadedImage[]) => void;
  maxImages?: number;
  folder?: string;
}

const MAX_SIZE_MB = 10;
const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/webp", "image/gif"];

export function ImageUploader({
  images,
  onChange,
  maxImages = 8,
  folder = "listings",
}: ImageUploaderProps) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const uploadFile = async (file: File): Promise<UploadedImage | null> => {
    if (!ALLOWED_TYPES.includes(file.type)) {
      setError(`"${file.name}" is not a supported image type (JPEG, PNG, WebP, GIF)`);
      return null;
    }
    if (file.size > MAX_SIZE_MB * 1024 * 1024) {
      setError(`"${file.name}" is too large (max ${MAX_SIZE_MB}MB)`);
      return null;
    }

    try {
      // Step 1: Get presigned upload URL
      const { data: urlRes } = await api.post("/media/upload-url", {
        filename: file.name,
        content_type: file.type,
        folder,
        size: file.size,
      });

      if (!urlRes.success) {
        setError("Failed to get upload URL");
        return null;
      }

      const { upload_url, public_url, key, _mock } = urlRes.data;

      // Step 2: Upload directly to R2 (or skip if mock)
      if (!_mock) {
        const uploadRes = await fetch(upload_url, {
          method: "PUT",
          body: file,
          headers: {
            "Content-Type": file.type,
            "Cache-Control": "public, max-age=31536000, immutable",
          },
        });

        if (!uploadRes.ok) {
          setError("Upload failed. Please try again.");
          return null;
        }
      }

      return { key, url: public_url, file_name: file.name };
    } catch {
      setError("Upload failed. Please check your connection.");
      return null;
    }
  };

  const handleFiles = useCallback(
    async (files: File[]) => {
      setError(null);
      const remaining = maxImages - images.length;
      if (remaining <= 0) {
        setError(`Maximum ${maxImages} images allowed`);
        return;
      }

      const toUpload = files.slice(0, remaining);
      setUploading(true);

      const results = await Promise.all(toUpload.map(uploadFile));
      const successful = results.filter(Boolean) as UploadedImage[];

      if (successful.length > 0) {
        onChange([...images, ...successful]);
      }

      setUploading(false);
    },
    [images, maxImages, onChange]
  );

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    if (files.length > 0) handleFiles(files);
    e.target.value = "";
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const files = Array.from(e.dataTransfer.files).filter((f) =>
      ALLOWED_TYPES.includes(f.type)
    );
    if (files.length > 0) handleFiles(files);
  };

  const removeImage = async (idx: number) => {
    const img = images[idx];
    onChange(images.filter((_, i) => i !== idx));
    // Attempt cleanup (non-blocking)
    try {
      await api.delete("/media/delete", { data: { key: img.key } });
    } catch {}
  };

  const canAdd = images.length < maxImages && !uploading;

  return (
    <div className="space-y-3">
      {/* Upload area */}
      {canAdd && (
        <div
          onClick={() => inputRef.current?.click()}
          onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
          className={`border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-all ${
            dragOver
              ? "border-[#0071CE] bg-[#0071CE]/5"
              : "border-gray-200 hover:border-[#0071CE] hover:bg-gray-50"
          }`}
        >
          {uploading ? (
            <div className="flex flex-col items-center gap-2 text-gray-400">
              <Loader2 className="w-8 h-8 animate-spin text-[#0071CE]" />
              <p className="text-sm font-medium text-gray-600">Uploading...</p>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2 text-gray-400">
              <Upload className="w-8 h-8 text-[#0071CE]" />
              <p className="text-sm font-semibold text-gray-600">
                Drop images here or <span className="text-[#0071CE] underline">browse</span>
              </p>
              <p className="text-xs text-gray-400">
                JPEG, PNG, WebP, GIF · Max {MAX_SIZE_MB}MB · Up to {maxImages} photos
              </p>
            </div>
          )}
        </div>
      )}

      <input
        ref={inputRef}
        type="file"
        accept={ALLOWED_TYPES.join(",")}
        multiple
        className="hidden"
        onChange={handleInputChange}
      />

      {/* Error */}
      {error && (
        <div className="flex items-center gap-2 bg-red-50 border border-red-200 rounded-xl px-3 py-2.5">
          <AlertCircle className="w-4 h-4 text-red-500 shrink-0" />
          <p className="text-sm text-red-600">{error}</p>
          <button onClick={() => setError(null)} className="ml-auto text-red-400 hover:text-red-600">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Image grid */}
      {images.length > 0 && (
        <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
          {images.map((img, idx) => (
            <div key={img.key} className="relative group aspect-square rounded-xl overflow-hidden bg-gray-100">
              <img
                src={img.url}
                alt={img.file_name}
                className="w-full h-full object-cover"
              />
              {/* Main badge */}
              {idx === 0 && (
                <div className="absolute bottom-1 left-1 bg-[#0071CE] text-white text-[10px] font-bold px-1.5 py-0.5 rounded">
                  Main
                </div>
              )}
              {/* Delete button */}
              <button
                onClick={() => removeImage(idx)}
                className="absolute top-1 right-1 bg-gray-900/70 text-white rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
              >
                <X className="w-3 h-3" />
              </button>
            </div>
          ))}

          {/* Add more button */}
          {canAdd && (
            <button
              onClick={() => inputRef.current?.click()}
              className="aspect-square rounded-xl border-2 border-dashed border-gray-200 flex flex-col items-center justify-center gap-1 text-gray-400 hover:border-[#0071CE] hover:text-[#0071CE] transition-colors"
            >
              <Plus className="w-5 h-5" />
              <span className="text-[10px] font-medium">Add</span>
            </button>
          )}
        </div>
      )}

      {/* Counter */}
      {images.length > 0 && (
        <p className="text-xs text-gray-400 text-right">
          {images.length}/{maxImages} photos · Drag to reorder (first photo = main)
        </p>
      )}
    </div>
  );
}
