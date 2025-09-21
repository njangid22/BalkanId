"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { DragEvent } from "react";
import { useMutation } from "urql";
import {
  Archive,
  ArrowRight,
  CloudUpload,
  HardDrive,
  ShieldCheck,
  Sparkles
} from "lucide-react";

import { cn } from "@/lib/utils";
import { useFilesDashboard } from "@/hooks/use-files-dashboard";

function formatBytes(bytes?: number | null) {
  if (!bytes || Number.isNaN(bytes)) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"] as const;
  let value = bytes;
  let i = 0;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i += 1;
  }
  return `${value.toFixed(value < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
}

function formatDate(value?: string | null) {
  if (!value) return "";
  return new Date(value).toLocaleString();
}

const CREATE_SHARE_MUTATION = /* GraphQL */ `
  mutation CreateShare($fileId: ID!, $visibility: ShareVisibility!, $expiresAt: Time) {
    createShare(input: { fileId: $fileId, visibility: $visibility, expiresAt: $expiresAt }) {
      id
      token
      visibility
      expiresAt
    }
  }
`;

const REVOKE_SHARE_MUTATION = /* GraphQL */ `
  mutation RevokeShare($id: ID!) {
    revokeShare(id: $id) {
      ok
    }
  }
`;
const DELETE_FILE_MUTATION = /* GraphQL */ `
  mutation DeleteFile($id: ID!) {
    deleteFile(id: $id) {
      ok
    }
  }
`;

type CreateShareResult = {
  createShare: {
    id: string;
    token?: string | null;
    visibility: string;
    expiresAt?: string | null;
  };
};

type RevokeShareResult = {
  revokeShare: {
    ok: boolean;
  };
};

type DeleteFileResult = {
  deleteFile: {
    ok: boolean;
  };
};

type ShareDetails = {
  visibility: string;
  token?: string | null;
  expiresAt?: string | null;
};

interface FilterDraft {
  search: string;
  mimeTypes: string[];
  minSizeMB?: string;
  maxSizeMB?: string;
  uploadedFrom?: string;
  uploadedTo?: string;
}

const MIME_TYPE_OPTIONS: { label: string; value: string }[] = [
  { label: "Images", value: "image/" },
  { label: "PDF", value: "application/pdf" },
  { label: "Documents", value: "application/msword" },
  { label: "Spreadsheets", value: "application/vnd.ms-excel" },
  { label: "Audio", value: "audio/" },
  { label: "Video", value: "video/" }
];

const DEFAULT_FILTER: FilterDraft = {
  search: "",
  mimeTypes: [],
  minSizeMB: "",
  maxSizeMB: "",
  uploadedFrom: "",
  uploadedTo: ""
};

export default function FilesPage() {
  const router = useRouter();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [pendingDownloadId, setPendingDownloadId] = useState<string | null>(null);
  const [shareMessage, setShareMessage] = useState<string | null>(null);
  const [shareError, setShareError] = useState<string | null>(null);
  const [shareState, setShareState] = useState<Record<string, ShareDetails | null>>({});
  const [shareLoadingIds, setShareLoadingIds] = useState<Record<string, boolean>>({});
  const [deleteMessage, setDeleteMessage] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteLoadingIds, setDeleteLoadingIds] = useState<Record<string, boolean>>({});
  const [isDragging, setIsDragging] = useState(false);
  const dragCounterRef = useRef(0);
  const [filterDraft, setFilterDraft] = useState<FilterDraft>({ ...DEFAULT_FILTER });
  const [activeFilter, setActiveFilter] = useState<FilterDraft>({ ...DEFAULT_FILTER });

  const graphFilter = useMemo<Record<string, unknown> | null>(() => {
    const payload: Record<string, unknown> = {};

    const trimmedSearch = activeFilter.search.trim();
    if (trimmedSearch) {
      payload.search = trimmedSearch;
    }

    if (activeFilter.mimeTypes.length > 0) {
      payload.mimeTypes = [...activeFilter.mimeTypes];
    }

    const parseMegabytes = (value?: string) => {
      if (!value) {
        return undefined;
      }
      const trimmed = value.trim();
      if (!trimmed) {
        return undefined;
      }
      const parsed = Number(trimmed);
      if (Number.isNaN(parsed) || parsed < 0) {
        return undefined;
      }
      return Math.round(parsed * 1024 * 1024);
    };

    const minBytes = parseMegabytes(activeFilter.minSizeMB);
    if (typeof minBytes === "number") {
      payload.minSize = minBytes;
    }

    const maxBytes = parseMegabytes(activeFilter.maxSizeMB);
    if (typeof maxBytes === "number") {
      payload.maxSize = maxBytes;
    }

    if (activeFilter.uploadedFrom && activeFilter.uploadedFrom.trim() !== "") {
      const fromDate = new Date(activeFilter.uploadedFrom);
      if (!Number.isNaN(fromDate.getTime())) {
        fromDate.setHours(0, 0, 0, 0);
        payload.uploadedFrom = fromDate.toISOString();
      }
    }

    if (activeFilter.uploadedTo && activeFilter.uploadedTo.trim() !== "") {
      const toDate = new Date(activeFilter.uploadedTo);
      if (!Number.isNaN(toDate.getTime())) {
        toDate.setHours(23, 59, 59, 999);
        payload.uploadedTo = toDate.toISOString();
      }
    }

    return Object.keys(payload).length > 0 ? payload : null;
  }, [activeFilter]);

  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

  const { data, fetching, error, refetch } = useFilesDashboard(graphFilter ?? undefined);
  const viewer = data?.viewer;
  const files = useMemo(() => data?.files.nodes ?? [], [data?.files.nodes]);
  const storageStats = data?.storageStats;
  const [, executeCreateShare] = useMutation<CreateShareResult, { fileId: string; visibility: "PUBLIC" | "PRIVATE"; expiresAt?: string | null }>(CREATE_SHARE_MUTATION);
  const [, executeRevokeShare] = useMutation<RevokeShareResult, { id: string }>(REVOKE_SHARE_MUTATION);
  const [, executeDeleteFile] = useMutation<DeleteFileResult, { id: string }>(DELETE_FILE_MUTATION);
  const filteredFiles = files;

  useEffect(() => {
    if (!fetching && !viewer) {
      router.replace("/login");
    }
  }, [fetching, viewer, router]);

  const quotaUsedPercent = useMemo(() => {
    if (!viewer?.quotaBytes || viewer.quotaBytes <= 0) {
      return 0;
    }
    return Math.min(100, ((storageStats?.totalUsageBytes ?? 0) / viewer.quotaBytes) * 100);
  }, [viewer, storageStats]);

  const statCards = useMemo(() => {
    return [
      {
        label: "Your role",
        value: viewer?.role ?? "USER",
        icon: <ShieldCheck className="h-4 w-4 text-emerald-300" />
      },
      {
        label: "Quota assigned",
        value: formatBytes(viewer?.quotaBytes ?? 0),
        icon: <HardDrive className="h-4 w-4 text-sky-300" />
      },
      {
        label: "Total files",
        value: (data?.files.totalCount ?? 0).toString(),
        icon: <Sparkles className="h-4 w-4 text-purple-300" />
      }
    ];
  }, [viewer, data]);

  const parseErrorResponse = useCallback(async (response: Response) => {
    try {
      const payload = await response.json();
      if (payload && typeof payload.error === "string") {
        return payload.error;
      }
    } catch (_error) {
      // ignore non-JSON responses
    }
    return response.statusText || "Request failed";
  }, []);

  const updateFilterDraft = (patch: Partial<FilterDraft>) => {
    setFilterDraft((prev) => ({ ...prev, ...patch }));
  };

  const toggleMimeType = (value: string) => {
    setFilterDraft((prev) => {
      const exists = prev.mimeTypes.includes(value);
      return {
        ...prev,
        mimeTypes: exists ? prev.mimeTypes.filter((item) => item !== value) : [...prev.mimeTypes, value]
      };
    });
  };

  const applyFilters = () => {
    setActiveFilter({ ...filterDraft });
  };

  const clearFilters = () => {
    setFilterDraft({ ...DEFAULT_FILTER });
    setActiveFilter({ ...DEFAULT_FILTER });
  };

  const isFilterActive = useMemo(() => {
    return (
      activeFilter.search.trim() !== '' ||
      activeFilter.mimeTypes.length > 0 ||
      (activeFilter.minSizeMB && activeFilter.minSizeMB.trim() !== '') ||
      (activeFilter.maxSizeMB && activeFilter.maxSizeMB.trim() !== '') ||
      (activeFilter.uploadedFrom && activeFilter.uploadedFrom.trim() !== '') ||
      (activeFilter.uploadedTo && activeFilter.uploadedTo.trim() !== '')
    );
  }, [activeFilter]);

  const fetchShareInfo = useCallback(async (fileId: string): Promise<ShareDetails | null> => {
    const response = await fetch(`${apiUrl}/files/${fileId}/share`, { credentials: 'include' });
    if (!response.ok) {
      if (response.status === 404) {
        return null;
      }
      throw new Error(await parseErrorResponse(response));
    }
    const payload = await response.json();
    const share = payload?.share;
    if (!share) {
      return null;
    }
    return {
      visibility: (share.visibility ?? 'PRIVATE') as string,
      token: share.token ?? null,
      expiresAt: share.expiresAt ?? null
    };
  }, [apiUrl, parseErrorResponse]);

  const copyShareLink = async (token: string) => {
    const link = `${apiUrl}/shares/${token}/download`;
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(link);
        setShareMessage("Public link copied to clipboard.");
        return;
      }
      throw new Error("clipboard unavailable");
    } catch (_clipboardErr) {
      window.prompt("Copy this share link", link);
      setShareMessage("Public link ready to copy.");
    }
  };

  const refreshShareState = async (fileId: string) => {
    setShareLoadingIds((prev) => ({ ...prev, [fileId]: true }));
    try {
      const info = await fetchShareInfo(fileId);
      setShareState((prev) => ({ ...prev, [fileId]: info }));
    } catch (shareErr) {
      setShareError(shareErr instanceof Error ? shareErr.message : 'Unable to load share info');
    } finally {
      setShareLoadingIds((prev) => ({ ...prev, [fileId]: false }));
    }
  };

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      const nextState: Record<string, ShareDetails | null> = {};
      for (const file of files) {
        try {
          const info = await fetchShareInfo(file.id);
          nextState[file.id] = info;
        } catch (shareErr) {
          if (!cancelled) {
            setShareError(shareErr instanceof Error ? shareErr.message : 'Unable to load share info');
          }
        }
      }
      if (!cancelled) {
        setShareState(nextState);
      }
    };
    if (files.length > 0) {
      load();
    } else {
      setShareState({});
    }
    return () => {
      cancelled = true;
    };
  }, [files, fetchShareInfo]);

  const handleUpload = async (fileList: FileList | File[] | null) => {
    if (!fileList || fileList.length === 0) {
      return;
    }

    const filesArray = Array.from(fileList as ArrayLike<File>);
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

    setIsUploading(true);
    setUploadError(null);
    try {
      const operations = {
        query: `
          mutation UploadFiles($files: [Upload!]!) {
            uploadFiles(files: $files) {
              files {
                id
              }
            }
          }
        `,
        variables: { files: new Array(filesArray.length).fill(null) }
      };

      const formData = new FormData();
      formData.append("operations", JSON.stringify(operations));

      const map: Record<string, string[]> = {};
      filesArray.forEach((_, index) => {
        map[index.toString()] = [`variables.files.${index}`];
      });

      formData.append("map", JSON.stringify(map));

      filesArray.forEach((file, index) => {
        formData.append(index.toString(), file, file.name);
      });

      const response = await fetch(`${apiUrl}/graphql`, {
        method: "POST",
        body: formData,
        credentials: "include"
      });

      const result = await response.json();
      if (result.errors?.length) {
        throw new Error(result.errors[0]?.message ?? "Upload failed");
      }

      refetch();
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const handleDownload = async (file: (typeof files)[number]) => {
    setDownloadError(null);
    setPendingDownloadId(file.id);
    try {
      const response = await fetch(`${apiUrl}/files/${file.id}/download`, {
        credentials: "include"
      });
      if (!response.ok) {
        throw new Error(await parseErrorResponse(response));
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = file.filenameOriginal || "download";
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      window.URL.revokeObjectURL(url);
    } catch (downloadErr) {
      setDownloadError(downloadErr instanceof Error ? downloadErr.message : "Download failed");
    } finally {
      setPendingDownloadId(null);
    }
  };

  const handleGenerateShare = async (file: (typeof files)[number]) => {
    setShareError(null);
    setShareMessage(null);
    setShareLoadingIds((prev) => ({ ...prev, [file.id]: true }));
    try {
      const result = await executeCreateShare({ fileId: file.id, visibility: "PUBLIC", expiresAt: null });
      if (result.error) {
        throw result.error;
      }
      const share = result.data?.createShare;
      if (!share?.token) {
        throw new Error("No share token returned");
      }
      await refreshShareState(file.id);
      await copyShareLink(share.token);
    } catch (shareErr) {
      setShareError(shareErr instanceof Error ? shareErr.message : "Unable to create share");
    } finally {
      setShareLoadingIds((prev) => ({ ...prev, [file.id]: false }));
    }
  };

  const handleCreatePrivateLink = async (file: (typeof files)[number]) => {
    setShareError(null);
    setShareMessage(null);
    setShareLoadingIds((prev) => ({ ...prev, [file.id]: true }));
    try {
      const result = await executeCreateShare({ fileId: file.id, visibility: "PRIVATE", expiresAt: null });
      if (result.error) {
        throw result.error;
      }
      const token = result.data?.createShare?.token;
      if (!token) {
        throw new Error("No share token returned");
      }
      await refreshShareState(file.id);
      await copyShareLink(token);
    } catch (err) {
      setShareError(err instanceof Error ? err.message : "Unable to create private link");
    } finally {
      setShareLoadingIds((prev) => ({ ...prev, [file.id]: false }));
    }
  };

  const handleDisableShare = async (file: (typeof files)[number]) => {
    setShareError(null);
    setShareMessage(null);
    setShareLoadingIds((prev) => ({ ...prev, [file.id]: true }));
    try {
      const result = await executeRevokeShare({ id: file.id });
      if (result.error) {
        throw result.error;
      }
      if (!result.data?.revokeShare?.ok) {
        throw new Error("Share revoke failed");
      }
      await refreshShareState(file.id);
      setShareMessage("Public link disabled.");
    } catch (shareErr) {
      setShareError(shareErr instanceof Error ? shareErr.message : "Unable to disable share");
    } finally {
      setShareLoadingIds((prev) => ({ ...prev, [file.id]: false }));
    }
  };

  const handleDeleteFile = async (file: (typeof files)[number]) => {
    if (!confirm(`Delete "${file.filenameOriginal}"? This action cannot be undone.`)) {
      return;
    }
    setShareMessage(null);
    setShareError(null);
    setDeleteError(null);
    setDeleteMessage(null);
    setDeleteLoadingIds((prev) => ({ ...prev, [file.id]: true }));
    try {
      const result = await executeDeleteFile({ id: file.id });
      if (result.error) {
        throw result.error;
      }
      const ok = result.data?.deleteFile?.ok ?? false;
      setShareState((prev) => {
        if (!(file.id in prev)) {
          return prev;
        }
        const next = { ...prev };
        delete next[file.id];
        return next;
      });
      await refetch();
      if (ok) {
        setDeleteMessage(`Deleted "${file.filenameOriginal}".`);
      } else {
        setDeleteMessage(`"${file.filenameOriginal}" was already removed.`);
      }
    } catch (deleteErr) {
      setDeleteError(deleteErr instanceof Error ? deleteErr.message : "Unable to delete file");
    } finally {
      setDeleteLoadingIds((prev) => {
        const next = { ...prev };
        delete next[file.id];
        return next;
      });
    }
  };

  const handleDragEnter = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    dragCounterRef.current += 1;
    if (!isDragging) {
      setIsDragging(true);
    }
  };

  const handleDragLeave = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    dragCounterRef.current = Math.max(0, dragCounterRef.current - 1);
    if (dragCounterRef.current === 0) {
      setIsDragging(false);
    }
  };

  const handleDragOver = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!isDragging) {
      setIsDragging(true);
    }
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    dragCounterRef.current = 0;
    setIsDragging(false);
    const { files: droppedFiles } = event.dataTransfer;
    if (droppedFiles && droppedFiles.length > 0) {
      void handleUpload(droppedFiles);
    }
  };


  if (fetching && !viewer) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-200">
        <div className="animate-pulse text-sm text-slate-400">Loading your workspace...</div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-100">
        <div className="max-w-md rounded-2xl border border-white/10 bg-white/5 p-6 text-center shadow-glow">
          <p className="text-lg font-semibold text-white">We hit a snag</p>
          <p className="mt-2 text-sm text-slate-300/80">{error.message}</p>
          <Link
            href="/"
            className="mt-6 inline-flex rounded-xl bg-white px-4 py-2 text-sm font-semibold text-slate-900"
          >
            Return home
          </Link>
        </div>
      </main>
    );
  }

  if (!viewer) {
    return null;
  }

  const savingsLabel = formatBytes(storageStats?.savingsBytes ?? 0);

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100">
      <div className="border-b border-white/10 bg-brand-surface/60 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-5 lg:px-10">
          <div className="flex items-center gap-6">
            <Link href="/" className="text-lg font-semibold tracking-tight text-white">
              BalkanID <span className="text-brand-accent">Vault</span>
            </Link>
            <Link href="/public" className="text-sm text-slate-300 hover:text-white">Public</Link>
          </div>
          <div className="flex items-center gap-3 text-sm text-slate-300">
            <div className="hidden flex-col text-right md:flex">
              <span className="font-medium text-white">{viewer.name ?? "Vault member"}</span>
              <span className="text-xs text-slate-400">{viewer.email}</span>
            </div>
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-brand-accent/30 text-sm font-semibold text-white">
              {(viewer.name ?? viewer.email ?? "?").charAt(0).toUpperCase()}
            </div>
          </div>
        </div>
      </div>

      <div className="mx-auto max-w-6xl px-6 py-10 lg:px-10">
        <div className="rounded-3xl border border-white/10 bg-gradient-card p-8 shadow-glow">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.4em] text-slate-300">Dashboard</p>
              <h1 className="mt-2 text-3xl font-semibold text-white md:text-4xl">
                Welcome back, {viewer.name ?? viewer.email}
              </h1>
              <p className="mt-4 max-w-2xl text-sm text-slate-200/80">
                Upload files, manage deduplicated storage, and share securely with your team. Advanced analytics and
                activity feeds are on the way.
              </p>
            </div>
            <div
              onDragEnter={handleDragEnter}
              onDragLeave={handleDragLeave}
              onDragOver={handleDragOver}
              onDrop={handleDrop}
              className={cn("flex flex-col gap-3 text-sm text-slate-200/80 rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface transition",
                isDragging ? "border-brand-accent/70 bg-brand-surface/40" : undefined
              )}
            >
              <button
                type="button"
                className={cn(
                  "inline-flex items-center gap-2 rounded-xl bg-white px-5 py-3 font-semibold text-slate-950 transition",
                  isUploading ? "opacity-70" : "hover:bg-slate-100"
                )}
                onClick={() => fileInputRef.current?.click()}
                disabled={isUploading}
              >
                <CloudUpload className="h-4 w-4" />
                {isUploading ? "Uploading..." : "Upload files"}
              </button>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                hidden
                onChange={(event) => handleUpload(event.target.files)}
              />
              <span className="text-xs text-slate-400">
                Drop files anywhere in this panel or use the button above. Maximum upload size is 10 MB per file by default.
              </span>
              {uploadError ? <span className="text-xs text-red-400">{uploadError}</span> : null}
            </div>
          </div>
        </div>

        <section className="mt-10 grid gap-5 md:grid-cols-3">
          {statCards.map(({ label, value, icon }) => (
            <div key={label} className="rounded-2xl border border-white/10 bg-white/5 p-5 shadow-surface">
              <div className="flex items-center justify-between text-xs text-slate-400">
                <span>{label}</span>
                {icon}
              </div>
              <p className="mt-3 text-xl font-semibold text-white">{value}</p>
            </div>
          ))}
          <div className="rounded-2xl border border-white/10 bg-white/5 p-5 shadow-surface">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span>Quota usage</span>
              <span>{quotaUsedPercent.toFixed(1)}%</span>
            </div>
            <div className="mt-3 h-2 w-full overflow-hidden rounded-full bg-slate-800">
              <div
                className="h-full rounded-full bg-gradient-to-r from-brand-accent to-sky-400 bg-sky-500"
                style={{ width: `${quotaUsedPercent}%` }}
              />
            </div>
            <p className="mt-3 text-xs text-slate-300/70">
              {formatBytes(storageStats?.totalUsageBytes ?? 0)} used of {formatBytes(viewer.quotaBytes ?? 0)} quota.
            </p>
          </div>
          <div className="rounded-2xl border border-white/10 bg-white/5 p-5 shadow-surface">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span>Dedupe savings</span>
              <span className="text-emerald-300">{storageStats?.savingsPercent.toFixed(1) ?? "0.0"}%</span>
            </div>
            <p className="mt-3 text-xl font-semibold text-white">{savingsLabel}</p>
            <p className="mt-2 text-xs text-slate-300/70">
              Savings are calculated against original file sizes for this account.
            </p>
          </div>
        </section>

        <section className="mt-12 rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface">
          <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div className="flex w-full flex-col gap-2 md:max-w-sm">
              <label className="text-xs font-semibold uppercase tracking-wide text-slate-300">Search</label>
              <input
                type="text"
                value={filterDraft.search}
                onChange={(event) => updateFilterDraft({ search: event.target.value })}
                placeholder="Search by name"
                className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
              />
            </div>
            <div className="flex w-full flex-col gap-2 md:max-w-xs">
              <label className="text-xs font-semibold uppercase tracking-wide text-slate-300">Size (MB)</label>
              <div className="flex gap-2">
                <input
                  type="number"
                  min="0"
                  value={filterDraft.minSizeMB}
                  onChange={(event) => updateFilterDraft({ minSizeMB: event.target.value })}
                  placeholder="Min"
                  className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
                />
                <input
                  type="number"
                  min="0"
                  value={filterDraft.maxSizeMB}
                  onChange={(event) => updateFilterDraft({ maxSizeMB: event.target.value })}
                  placeholder="Max"
                  className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
                />
              </div>
            </div>
            <div className="flex w-full flex-col gap-2 md:max-w-xs">
              <label className="text-xs font-semibold uppercase tracking-wide text-slate-300">Date range</label>
              <div className="flex gap-2">
                <input
                  type="date"
                  value={filterDraft.uploadedFrom}
                  onChange={(event) => updateFilterDraft({ uploadedFrom: event.target.value })}
                  className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white focus:border-brand-accent focus:outline-none"
                />
                <input
                  type="date"
                  value={filterDraft.uploadedTo}
                  onChange={(event) => updateFilterDraft({ uploadedTo: event.target.value })}
                  className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white focus:border-brand-accent focus:outline-none"
                />
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={applyFilters}
                className="rounded-full border border-brand-accent/70 px-4 py-2 text-xs font-semibold text-brand-accent transition hover:bg-brand-accent/10"
              >
                Apply filters
              </button>
              <button
                type="button"
                onClick={clearFilters}
                disabled={!isFilterActive}
                className="rounded-full border border-white/20 px-4 py-2 text-xs font-semibold text-slate-200 transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-60"
              >
                Clear
              </button>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            {MIME_TYPE_OPTIONS.map((option) => {
              const active = filterDraft.mimeTypes.includes(option.value);
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => toggleMimeType(option.value)}
                  className={cn("rounded-full border px-3 py-1 text-xs font-semibold transition",
                    active ? "border-brand-accent/70 bg-brand-accent/20 text-brand-accent" : "border-white/10 text-slate-300 hover:bg-white/10"
                  )}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
        </section>

        <section className="mt-12 rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold text-white">Your files</h2>
            <span className="text-xs text-slate-400">{filteredFiles.length} of {files.length} showing</span>
          </div>

          {downloadError ? (
            <div className="mt-4 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-200">{downloadError}</div>
          ) : null}
          {shareError ? (
            <div className="mt-2 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-200">{shareError}</div>
          ) : null}
          {shareMessage ? (
            <div className="mt-2 rounded-lg border border-emerald-400/40 bg-emerald-500/10 px-4 py-2 text-sm text-emerald-200">{shareMessage}</div>
          ) : null}

          {deleteError ? (
            <div className="mt-2 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-200">{deleteError}</div>
          ) : null}
          {deleteMessage ? (
            <div className="mt-2 rounded-lg border border-amber-400/40 bg-amber-500/10 px-4 py-2 text-sm text-amber-200">{deleteMessage}</div>
          ) : null}

          {filteredFiles.length === 0 ? (
            files.length === 0 ? (
              <div className="mt-8 rounded-xl border border-dashed border-white/10 bg-white/5 p-6 text-center text-sm text-slate-300/80">
                <Archive className="mx-auto h-8 w-8 text-brand-accent" />
                <p className="mt-3 font-medium text-white">No files uploaded yet</p>
                <p className="mt-2 text-xs text-slate-300/70">
                  Drop files here or use the upload button above to start populating your vault.
                </p>
              </div>
            ) : (
              <div className="mt-8 rounded-xl border border-dashed border-white/10 bg-white/5 p-6 text-center text-sm text-slate-300/80">
                <p className="mt-3 font-medium text-white">No files match your filters</p>
                <p className="mt-2 text-xs text-slate-300/70">Adjust or clear filters to see more files.</p>
                <button
                  type="button"
                  onClick={clearFilters}
                  className="mt-4 inline-flex items-center gap-2 rounded-full border border-white/20 px-3 py-1 text-xs font-semibold text-white transition hover:bg-white/10"
                >
                  Reset filters
                </button>
              </div>
            )
          ) : (
            <div className="mt-6 overflow-x-auto rounded-xl border border-white/10">
              <table className="min-w-full divide-y divide-white/10 text-sm">
                <thead className="bg-white/5 text-xs uppercase tracking-wide text-slate-300">
                  <tr>
                    <th className="px-4 py-3 text-left">Name</th>
                    <th className="px-4 py-3 text-left">MIME</th>
                    <th className="px-4 py-3 text-right">Size</th>
                    <th className="px-4 py-3 text-right">Downloads</th>
                    <th className="px-4 py-3 text-left">Uploaded</th>
                    <th className="px-4 py-3 text-left">Status</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5 bg-slate-950/40 text-slate-200">
                  {filteredFiles.map((file) => (
                    <tr key={file.id}>
                      <td className="px-4 py-3 font-medium text-white">{file.filenameOriginal}</td>
                      <td className="px-4 py-3 text-slate-300">
                        {file.mimeDeclared ?? file.mimeDetected ?? "Unknown"}
                      </td>
                      <td className="px-4 py-3 text-right text-slate-200/80">{formatBytes(file.sizeBytesOriginal)}</td>
                      <td className="px-4 py-3 text-right text-slate-200/80">{file.downloadCount}</td>
                      <td className="px-4 py-3 text-slate-200/80">{formatDate(file.uploadedAt)}</td>
                      <td className="px-4 py-3">
                        <div className="flex flex-col items-start gap-1">
                          <span
                            className={cn(
                              "inline-flex items-center rounded-full px-2 py-1 text-xs font-semibold",
                              file.deduped ? "bg-emerald-500/20 text-emerald-300" : "bg-slate-500/20 text-slate-200"
                            )}
                          >
                            {file.deduped ? "Deduped" : "Unique"}
                          </span>
                          {shareState[file.id]?.visibility === "PUBLIC" ? (
                            <span className="inline-flex items-center rounded-full bg-sky-500/20 px-2 py-1 text-xs font-semibold text-sky-300">
                              Public
                            </span>
                          ) : null}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex flex-wrap items-center justify-end gap-2">
                          <button
                            type="button"
                            onClick={() => handleDownload(file)}
                            disabled={pendingDownloadId === file.id}
                            className={cn(
                              "rounded-full border border-white/20 px-3 py-1 text-xs font-semibold transition",
                              pendingDownloadId === file.id ? "bg-white/10 text-slate-300" : "text-slate-100 hover:bg-white/10"
                            )}
                          >
                            {pendingDownloadId === file.id ? "Downloading..." : "Download"}
                          </button>
                          <button
                            type="button"
                            onClick={() =>
                              shareState[file.id]?.visibility === "PUBLIC"
                                ? (async () => { await executeCreateShare({ fileId: file.id, visibility: "PRIVATE", expiresAt: null }); await refreshShareState(file.id); setShareMessage("Made private."); })()
                                : handleGenerateShare(file)
                            }
                            disabled={Boolean(shareLoadingIds[file.id])}
                            className={cn(
                              "rounded-full border px-3 py-1 text-xs font-semibold transition",
                              shareState[file.id]?.visibility === "PUBLIC"
                                ? "border-red-400/60 text-red-300 hover:bg-red-400/10"
                                : "border-brand-accent/60 text-brand-accent hover:bg-brand-accent/10"
                            )}
                          >
                            {shareLoadingIds[file.id]
                              ? "Updating..."
                              : shareState[file.id]?.visibility === "PUBLIC"
                                ? "Make Private"
                                : "Make Public"}
                          </button>
                          <button
                            type="button"
                            onClick={() => handleCreatePrivateLink(file)}
                            disabled={Boolean(shareLoadingIds[file.id])}
                            className="rounded-full border border-white/20 px-3 py-1 text-xs font-semibold text-slate-100 transition hover:bg-white/10"
                          >
                            Copy private link
                          </button>
                          {shareState[file.id]?.token ? (
                            <button
                              type="button"
                              onClick={() => copyShareLink(shareState[file.id]!.token as string)}
                              className="rounded-full border border-white/20 px-3 py-1 text-xs font-semibold text-slate-100 transition hover:bg-white/10"
                            >
                              Copy current link
                            </button>
                          ) : null}
                          <button
                            type="button"
                            onClick={() => handleDisableShare(file)}
                            disabled={Boolean(shareLoadingIds[file.id])}
                            className={cn(
                              "rounded-full border px-3 py-1 text-xs font-semibold transition",
                              "border-red-400/60 text-red-300 hover:bg-red-400/10"
                            )}
                          >
                            Disable link
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDeleteFile(file)}
                            disabled={Boolean(deleteLoadingIds[file.id])}
                            className={cn(
                              "rounded-full border px-3 py-1 text-xs font-semibold transition",
                              deleteLoadingIds[file.id]
                                ? "border-red-500/40 bg-red-500/10 text-red-200/70"
                                : "border-red-500/60 text-red-200 hover:bg-red-500/10"
                            )}
                          >
                            {deleteLoadingIds[file.id] ? "Deleting..." : "Delete"}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="mt-12 grid gap-6 md:grid-cols-2">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface">
            <h2 className="text-lg font-semibold text-white">Highlights</h2>
            <ul className="mt-4 space-y-3 text-sm text-slate-300/80">
              <li>• Drag-and-drop multi-upload with quota awareness.</li>
              <li>• Deep filtering by MIME, tags, uploader, size, and date.</li>
              <li>• Public/private links with secure downloads and tracking.</li>
            </ul>
          </div>
          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface">
            <h2 className="text-lg font-semibold text-white">Need to invite stakeholders?</h2>
            <p className="mt-3 text-sm text-slate-300/80">
              Configure Supabase service keys in <code className="rounded bg-slate-900/70 px-1 py-0.5 text-xs">.env</code>,
              then start the backend. Everyone with Google SSO can join instantly.
            </p>
            <Link
              href="/setup"
              className="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-brand-accent transition hover:text-white"
            >
              View setup guide <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
        </section>
      </div>
    </main>
  );
}



