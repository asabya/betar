import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Bot,
  MessageSquare,
  GitBranch,
  ShoppingCart,
  Wallet,
  Users,
} from 'lucide-react';

const links = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/agents', label: 'Agents', icon: Bot },
  { to: '/sessions', label: 'Sessions', icon: MessageSquare },
  { to: '/workflows', label: 'Workflows', icon: GitBranch },
  { to: '/orders', label: 'Orders', icon: ShoppingCart },
  { to: '/wallet', label: 'Wallet', icon: Wallet },
  { to: '/peers', label: 'Peers', icon: Users },
];

export function Sidebar() {
  return (
    <>
      {/* Desktop sidebar */}
      <nav aria-label="Main navigation" className="hidden lg:flex flex-col w-56 bg-[var(--color-surface)] backdrop-blur-xl border-r border-[var(--color-border)] p-4 gap-1 relative z-[2]">
        <div className="px-3 mb-6">
          <span className="text-xl font-bold tracking-tight" style={{ fontFamily: 'var(--font-display)' }}>
            be<span className="text-[var(--color-primary)]">tar</span>
          </span>
        </div>
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all duration-200 ${
                isActive
                  ? 'bg-[var(--color-primary-subtle)] text-[var(--color-primary-light)] border border-[var(--color-border-hover)]'
                  : 'text-[var(--color-text-muted)] hover:text-[var(--color-text)] hover:bg-[var(--color-surface-hover)] border border-transparent'
              }`
            }
          >
            <link.icon size={18} aria-hidden="true" />
            {link.label}
          </NavLink>
        ))}
      </nav>

      {/* Mobile bottom nav */}
      <nav aria-label="Mobile navigation" className="lg:hidden fixed bottom-0 left-0 right-0 bg-[var(--color-chrome)] backdrop-blur-xl border-t border-[var(--color-border)] flex justify-around py-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] z-50">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/'}
            className={({ isActive }) =>
              `flex flex-col items-center gap-0.5 text-xs transition-colors min-w-[44px] min-h-[44px] justify-center ${
                isActive ? 'text-[var(--color-primary)]' : 'text-[var(--color-text-dim)]'
              }`
            }
          >
            <link.icon size={20} aria-hidden="true" />
            <span>{link.label}</span>
          </NavLink>
        ))}
      </nav>
    </>
  );
}
