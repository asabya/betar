import { Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Dashboard } from './pages/Dashboard';
import { Agents } from './pages/Agents';
import { Sessions } from './pages/Sessions';
import { Workflows } from './pages/Workflows';
import { Orders } from './pages/Orders';
import { WalletPage } from './pages/Wallet';
import { Peers } from './pages/Peers';

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/agents" element={<Agents />} />
        <Route path="/sessions" element={<Sessions />} />
        <Route path="/workflows" element={<Workflows />} />
        <Route path="/orders" element={<Orders />} />
        <Route path="/wallet" element={<WalletPage />} />
        <Route path="/peers" element={<Peers />} />
      </Route>
    </Routes>
  );
}
