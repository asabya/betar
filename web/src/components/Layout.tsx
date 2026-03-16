import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { StatusBar } from './StatusBar';

export function Layout() {
  return (
    <div className="min-h-screen flex flex-col lg:flex-row">
      {/* Atmospheric orbs */}
      <div className="orb orb-1" aria-hidden="true" />
      <div className="orb orb-2" aria-hidden="true" />
      <div className="orb orb-3" aria-hidden="true" />

      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0 relative z-[1]">
        <StatusBar />
        <main className="flex-1 p-4 lg:p-6 overflow-auto pb-20 lg:pb-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
