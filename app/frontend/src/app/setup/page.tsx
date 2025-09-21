export default function SetupGuide() {
    return (
        <main className="min-h-screen bg-slate-950 text-slate-100">
            <div className="mx-auto max-w-3xl px-6 py-12 lg:px-10">
                <h1 className="text-3xl font-semibold text-white">Setup Guide</h1>
                <p className="mt-2 text-slate-300/80">Follow these steps to run BalkanID Vault locally.</p>

                <section className="mt-8 space-y-4 text-sm leading-6 text-slate-300/90">
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                        <h2 className="text-lg font-semibold text-white">Environment variables</h2>
                        <p className="mt-2">Create a <code className="rounded bg-slate-900/70 px-1 py-0.5">.env</code> file for the backend with:</p>
                        <pre className="mt-3 overflow-x-auto rounded-xl border border-white/10 bg-slate-900/70 p-4 text-xs text-slate-200">
                            PORT=8080
                            FRONTEND_URL=http://localhost:3000
                            JWT_SECRET=change-me
                            SESSION_COOKIE_NAME=vault_session
                            DEFAULT_USER_QUOTA_BYTES=10737418240
                            MAX_UPLOAD_BYTES=10485760
                            SUPABASE_URL=your_supabase_url
                            SUPABASE_ANON_KEY=your_public_anon_key
                            SUPABASE_SERVICE_ROLE_KEY=your_service_role_key
                            SUPABASE_DB_URL=postgresql://user:pass@host:5432/dbname
                            STORAGE_BUCKET=blobs
                            REDIS_URL=redis://localhost:6379
                            OAUTH_REDIRECT_URL=http://localhost:8080/auth/google/callback
                            GOOGLE_CLIENT_ID=your_google_client_id
                            GOOGLE_CLIENT_SECRET=your_google_client_secret
                        </pre>
                    </div>

                    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                        <h2 className="text-lg font-semibold text-white">Frontend config</h2>
                        <p className="mt-2">Set the API base URL for the Next.js app:</p>
                        <pre className="mt-3 overflow-x-auto rounded-xl border border-white/10 bg-slate-900/70 p-4 text-xs text-slate-200">
                            NEXT_PUBLIC_API_URL=http://localhost:8080
                        </pre>
                    </div>

                    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                        <h2 className="text-lg font-semibold text-white">Run locally</h2>
                        <ol className="mt-2 list-decimal space-y-2 pl-5">
                            <li>Start the backend: <code className="rounded bg-slate-900/70 px-1 py-0.5">go run ./app/backend/cmd/server</code></li>
                            <li>Start the frontend: <code className="rounded bg-slate-900/70 px-1 py-0.5">cd app/frontend && npm install && npm run dev</code></li>
                            <li>Open the app at <code className="rounded bg-slate-900/70 px-1 py-0.5">http://localhost:3000</code></li>
                        </ol>
                    </div>

                    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                        <h2 className="text-lg font-semibold text-white">Google OAuth</h2>
                        <p className="mt-2">Create OAuth credentials in Google Cloud Console and set the Client ID/Secret above. Authorized redirect URI must be:</p>
                        <pre className="mt-3 overflow-x-auto rounded-xl border border-white/10 bg-slate-900/70 p-4 text-xs text-slate-200">http://localhost:8080/auth/google/callback</pre>
                    </div>
                </section>
            </div>
        </main>
    );
}
