import Link from "next/link";
import { ArrowRight, CloudUpload, ShieldCheck, Sparkles } from "lucide-react";

const featureCards = [
  {
    title: "Zero-trust vault",
    description: "Content hashing, quotas, and audit trails keep every file accountable.",
    icon: ShieldCheck
  },
  {
    title: "Instant intelligence",
    description: "Search by metadata, tags, MIME type, and date without throttling performance.",
    icon: Sparkles
  },
  {
    title: "One-click uploads",
    description: "Drag-and-drop multi-upload with automatic dedupe saves time and storage.",
    icon: CloudUpload
  }
];

export default function Home() {
  return (
    <main className="relative min-h-screen overflow-hidden">
      <div className="absolute inset-0 bg-gradient-hero" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(79,70,229,0.45),transparent_45%)]" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_bottom_left,rgba(37,99,235,0.35),transparent_40%)]" />

      <div className="relative mx-auto max-w-6xl px-6 pb-24 pt-12 lg:px-10">
        <header className="flex flex-wrap items-center justify-between gap-4 text-slate-100">
          <Link href="/" className="text-xl font-semibold tracking-tight">
            BalkanID <span className="text-brand-accent">Vault</span>
          </Link>
          <div className="flex items-center gap-3 text-sm">
            <span className="hidden text-slate-300 md:inline">Already onboard?</span>
            <Link
              href="/login"
              className="rounded-full border border-slate-300/30 px-5 py-2 text-slate-50 transition hover:border-brand-accent/60 hover:bg-brand-accent/10"
            >
              Sign in
            </Link>
            <Link
              href="/files"
              className="inline-flex items-center gap-2 rounded-full bg-white/10 px-5 py-2 font-medium text-white shadow-glow transition hover:bg-white/20"
            >
              Launch App <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </header>

        <section className="mt-20 grid gap-16 lg:grid-cols-[1.4fr_1fr] lg:items-center">
          <div className="space-y-8 text-slate-100">
            <div className="inline-flex items-center gap-3 rounded-full border border-white/10 bg-white/5 px-5 py-2 text-xs uppercase tracking-[0.4em] text-slate-200/80">
              Secure. Searchable. Shareable.
            </div>
            <h1 className="text-4xl font-semibold leading-tight text-white md:text-6xl">
              A modern vault for teams that need trust, speed, and clarity.
            </h1>
            <p className="max-w-2xl text-lg text-slate-200/80">
              Upload once. Deduplicate intelligently. Share with confidence. BalkanID Vault combines granular access
              controls, real-time analytics, and delightful UX into a single secure workspace.
            </p>
            <div className="flex flex-wrap items-center gap-4">
              <Link
                href="/login"
                className="inline-flex items-center gap-2 rounded-xl bg-white px-6 py-3 text-base font-semibold text-slate-950 shadow-glow transition hover:shadow-lg"
              >
                Sign in with Google <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                href="#highlights"
                className="inline-flex items-center gap-2 rounded-xl border border-white/30 px-6 py-3 text-base font-medium text-slate-100 transition hover:border-white"
              >
                Explore highlights
              </Link>
            </div>
            <dl className="grid gap-6 text-sm text-slate-200/70 md:grid-cols-3">
              <div>
                <dt className="text-slate-300">Upload streak</dt>
                <dd className="text-xl font-semibold text-white">Drag, drop, done</dd>
              </div>
              <div>
                <dt className="text-slate-300">Dedup savings</dt>
                <dd className="text-xl font-semibold text-white">Up to 60% reclaimed</dd>
              </div>
              <div>
                <dt className="text-slate-300">Access control</dt>
                <dd className="text-xl font-semibold text-white">Zero-trust ready</dd>
              </div>
            </dl>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface">
            <h3 className="text-lg font-semibold text-white">Why Vault?</h3>
            <ul className="mt-4 space-y-2 text-sm text-slate-200/80">
              <li>• Smart deduplication saves storage automatically.</li>
              <li>• Powerful search by metadata, tags, and MIME type.</li>
              <li>• Public and private sharing with download tracking.</li>
            </ul>
            <Link href="/setup" className="mt-5 inline-flex items-center gap-2 text-sm font-semibold text-brand-accent transition hover:text-white">
              View setup guide <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </section>

        <section id="highlights" className="mt-24 grid gap-6 md:grid-cols-3">
          {featureCards.map(({ title, description, icon: Icon }) => (
            <article
              key={title}
              className="rounded-2xl border border-white/10 bg-white/5 p-6 shadow-surface transition hover:border-brand-accent/60 hover:shadow-glow"
            >
              <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-brand-accent/20 text-brand-accent">
                <Icon className="h-6 w-6" />
              </div>
              <h3 className="text-lg font-semibold text-white">{title}</h3>
              <p className="mt-2 text-sm text-slate-200/70">{description}</p>
            </article>
          ))}
        </section>

        <section className="mt-20 rounded-3xl border border-white/10 bg-white/5 p-8 text-slate-100 shadow-glow">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 className="text-2xl font-semibold">Ready to reclaim your storage budget?</h2>
              <p className="mt-2 max-w-xl text-slate-200/80">
                Spin up the backend, invite your team, and start sharing confidently in minutes.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Link
                href="/login"
                className="inline-flex items-center gap-2 rounded-xl bg-white px-5 py-3 text-sm font-semibold text-slate-950 transition hover:bg-slate-100"
              >
                Sign in to continue <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                href="/files"
                className="inline-flex items-center gap-2 rounded-xl border border-white/20 px-5 py-3 text-sm font-semibold text-white transition hover:border-white"
              >
                Go to dashboard
              </Link>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
