type BadgeProps = { children: React.ReactNode; variant?: 'default' | 'overnight' | 'concerning' | 'critical' | 'verified' | 'signal'; className?: string };
export function Badge({ children, variant = 'default', className = '' }: BadgeProps) {
  const colors = {
    default: 'bg-surface-2 text-muted border border-border',
    overnight: 'bg-[var(--signal-soft)] text-[var(--overnight)] border border-[var(--overnight)]/20',
    concerning: 'bg-[#fff7ed] text-[var(--concerning)] border border-[var(--concerning)]/20',
    critical: 'bg-[#fef2f2] text-[var(--critical)] border border-[var(--critical)]/20',
    verified: 'bg-[#ecfdf5] text-[var(--verified)] border border-[var(--verified)]/20',
    signal: 'bg-[var(--signal-soft)] text-[var(--signal)] border border-[var(--signal)]/20',
  };
  return <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${colors[variant]} ${className}`}>{children}</span>;
}
