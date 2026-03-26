import { useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import {
  ShieldCheck, ShieldX, Clock, Upload, Loader2, CheckCircle, AlertCircle, Image as ImageIcon,
} from "lucide-react";

interface KYCStatusResponse {
  status: "not_submitted" | "pending" | "under_review" | "approved" | "rejected";
  message?: string;
  rejection_reason?: string;
  approved_at?: string;
  expires_at?: string;
  full_name?: string;
  country?: string;
}

interface KYCForm {
  full_name: string;
  id_number: string;
  country: string;
  nationality: string;
  date_of_birth: string;
  doc_type: string;
}

interface UploadedFile {
  file: File;
  publicUrl: string | null;
  uploading: boolean;
  error: string | null;
  preview: string;
}

const STATUS_CONFIG = {
  not_submitted: {
    icon: AlertCircle,
    label: "Identity Not Verified",
    desc: "Verify your identity to unlock the trusted seller badge and higher limits.",
    color: "text-gray-500",
    bg: "bg-gray-50 border-gray-200",
  },
  pending: {
    icon: Clock,
    label: "Verification Pending",
    desc: "Your documents have been submitted and are awaiting review.",
    color: "text-yellow-600",
    bg: "bg-yellow-50 border-yellow-200",
  },
  under_review: {
    icon: Clock,
    label: "Under Review",
    desc: "Our team is reviewing your documents. This usually takes 1–2 business days.",
    color: "text-blue-600",
    bg: "bg-blue-50 border-blue-200",
  },
  approved: {
    icon: ShieldCheck,
    label: "Identity Verified",
    desc: "Your identity has been verified. You now have a trusted seller badge.",
    color: "text-green-600",
    bg: "bg-green-50 border-green-200",
  },
  rejected: {
    icon: ShieldX,
    label: "Verification Failed",
    desc: "Your verification was rejected. Please resubmit with correct documents.",
    color: "text-red-600",
    bg: "bg-red-50 border-red-200",
  },
};

const DOCUMENT_TYPES = [
  { value: "emirates_id", label: "Emirates ID" },
  { value: "passport", label: "Passport" },
  { value: "national_id", label: "National ID" },
  { value: "driving_license", label: "Driving License" },
];

const COUNTRIES = [
  { code: "ARE", label: "United Arab Emirates" },
  { code: "SAU", label: "Saudi Arabia" },
  { code: "KWT", label: "Kuwait" },
  { code: "BHR", label: "Bahrain" },
  { code: "QAT", label: "Qatar" },
  { code: "OMN", label: "Oman" },
  { code: "EGY", label: "Egypt" },
  { code: "JOR", label: "Jordan" },
  { code: "LBN", label: "Lebanon" },
  { code: "USA", label: "United States" },
  { code: "GBR", label: "United Kingdom" },
  { code: "IND", label: "India" },
  { code: "PAK", label: "Pakistan" },
  { code: "BGD", label: "Bangladesh" },
];

const INITIAL_FORM: KYCForm = {
  full_name: "",
  id_number: "",
  country: "ARE",
  nationality: "ARE",
  date_of_birth: "",
  doc_type: "emirates_id",
};

const INITIAL_UPLOAD: UploadedFile = { file: null as unknown as File, publicUrl: null, uploading: false, error: null, preview: "" };

async function uploadKycFile(file: File, side: string): Promise<string> {
  const presignRes = await api.post("/media/upload-url", {
    filename: file.name,
    content_type: file.type,
    folder: `kyc/${side}`,
    size: file.size,
  });

  const { upload_url, public_url, _mock } = presignRes.data.data as {
    upload_url: string;
    public_url: string;
    _mock?: boolean;
  };

  if (!_mock) {
    const putRes = await fetch(upload_url, {
      method: "PUT",
      headers: { "Content-Type": file.type },
      body: file,
    });
    if (!putRes.ok) {
      throw new Error(`Upload failed: ${putRes.status} ${putRes.statusText}`);
    }
  }

  return public_url;
}

interface FilePickerProps {
  label: string;
  required?: boolean;
  side: string;
  value: UploadedFile;
  onChange: (f: UploadedFile) => void;
}

function FilePicker({ label, required, side, value, onChange }: FilePickerProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const preview = URL.createObjectURL(file);
    onChange({ file, publicUrl: null, uploading: true, error: null, preview });

    try {
      const publicUrl = await uploadKycFile(file, side);
      onChange({ file, publicUrl, uploading: false, error: null, preview });
    } catch {
      onChange({ file, publicUrl: null, uploading: false, error: "Upload failed. Please try again.", preview });
    }
  };

  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">
        {label} {required && <span className="text-red-400">*</span>}
      </label>
      <div
        onClick={() => inputRef.current?.click()}
        className={`relative cursor-pointer border-2 border-dashed rounded-xl flex flex-col items-center justify-center gap-1.5 py-4 transition-colors ${
          value.publicUrl
            ? "border-green-300 bg-green-50"
            : value.error
              ? "border-red-300 bg-red-50"
              : "border-gray-200 bg-gray-50 hover:border-[#0071CE] hover:bg-blue-50"
        }`}
      >
        {value.uploading ? (
          <Loader2 size={20} className="animate-spin text-[#0071CE]" />
        ) : value.publicUrl ? (
          <>
            {value.preview ? (
              <img src={value.preview} alt="preview" className="w-16 h-16 object-cover rounded-lg" />
            ) : (
              <CheckCircle size={20} className="text-green-500" />
            )}
            <span className="text-xs text-green-600 font-medium">Uploaded</span>
          </>
        ) : (
          <>
            <ImageIcon size={20} className="text-gray-400" />
            <span className="text-xs text-gray-500">Click to upload</span>
            <span className="text-[10px] text-gray-400">JPG, PNG, WebP up to 10MB</span>
          </>
        )}
        {value.error && (
          <span className="text-[10px] text-red-500 mt-1">{value.error}</span>
        )}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={handleFileChange}
      />
    </div>
  );
}

export function KYCSection() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<KYCForm>(INITIAL_FORM);
  const [docFront, setDocFront] = useState<UploadedFile>(INITIAL_UPLOAD);
  const [docBack, setDocBack] = useState<UploadedFile>(INITIAL_UPLOAD);
  const [selfie, setSelfie] = useState<UploadedFile>(INITIAL_UPLOAD);

  const { data: kycStatus, isLoading } = useQuery<KYCStatusResponse>({
    queryKey: ["kyc-status"],
    queryFn: () =>
      api.get("/kyc/status").then((r) => r.data?.data ?? r.data),
    staleTime: 30_000,
    retry: 1,
  });

  const { mutate: submit, isPending: submitting, error: submitError } = useMutation({
    mutationFn: () => {
      if (!docFront.publicUrl) throw new Error("Please upload the front of your document.");
      if (!selfie.publicUrl) throw new Error("Please upload your selfie.");

      const documents: { document_type: string; file_url: string; side: string; mime_type: string }[] = [
        {
          document_type: form.doc_type,
          file_url: docFront.publicUrl,
          side: "front",
          mime_type: docFront.file?.type ?? "image/jpeg",
        },
      ];

      if (docBack.publicUrl) {
        documents.push({
          document_type: form.doc_type,
          file_url: docBack.publicUrl,
          side: "back",
          mime_type: docBack.file?.type ?? "image/jpeg",
        });
      }

      documents.push({
        document_type: "selfie",
        file_url: selfie.publicUrl,
        side: "front",
        mime_type: selfie.file?.type ?? "image/jpeg",
      });

      return api.post("/kyc/submit", {
        full_name: form.full_name,
        id_number: form.id_number,
        country: form.country,
        nationality: form.nationality,
        date_of_birth: form.date_of_birth,
        documents,
      }).then((r) => r.data);
    },
    onSuccess: () => {
      setShowForm(false);
      setForm(INITIAL_FORM);
      setDocFront(INITIAL_UPLOAD);
      setDocBack(INITIAL_UPLOAD);
      setSelfie(INITIAL_UPLOAD);
      qc.invalidateQueries({ queryKey: ["kyc-status"] });
    },
  });

  const status = kycStatus?.status ?? "not_submitted";
  const config = STATUS_CONFIG[status];
  const Icon = config.icon;

  const canSubmit = status === "not_submitted" || status === "rejected";

  const anyUploading = docFront.uploading || docBack.uploading || selfie.uploading;
  const formIsValid =
    form.full_name.trim() !== "" &&
    form.id_number.trim() !== "" &&
    form.date_of_birth !== "" &&
    !!docFront.publicUrl &&
    !!selfie.publicUrl &&
    !anyUploading;

  const handleCancel = () => {
    setShowForm(false);
    setForm(INITIAL_FORM);
    setDocFront(INITIAL_UPLOAD);
    setDocBack(INITIAL_UPLOAD);
    setSelfie(INITIAL_UPLOAD);
  };

  if (isLoading) {
    return (
      <div className="bg-white rounded-2xl shadow-sm p-5 mb-6 animate-pulse">
        <div className="h-6 w-40 bg-gray-200 rounded mb-2" />
        <div className="h-4 w-64 bg-gray-100 rounded" />
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-sm border mb-6 overflow-hidden">
      <div className={`${config.bg} border-b px-5 py-4 flex items-center justify-between gap-3`}>
        <div className="flex items-center gap-3">
          <Icon size={22} className={config.color} />
          <div>
            <p className={`font-semibold text-sm ${config.color}`}>{config.label}</p>
            <p className="text-xs text-gray-500 mt-0.5">{config.desc}</p>
            {status === "rejected" && kycStatus?.rejection_reason && (
              <p className="text-xs text-red-500 mt-1">Reason: {kycStatus.rejection_reason}</p>
            )}
            {status === "approved" && kycStatus?.expires_at && (
              <p className="text-xs text-green-600 mt-1">
                Valid until {new Date(kycStatus.expires_at).toLocaleDateString("en-AE")}
              </p>
            )}
          </div>
        </div>
        {canSubmit && !showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="shrink-0 bg-[#0071CE] text-white text-xs font-semibold px-4 py-2 rounded-lg hover:bg-[#005BA1] transition-colors flex items-center gap-1.5"
          >
            <Upload size={13} /> Verify Now
          </button>
        )}
      </div>

      {showForm && canSubmit && (
        <div className="p-5 space-y-4">
          <h3 className="font-semibold text-gray-800 text-sm">Identity Verification</h3>
          <p className="text-xs text-gray-500">
            Upload your government-issued ID to get verified. Your information is encrypted and secure.
          </p>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Full Name <span className="text-red-400">*</span></label>
              <input
                value={form.full_name}
                onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                placeholder="As on your ID"
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-[#0071CE]"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">ID Number <span className="text-red-400">*</span></label>
              <input
                value={form.id_number}
                onChange={(e) => setForm({ ...form, id_number: e.target.value })}
                placeholder="e.g. 784-1990-XXXXXXX-X"
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-[#0071CE]"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Country</label>
              <select
                value={form.country}
                onChange={(e) => setForm({ ...form, country: e.target.value })}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-[#0071CE]"
              >
                {COUNTRIES.map((c) => (
                  <option key={c.code} value={c.code}>{c.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Nationality</label>
              <select
                value={form.nationality}
                onChange={(e) => setForm({ ...form, nationality: e.target.value })}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-[#0071CE]"
              >
                {COUNTRIES.map((c) => (
                  <option key={c.code} value={c.code}>{c.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Date of Birth <span className="text-red-400">*</span></label>
              <input
                type="date"
                value={form.date_of_birth}
                onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-[#0071CE]"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Document Type</label>
              <select
                value={form.doc_type}
                onChange={(e) => setForm({ ...form, doc_type: e.target.value })}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:border-[#0071CE]"
              >
                {DOCUMENT_TYPES.map((d) => (
                  <option key={d.value} value={d.value}>{d.label}</option>
                ))}
              </select>
            </div>
          </div>

          <div className="bg-gray-50 rounded-xl p-4 border border-dashed border-gray-200 space-y-3">
            <p className="text-xs font-medium text-gray-600 flex items-center gap-1.5">
              <Upload size={12} /> Document Images <span className="text-red-400">*</span>
            </p>
            <div className="grid grid-cols-3 gap-3">
              <FilePicker
                label="ID Front"
                required
                side="doc_front"
                value={docFront}
                onChange={setDocFront}
              />
              <FilePicker
                label="ID Back"
                side="doc_back"
                value={docBack}
                onChange={setDocBack}
              />
              <FilePicker
                label="Selfie"
                required
                side="selfie"
                value={selfie}
                onChange={setSelfie}
              />
            </div>
          </div>

          {submitError && (
            <p className="text-xs text-red-500 flex items-center gap-1">
              <AlertCircle size={12} />
              {(submitError as Error).message || "Failed to submit. Please check your details and try again."}
            </p>
          )}

          <div className="flex gap-3">
            <button
              onClick={handleCancel}
              className="flex-1 border border-gray-200 text-gray-600 text-sm font-semibold py-2.5 rounded-xl hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => submit()}
              disabled={!formIsValid || submitting}
              className="flex-1 bg-[#0071CE] text-white text-sm font-semibold py-2.5 rounded-xl hover:bg-[#005BA1] transition-colors disabled:opacity-40 flex items-center justify-center gap-2"
            >
              {submitting ? <Loader2 size={14} className="animate-spin" /> : <CheckCircle size={14} />}
              Submit for Review
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
