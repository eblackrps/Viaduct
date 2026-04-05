import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { DiscoveryResult } from "../types";

interface PlatformSummaryProps {
  inventory: DiscoveryResult | null;
}

export function PlatformSummary({ inventory }: PlatformSummaryProps) {
  const rows = Object.values(
    (inventory?.vms ?? []).reduce<Record<string, { platform: string; count: number; cpu: number; memory: number }>>((accumulator, vm) => {
      const item = accumulator[vm.platform] ?? {
        platform: vm.platform,
        count: 0,
        cpu: 0,
        memory: 0,
      };
      item.count += 1;
      item.cpu += vm.cpu_count;
      item.memory += vm.memory_mb;
      accumulator[vm.platform] = item;
      return accumulator;
    }, {}),
  );

  return (
    <section className="grid gap-5 xl:grid-cols-[1.3fr_1fr]">
      <div className="panel p-5">
        <p className="font-display text-2xl text-ink">Platform Totals</p>
        <p className="mt-1 text-sm text-slate-500">A quick capacity snapshot across the discovered estate.</p>

        <div className="mt-5 grid gap-4 md:grid-cols-3">
          {rows.map((row) => (
            <article key={row.platform} className="rounded-3xl bg-slate-50 p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-500">{row.platform}</p>
              <p className="mt-3 font-display text-3xl text-ink">{row.count}</p>
              <p className="mt-1 text-sm text-slate-500">{row.cpu} vCPU / {row.memory.toLocaleString()} MB</p>
            </article>
          ))}
        </div>
      </div>

      <div className="panel h-[320px] p-5">
        <p className="font-display text-2xl text-ink">Hypervisor Spread</p>
        <p className="mt-1 text-sm text-slate-500">Compare VM counts side by side before planning a migration wave.</p>
        <div className="mt-5 h-[220px]">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={rows}>
              <CartesianGrid strokeDasharray="4 4" vertical={false} stroke="#e2e8f0" />
              <XAxis dataKey="platform" tickLine={false} axisLine={false} />
              <YAxis tickLine={false} axisLine={false} />
              <Tooltip />
              <Bar dataKey="count" fill="#1f4e79" radius={[10, 10, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </section>
  );
}
