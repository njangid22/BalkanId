"use client";

import Link from "next/link";

const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const oauthUrl = `${apiUrl}/auth/google/start`;

export default function LoginPage() {
  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-slate-950">
      <div className="absolute inset-0 bg-gradient-hero opacity-90" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(37,99,235,0.45),transparent_55%)]" />

      <div className="relative mx-auto w-full max-w-3xl rounded-3xl border border-white/10 bg-white/10 px-8 py-12 shadow-glow backdrop-blur">
        <div className="flex flex-col gap-8 md:flex-row md:items-center md:justify-between">
          <div className="space-y-4 text-slate-100">
            <span className="inline-flex items-center gap-2 rounded-full bg-white/10 px-4 py-1 text-xs uppercase tracking-[0.35em]">
              BalkanID Vault
            </span>
            <h1 className="text-3xl font-semibold leading-tight md:text-4xl">
              Sign in with Google to unlock your secure file workspace.
            </h1>
            <p className="text-sm text-slate-200/80">
              We use Google OAuth for just-in-time identity, then issue a short-lived session to keep your files safe.
              No passwords to remember, no friction.
            </p>
          </div>

          <div className="flex w-full max-w-sm flex-col gap-4 rounded-2xl border border-white/10 bg-slate-950/70 p-6 shadow-surface">
            <button
              className="inline-flex items-center justify-center gap-3 rounded-xl bg-white px-5 py-3 text-sm font-semibold text-slate-900 transition hover:bg-slate-200"
              onClick={() => {
                window.location.href = oauthUrl;
              }}
            >
              <svg aria-hidden="true" className="h-5 w-5" viewBox="0 0 24 24">
                <path
                  d="M12 11.9v3.8h5.4c-.2 1.4-1.6 4.2-5.4 4.2-3.3 0-6-2.7-6-6s2.7-6 6-6c1.9 0 3.1.8 3.8 1.4l2.6-2.5C16.5 4.6 14.5 3.6 12 3.6c-5.1 0-9.3 4.1-9.3 9.3s4.2 9.3 9.3 9.3c5.4 0 9-3.9 9-9.5 0-.6-.1-1-.1-1.4H12z"
                  fill="#1B1B1B"
                />
              </svg>
              Continue with Google
            </button>
            <p className="text-xs text-slate-400">
              By continuing you agree to our zero-trust access policy. You will be prompted to grant read-only access to
              your Google profile to verify your identity.
            </p>
            <Link
              href="/"
              className="text-xs font-medium text-slate-200/70 underline-offset-4 transition hover:text-white hover:underline"
            >
              Back to landing
            </Link>
          </div>
        </div>
      </div>
    </main>
  );
}
