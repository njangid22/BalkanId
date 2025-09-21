"use client";

import { useMemo, useState } from "react";
import { useQuery } from "urql";
import Link from "next/link";

const PUBLIC_FILES_QUERY = /* GraphQL */ `
  query PublicFiles($filter: FileFilter) {
    files(scope: PUBLIC, filter: $filter) {
      totalCount
      nodes {
        id
        filenameOriginal
        sizeBytesOriginal
        mimeDeclared
        mimeDetected
        uploadedAt
        downloadCount
        deduped
        tags
        owner { id name email }
      }
    }
  }
`;

function formatBytes(bytes?: number | null) {
    if (!bytes || Number.isNaN(bytes)) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB"] as const;
    let value = bytes;
    let i = 0;
    while (value >= 1024 && i < units.length - 1) {
        value /= 1024;
        i += 1;
    }
    return `${value.toFixed(value < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
}

export default function PublicPage() {
    const [uploaderName, setUploaderName] = useState("");
    const [uploaderId, setUploaderId] = useState("");
    const [search, setSearch] = useState("");
    const [mime, setMime] = useState("");
    const [pendingDownloadId, setPendingDownloadId] = useState<string | null>(null);
    const [downloadError, setDownloadError] = useState<string | null>(null);
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

    const filter = useMemo(() => {
        const f: Record<string, unknown> = {};
        if (search.trim()) f.search = search.trim();
        if (mime.trim()) f.mimeTypes = [mime.trim()];
        if (uploaderName.trim()) f.uploaderName = uploaderName.trim();
        if (uploaderId.trim()) f.uploaderId = uploaderId.trim();
        return Object.keys(f).length ? f : null;
    }, [search, mime, uploaderName, uploaderId]);

    const [{ data, fetching, error }] = useQuery<{ files: { totalCount: number; nodes: any[] } }>({
        query: PUBLIC_FILES_QUERY,
        variables: { filter },
        requestPolicy: "cache-and-network",
    });

    const handlePublicDownload = async (file: any) => {
        try {
            setDownloadError(null);
            setPendingDownloadId(file.id);
            const response = await fetch(`${apiUrl}/public/files/${file.id}/download`);
            if (!response.ok) {
                const text = await response.text();
                throw new Error(text || response.statusText || "Download failed");
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
        } catch (err) {
            setDownloadError(err instanceof Error ? err.message : "Download failed");
        } finally {
            setPendingDownloadId(null);
        }
    };

    return (
        <main className="p-6 max-w-6xl mx-auto">
            <div className="mb-6 flex items-center justify-between">
                <h1 className="text-2xl font-semibold">Public Files</h1>
                <Link href="/files" className="text-sky-500 hover:underline">Back to your files</Link>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mb-6">
                <input
                    className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
                    placeholder="Search filename..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
                <input
                    className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
                    placeholder="MIME (e.g. image/)"
                    value={mime}
                    onChange={(e) => setMime(e.target.value)}
                />
                <input
                    className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
                    placeholder="Uploader name or email"
                    value={uploaderName}
                    onChange={(e) => setUploaderName(e.target.value)}
                />
                <input
                    className="w-full rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-white placeholder:text-slate-400 focus:border-brand-accent focus:outline-none"
                    placeholder="Uploader ID (UUID)"
                    value={uploaderId}
                    onChange={(e) => setUploaderId(e.target.value)}
                />
            </div>

            {error && (
                <div className="text-red-500 mb-4">{error.message}</div>
            )}

            {fetching ? (
                <div className="space-y-2">
                    {Array.from({ length: 6 }).map((_, i) => (
                        <div key={i} className="animate-pulse rounded border border-white/10 bg-white/5 p-3">
                            <div className="h-4 w-1/3 rounded bg-white/10" />
                            <div className="mt-2 h-3 w-1/2 rounded bg-white/10" />
                        </div>
                    ))}
                </div>
            ) : (
                <div>
                    <div className="text-sm text-muted-foreground mb-2">
                        {data?.files.totalCount ?? 0} results
                    </div>
                    {downloadError ? (
                        <div className="mb-3 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-200">{downloadError}</div>
                    ) : null}
                    {(!data?.files.nodes || data.files.nodes.length === 0) ? (
                        <div className="rounded border border-dashed border-white/10 p-8 text-center text-sm text-muted-foreground">
                            No public files match your filter.
                            <button
                                className="ml-2 rounded-full border border-brand-accent/70 px-3 py-1 text-xs font-semibold text-brand-accent hover:bg-brand-accent/10"
                                onClick={() => { setSearch(""); setMime(""); setUploaderName(""); setUploaderId(""); }}
                            >
                                Clear filters
                            </button>
                        </div>
                    ) : (
                        <div className="divide-y border rounded">
                            {data.files.nodes.map((f) => (
                                <div key={f.id} className="p-3 flex items-center justify-between gap-4">
                                    <div>
                                        <div className="font-medium">{f.filenameOriginal}</div>
                                        <div className="text-xs text-muted-foreground">
                                            {formatBytes(f.sizeBytesOriginal)} • {(f.mimeDeclared || f.mimeDetected) ?? "unknown"}
                                        </div>
                                        <div className="text-xs text-muted-foreground">
                                            by {f.owner?.name || f.owner?.email} • uploaded {new Date(f.uploadedAt).toLocaleString()}
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-3">
                                        <span className="text-xs text-muted-foreground">downloads: {f.downloadCount}</span>
                                        <button
                                            className="rounded-full border border-white/20 px-3 py-1 text-xs font-semibold text-white hover:bg-white/10"
                                            onClick={() => handlePublicDownload(f)}
                                            disabled={pendingDownloadId === f.id}
                                        >
                                            {pendingDownloadId === f.id ? "Downloading..." : "Download"}
                                        </button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </main>
    );
}
