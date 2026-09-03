import { useEffect, useState, useMemo } from 'react';
import axios from 'axios';
import { 
  Home, 
  ArrowLeftRight, 
  FileText, 
  Link as LinkIcon, 
  Search, 
  Bell, 
  Activity, 
  ChevronDown, 
  XCircle,
  HelpCircle,
  Menu,
  ShoppingCart
} from 'lucide-react';
import Reports from './Reports';

interface MerchantDashboardProps {
  onBackToCheckout: () => void;
}

export default function MerchantDashboard({ onBackToCheckout }: MerchantDashboardProps) {
  const [payments, setPayments] = useState<any[]>([]);
  const [settlements, setSettlements] = useState<any[]>([]);
  const [refunds, setRefunds] = useState<any[]>([]);
  const [activeTab, setActiveTab] = useState('orders');
  const [statusFilter, setStatusFilter] = useState('All');
  const [searchQuery, setSearchQuery] = useState('');

  const [payoutSchedule, setPayoutSchedule] = useState('instant');
  const [showSettings, setShowSettings] = useState(false);
  const [overviewFilter, setOverviewFilter] = useState('All time');

  const fetchData = async () => {
    try {
      const res = await axios.get('/v1/admin/payments');
      setPayments(res.data || []);
      const setRes = await axios.get('/v1/admin/settlements');
      setSettlements(setRes.data || []);
      const refRes = await axios.get('/v1/admin/refunds');
      setRefunds(refRes.data || []);
      const settingsRes = await axios.get('/v1/admin/merchant/settings');
      if (settingsRes.data && settingsRes.data.payout_schedule) {
        setPayoutSchedule(settingsRes.data.payout_schedule);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const updateSettings = async (schedule: string) => {
    try {
      await axios.put('/v1/admin/merchant/settings', { payout_schedule: schedule });
      setPayoutSchedule(schedule);
      setShowSettings(false);
    } catch (err) {
      console.error(err);
    }
  };

  const [refundTarget, setRefundTarget] = useState<{ id: string; amount: number } | null>(null);
  const [isRefunding, setIsRefunding] = useState(false);

  const handleConfirmRefund = async () => {
    if (!refundTarget) return;
    try {
      setIsRefunding(true);
      await axios.post('/v1/admin/refunds', { payment_id: refundTarget.id, amount: refundTarget.amount });
      setRefundTarget(null);
      fetchData();
    } catch (err) {
      console.error("Failed to issue refund:", err);
      alert("Failed to issue refund. Please try again.");
    } finally {
      setIsRefunding(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 3000);
    return () => clearInterval(interval);
  }, []);

  const overviewPayments = useMemo(() => {
    if (overviewFilter === 'Today') {
      const startOfDay = new Date();
      startOfDay.setHours(0, 0, 0, 0);
      return payments.filter(p => new Date(p.created_at) >= startOfDay);
    }
    return payments;
  }, [payments, overviewFilter]);

  const overviewRefunds = useMemo(() => {
    if (overviewFilter === 'Today') {
      const startOfDay = new Date();
      startOfDay.setHours(0, 0, 0, 0);
      return refunds.filter(r => new Date(r.created_at) >= startOfDay);
    }
    return refunds;
  }, [refunds, overviewFilter]);

  const totalRefundedAmount = useMemo(() => {
    return overviewRefunds.reduce((acc, curr) => acc + curr.amount, 0);
  }, [overviewRefunds]);

  const totalRevenue = useMemo(() => {
    return overviewPayments.filter(p => p.status === 'processed').reduce((acc, curr) => acc + curr.amount, 0);
  }, [overviewPayments]);

  const failedCount = useMemo(() => {
    return overviewPayments.filter(p => p.status === 'failed').length;
  }, [overviewPayments]);

  const filteredPayments = useMemo(() => {
    let filtered = payments;
    
    // Status Filter (Mapping local UI states to actual DB statuses)
    if (statusFilter === 'Created') filtered = filtered.filter(p => p.status === 'created');
    if (statusFilter === 'Initiated') filtered = filtered.filter(p => p.status === 'initiated');
    if (statusFilter === 'Processed') filtered = filtered.filter(p => p.status === 'processed');
    
    // Search Filter
    if (searchQuery) {
      filtered = filtered.filter(p => p.id.toLowerCase().includes(searchQuery.toLowerCase()));
    }
    
    // Sort by newest
    return filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }, [payments, statusFilter, searchQuery]);

  const filteredSettlements = useMemo(() => {
    let filtered = settlements;
    
    if (statusFilter === 'Created') filtered = filtered.filter(s => s.status === 'created');
    if (statusFilter === 'Initiated') filtered = filtered.filter(s => s.status === 'initiated');
    if (statusFilter === 'Processed') filtered = filtered.filter(s => s.status === 'processed');
    
    if (searchQuery) {
      filtered = filtered.filter(s => s.id.toLowerCase().includes(searchQuery.toLowerCase()));
    }
    
    return filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }, [settlements, statusFilter, searchQuery]);

  const filteredRefunds = useMemo(() => {
    let filtered = refunds;
    
    if (statusFilter === 'Created') filtered = filtered.filter(r => r.status === 'created');
    if (statusFilter === 'Initiated') filtered = filtered.filter(r => r.status === 'initiated');
    if (statusFilter === 'Processed') filtered = filtered.filter(r => r.status === 'processed');
    
    if (searchQuery) {
      filtered = filtered.filter(r => r.id.toLowerCase().includes(searchQuery.toLowerCase()) || r.payment_id.toLowerCase().includes(searchQuery.toLowerCase()));
    }
    
    return filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }, [refunds, statusFilter, searchQuery]);

  return (
    <div className="flex h-screen w-full bg-[#f4f5f6] overflow-hidden text-sm animate-fade-in">
      
      {/* Left Sidebar */}
      <aside className="w-64 bg-[#f8f9fa] border-r border-gray-200 flex flex-col hidden md:flex shrink-0">
        <div className="h-16 flex items-center px-6 cursor-pointer" onClick={onBackToCheckout}>
          <div className="w-8 h-8 bg-blue-600 rounded flex items-center justify-center mr-3">
            <span className="text-white font-bold text-lg">P</span>
          </div>
          <span className="font-bold text-xl tracking-tight text-gray-800">PayFast</span>
        </div>
        
        <div className="flex-1 overflow-y-auto py-4">
          <nav className="space-y-1">
            <SidebarItem icon={<Home size={18} />} label="Home" />
            <SidebarItem icon={<ArrowLeftRight size={18} />} label="Transactions" active={activeTab === 'payments' || activeTab === 'orders'} onClick={() => setActiveTab('orders')} />
            <SidebarItem icon={<Activity size={18} />} label="Settlements" active={activeTab === 'settlements'} onClick={() => setActiveTab('settlements')} />
            <SidebarItem icon={<ArrowLeftRight size={18} />} label="Refunds" active={activeTab === 'refunds'} onClick={() => setActiveTab('refunds')} />
            <SidebarItem icon={<FileText size={18} />} label="Reports" active={activeTab === 'reports'} onClick={() => setActiveTab('reports')} />
            
            <div className="pt-6 pb-2 px-6">
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Payment Products</p>
            </div>
            
            <SidebarItem icon={<LinkIcon size={18} />} label="Payment Links" badge="New Update" />
            <SidebarItem icon={<FileText size={18} />} label="Payment Pages" />
            
            <div className="pt-6 pb-2 px-6">
               <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Demo Tools</p>
            </div>
            <SidebarItem icon={<ShoppingCart size={18} />} label="Back to Checkout" onClick={onBackToCheckout} />
          </nav>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col h-screen overflow-hidden">
        
        {/* Top Navbar */}
        <header className="h-16 bg-[#1e2330] flex items-center justify-between px-6 shrink-0 text-white">
          <div className="flex items-center gap-6">
             <div className="flex items-center gap-2 md:hidden">
               <Menu size={24} cursor="pointer" onClick={onBackToCheckout} />
             </div>
             <nav className="hidden md:flex items-center gap-6 text-sm font-medium text-slate-300">
               <a href="#" className="hover:text-white transition-colors">Home</a>
               <a href="#" className="text-white border-b-2 border-blue-500 pb-1">Payments</a>
               <a href="#" className="hover:text-white transition-colors">Banking+</a>
               <a href="#" className="hover:text-white transition-colors">Payroll</a>
             </nav>
          </div>

          <div className="flex items-center gap-4">
             <div className="flex items-center bg-[#2d3240] rounded-full p-1 border border-slate-600">
               <div className="bg-emerald-500 w-3 h-3 rounded-full ml-2 mr-2"></div>
               <span className="text-xs font-bold mr-3 tracking-wider">TEST</span>
             </div>
             <div className="relative hidden lg:block">
               <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
               <input 
                 type="text" 
                 placeholder="Search payment products..." 
                 className="bg-[#2d3240] border border-slate-600 rounded text-sm text-white pl-9 pr-4 py-1.5 focus:outline-none focus:border-blue-500 w-64"
               />
             </div>
             <Activity size={20} className="text-slate-400 hover:text-white cursor-pointer hidden sm:block" />
             <Bell size={20} className="text-slate-400 hover:text-white cursor-pointer hidden sm:block" />
             <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center font-semibold cursor-pointer shrink-0">
               SY
             </div>
          </div>
        </header>

        {/* Dashboard Content */}
        <div className="flex-1 overflow-y-auto bg-gray-50/30">
           {activeTab === 'reports' ? (
             <Reports />
           ) : (
             <div className="p-4 md:p-6 lg:px-8">
               {/* Header & Documentation */}
               <div className="flex items-center justify-between mb-4">
                 <h1 className="text-lg font-semibold text-gray-800 flex items-center gap-2">
                   Overview 
                   <div className="relative inline-block group">
                     <span className="text-blue-600 cursor-pointer text-sm flex items-center py-2">{overviewFilter} <ChevronDown size={14} className="ml-1" /></span>
                     <div className="absolute left-0 top-full mt-0 w-32 bg-white rounded-md shadow-lg border border-gray-100 hidden group-hover:block z-10">
                       <div className="py-1">
                         <button onClick={() => setOverviewFilter('Today')} className="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">Today</button>
                         <button onClick={() => setOverviewFilter('All time')} className="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">All time</button>
                       </div>
                     </div>
                   </div>
                 </h1>
                 <a href="#" className="text-blue-600 hover:underline flex items-center gap-1 font-medium hidden sm:flex">
                    Documentation
                 </a>
               </div>

           {/* Metrics Grid */}
           <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
             {/* Collected Amount (Large) */}
             <div className="sm:col-span-2 bg-white rounded-lg border border-gray-200 p-5 shadow-sm">
                <p className="text-sm text-gray-500 font-medium flex items-center gap-1 mb-2">Collected Amount <HelpCircle size={14} /></p>
                <h2 className="text-3xl md:text-4xl font-light text-gray-900 mb-2">${(totalRevenue / 100).toFixed(2)}</h2>
                <p className="text-xs text-gray-500">from {overviewPayments.filter(p => p.status === 'processed').length} processed payments</p>
             </div>

             {/* Refunds */}
             <div className="bg-white rounded-lg border border-gray-200 p-5 shadow-sm flex flex-col justify-between">
                <div className="flex justify-between items-start">
                  <p className="text-sm text-gray-500 font-medium flex items-center gap-1">
                    <ArrowLeftRight size={16} className="text-blue-500" /> Refunds <HelpCircle size={14} />
                  </p>
                </div>
                <div className="mt-4 sm:mt-0">
                  <h3 className="text-2xl font-light text-gray-900 mb-1">${(totalRefundedAmount / 100).toFixed(2)}</h3>
                  <p className="text-xs text-gray-500">from {overviewRefunds.length} refunds</p>
                </div>
             </div>

             {/* Failed */}
             <div className="bg-white rounded-lg border border-gray-200 p-5 shadow-sm flex flex-col justify-between">
                <div className="flex justify-between items-start">
                  <p className="text-sm text-gray-500 font-medium flex items-center gap-1">
                    <XCircle size={16} className="text-red-500" /> Failed <HelpCircle size={14} />
                  </p>
                </div>
                <div className="mt-4 sm:mt-0">
                  <h3 className="text-2xl font-light text-gray-900 mb-1">{failedCount}</h3>
                  <p className="text-xs text-gray-500">payments</p>
                </div>
             </div>
           </div>

           {/* Tabs */}
           <div className="border-b border-gray-200 mb-6 flex gap-6">
              <button 
                onClick={() => setActiveTab('payments')}
                className={`pb-3 font-medium text-sm transition-colors ${activeTab === 'payments' ? 'border-b-2 border-blue-600 text-gray-900' : 'text-gray-500 hover:text-gray-700'}`}
              >
                Payments
              </button>
              <button 
                onClick={() => setActiveTab('orders')}
                className={`pb-3 font-medium text-sm transition-colors ${activeTab === 'orders' ? 'border-b-2 border-blue-600 text-gray-900' : 'text-gray-500 hover:text-gray-700'}`}
              >
                Orders
              </button>
              <button 
                onClick={() => setActiveTab('settlements')}
                className={`pb-3 font-medium text-sm transition-colors ${activeTab === 'settlements' ? 'border-b-2 border-blue-600 text-gray-900' : 'text-gray-500 hover:text-gray-700'}`}
              >
                Settlements
              </button>
              <button 
                onClick={() => setActiveTab('refunds')}
                className={`pb-3 font-medium text-sm transition-colors ${activeTab === 'refunds' ? 'border-b-2 border-blue-600 text-gray-900' : 'text-gray-500 hover:text-gray-700'}`}
              >
                Refunds
              </button>
           </div>

           {/* Data Table Area */}
           <div className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden flex flex-col">
              
              {/* Toolbar */}
              <div className="p-4 border-b border-gray-100 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                 
                 {/* Status Filters */}
                 <div className="flex items-center gap-2 overflow-x-auto w-full sm:w-auto pb-2 sm:pb-0 hide-scrollbar">
                    {['All', 'Created', 'Initiated', 'Processed'].map(status => (
                      <button 
                        key={status}
                        onClick={() => setStatusFilter(status)}
                        className={`px-4 py-1.5 rounded-full text-xs font-semibold whitespace-nowrap transition-colors ${
                          statusFilter === status 
                            ? 'bg-gray-800 text-white' 
                            : 'text-gray-600 bg-gray-100 hover:bg-gray-200'
                        }`}
                      >
                        {status}
                      </button>
                    ))}
                 </div>

                 {/* Search */}
                 <div className="relative w-full sm:w-64">
                   <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                   <input 
                     type="text" 
                     placeholder="Search in Order Id" 
                     value={searchQuery}
                     onChange={(e) => setSearchQuery(e.target.value)}
                     className="w-full border border-gray-300 rounded text-sm pl-8 pr-2 py-1.5 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                   />
                 </div>
              </div>

              {/* Status Dropdown (Extra Filter Row) */}
              <div className="px-4 py-3 border-b border-gray-200 bg-gray-50 flex justify-between items-center">
                 <button className="flex items-center gap-2 text-xs font-semibold text-gray-700 bg-white border border-gray-300 px-3 py-1.5 rounded shadow-sm hover:bg-gray-50 transition-colors">
                    Status <ChevronDown size={14} />
                 </button>
                 <button 
                    onClick={() => setShowSettings(true)}
                    className="ml-auto inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition-colors border border-blue-200 shadow-sm"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                    Settlement Schedule
                  </button>
              </div>

              {/* Table */}
              {(activeTab === 'payments' || activeTab === 'orders') && (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-gray-200 bg-white">
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Order Id</th>
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Amount</th>
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Settlement ID</th>
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Attempts</th>
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Created At</th>
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider text-right">Status</th>
                      <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {filteredPayments.map(p => (
                      <tr key={p.id} className="hover:bg-blue-50/50 transition-colors cursor-pointer group">
                        <td className="px-6 py-4">
                          <span className="font-mono text-blue-600 font-medium group-hover:underline">{p.id}</span>
                        </td>
                        <td className="px-6 py-4 text-gray-900 font-medium">
                          ${(p.amount / 100).toFixed(2)}
                        </td>
                        <td className="px-6 py-4 text-gray-500 font-mono text-xs">
                          {p.settlement_id ? p.settlement_id : (p.status === 'initiated' ? 'Pending Settlement' : '-')}
                        </td>
                        <td className="px-6 py-4 text-gray-500">
                          {p.status === 'created' ? 0 : 1}
                        </td>
                        <td className="px-6 py-4 text-gray-500 text-xs whitespace-nowrap">
                           {new Date(p.created_at).toLocaleString('en-US', {
                             month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'
                           })}
                        </td>
                        <td className="px-6 py-4 text-right">
                          <StatusBadge status={p.status} />
                        </td>
                        <td className="px-6 py-4 text-right">
                          {p.status === 'processed' && (
                            <button
                              onClick={(e) => { e.stopPropagation(); setRefundTarget({ id: p.id, amount: p.amount }); }}
                              className="text-xs font-medium text-blue-600 hover:text-blue-800 bg-blue-50 px-3 py-1.5 rounded border border-blue-200 transition-colors"
                            >
                              Refund
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                    {filteredPayments.length === 0 && (
                      <tr>
                        <td colSpan={6} className="px-6 py-16 text-center text-gray-400">
                          <div className="flex flex-col items-center gap-3">
                            <Activity size={40} className="opacity-20 text-gray-400" />
                            <p className="font-medium text-gray-500">No orders found matching your criteria.</p>
                          </div>
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
              )}
              
              {activeTab === 'settlements' && (
                <div className="overflow-x-auto border-t border-gray-200">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="border-b border-gray-200 bg-white">
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Settlement Id</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Amount</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Created At</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider text-right">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {filteredSettlements.map(s => (
                        <tr key={s.id} className="hover:bg-blue-50/50 transition-colors cursor-pointer group">
                          <td className="px-6 py-4">
                            <span className="font-mono text-blue-600 font-medium group-hover:underline">{s.id}</span>
                          </td>
                          <td className="px-6 py-4 text-gray-900 font-medium">
                            ${(s.amount / 100).toFixed(2)}
                          </td>
                          <td className="px-6 py-4 text-gray-500 text-xs whitespace-nowrap">
                             {new Date(s.created_at).toLocaleString('en-US', {
                               month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'
                             })}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <StatusBadge status={s.status} />
                          </td>
                        </tr>
                      ))}
                      {filteredSettlements.length === 0 && (
                        <tr>
                          <td colSpan={4} className="px-6 py-16 text-center text-gray-400">
                            <div className="flex flex-col items-center gap-3">
                              <Activity size={40} className="opacity-20 text-gray-400" />
                              <p className="font-medium text-gray-500">No settlements found.</p>
                            </div>
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}

              {activeTab === 'refunds' && (
                <div className="overflow-x-auto border-t border-gray-200">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="border-b border-gray-200 bg-white">
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Refund Id</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Payment Id</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Amount</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider">Created At</th>
                        <th className="px-6 py-4 text-xs font-bold text-gray-500 uppercase tracking-wider text-right">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {filteredRefunds.map(r => (
                        <tr key={r.id} className="hover:bg-blue-50/50 transition-colors cursor-pointer group">
                          <td className="px-6 py-4">
                            <span className="font-mono text-blue-600 font-medium group-hover:underline">{r.id}</span>
                          </td>
                          <td className="px-6 py-4 text-gray-500 font-mono text-xs">
                            {r.payment_id}
                          </td>
                          <td className="px-6 py-4 text-gray-900 font-medium">
                            ${(r.amount / 100).toFixed(2)}
                          </td>
                          <td className="px-6 py-4 text-gray-500 text-xs whitespace-nowrap">
                             {new Date(r.created_at).toLocaleString('en-US', {
                               month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'
                             })}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <StatusBadge status={r.status} />
                          </td>
                        </tr>
                      ))}
                      {filteredRefunds.length === 0 && (
                        <tr>
                          <td colSpan={5} className="px-6 py-16 text-center text-gray-400">
                            <p className="font-medium text-gray-500">No refunds have been issued yet.</p>
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}
           </div>
           
           {/* Footer Padding */}
           <div className="h-12"></div>
          </div>
        )}
        </div>
      </main>

      {/* Settings Modal */}
      {showSettings && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-lg shadow-xl w-[400px] overflow-hidden">
             <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
                <h3 className="font-bold text-gray-800">Settlement Schedule</h3>
                <button onClick={() => setShowSettings(false)} className="text-gray-400 hover:text-gray-600">
                  <XCircle size={20} />
                </button>
             </div>
             <div className="p-6">
                <label className="block text-sm font-medium text-gray-700 mb-2">Settlement Schedule</label>
                <div className="space-y-3">
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="radio" name="payout" value="instant" checked={payoutSchedule === 'instant'} onChange={() => setPayoutSchedule('instant')} className="text-blue-600" />
                    <span className="text-sm text-gray-800">Instant (real-time)</span>
                  </label>
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="radio" name="payout" value="1_hour" checked={payoutSchedule === '1_hour'} onChange={() => setPayoutSchedule('1_hour')} className="text-blue-600" />
                    <span className="text-sm text-gray-800">T+1 Hour</span>
                  </label>
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="radio" name="payout" value="12_hours" checked={payoutSchedule === '12_hours'} onChange={() => setPayoutSchedule('12_hours')} className="text-blue-600" />
                    <span className="text-sm text-gray-800">T+12 Hours</span>
                  </label>
                  <label className="flex items-center gap-3 cursor-pointer">
                    <input type="radio" name="payout" value="24_hours" checked={payoutSchedule === '24_hours'} onChange={() => setPayoutSchedule('24_hours')} className="text-blue-600" />
                    <span className="text-sm text-gray-800">T+24 Hours (Next Day)</span>
                  </label>
                </div>
             </div>
             <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end gap-3">
                <button onClick={() => setShowSettings(false)} className="px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-800">Cancel</button>
                <button onClick={() => updateSettings(payoutSchedule)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm font-medium hover:bg-blue-700 shadow-sm">Save Changes</button>
             </div>
          </div>
        </div>
      )}

      {refundTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="bg-white rounded-xl shadow-2xl max-w-md w-full p-6 border border-gray-100 animate-in fade-in zoom-in duration-150">
            <h3 className="text-lg font-bold text-gray-900 mb-2">Confirm Refund</h3>
            <p className="text-sm text-gray-600 mb-4">
              Are you sure you want to issue a full refund of{' '}
              <span className="font-semibold text-gray-900">${(refundTarget.amount / 100).toFixed(2)}</span> for payment{' '}
              <span className="font-mono text-xs bg-gray-100 px-1.5 py-0.5 rounded text-blue-600">{refundTarget.id}</span>?
            </p>
            <div className="flex items-center justify-end gap-3 mt-6">
              <button
                type="button"
                disabled={isRefunding}
                onClick={() => setRefundTarget(null)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={isRefunding}
                onClick={handleConfirmRefund}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg transition-colors shadow-sm disabled:opacity-50"
              >
                {isRefunding ? 'Refunding...' : 'Confirm Refund'}
              </button>
            </div>
          </div>
        </div>
      )}
      
      <style>{`
        .hide-scrollbar::-webkit-scrollbar {
            display: none;
        }
        .hide-scrollbar {
            -ms-overflow-style: none;
            scrollbar-width: none;
        }
      `}</style>
    </div>
  );
}

function SidebarItem({ icon, label, active = false, badge, onClick }: { icon: React.ReactNode, label: string, active?: boolean, badge?: string, onClick?: () => void }) {
  return (
    <div 
      onClick={onClick}
      className={`flex items-center justify-between px-6 py-3 cursor-pointer transition-colors ${
        active ? 'bg-blue-50 border-r-4 border-blue-600 text-blue-700 font-semibold' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 font-medium'
      }`}
    >
      <div className="flex items-center gap-3">
        <span className={`${active ? 'text-blue-600' : 'text-gray-500'}`}>{icon}</span>
        <span>{label}</span>
      </div>
      {badge && (
        <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-100 text-emerald-800 uppercase tracking-wide">
          {badge}
        </span>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const isProcessed = status === 'processed';
  const isFailed = status === 'failed';
  const isInitiated = status === 'initiated';
  const isRefunded = status === 'refunded';
  
  let colorClass = 'bg-gray-100 text-gray-600 border-gray-200'; // created
  if (isProcessed) colorClass = 'bg-blue-50 text-blue-700 border-blue-200';
  if (isFailed) colorClass = 'bg-red-50 text-red-700 border-red-200';
  if (isInitiated) colorClass = 'bg-orange-50 text-orange-700 border-orange-200';
  if (isRefunded) colorClass = 'bg-purple-50 text-purple-700 border-purple-200';

  return (
    <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-bold uppercase tracking-wider border ${colorClass}`}>
      {status}
    </span>
  );
}
