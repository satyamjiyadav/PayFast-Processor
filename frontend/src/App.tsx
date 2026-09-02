import { useState } from 'react';
import CheckoutForm from './components/CheckoutForm';
import MerchantDashboard from './components/MerchantDashboard';
import { LayoutDashboard, ShoppingCart } from 'lucide-react';

function App() {
  const [activeTab, setActiveTab] = useState<'checkout' | 'dashboard'>('checkout');

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Global Navbar (Only show for Checkout Demo) */}
      {activeTab === 'checkout' && (
        <nav className="bg-white border-b border-gray-200 sticky top-0 z-10">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between h-16">
              <div className="flex items-center">
                <div className="flex-shrink-0 flex items-center gap-2">
                  <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
                    <span className="text-white font-bold text-xl">P</span>
                  </div>
                  <span className="font-bold text-xl text-gray-900 tracking-tight">PayFast</span>
                </div>
                <div className="ml-10 flex space-x-8">
                  <button
                    onClick={() => setActiveTab('checkout')}
                    className="inline-flex items-center gap-2 px-1 pt-1 border-b-2 text-sm font-medium transition-colors border-blue-500 text-blue-600"
                  >
                    <ShoppingCart size={18} /> Checkout Demo
                  </button>
                  <button
                    onClick={() => setActiveTab('dashboard')}
                    className="inline-flex items-center gap-2 px-1 pt-1 border-b-2 text-sm font-medium transition-colors border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
                  >
                    <LayoutDashboard size={18} /> Go to Merchant Dashboard
                  </button>
                </div>
              </div>
            </div>
          </div>
        </nav>
      )}

      {/* Main Content */}
      {activeTab === 'checkout' ? (
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 flex justify-center">
          <CheckoutForm />
        </main>
      ) : (
        <MerchantDashboard onBackToCheckout={() => setActiveTab('checkout')} />
      )}

      <style>{`
        @keyframes slide {
          0% { transform: translateX(-100%); }
          100% { transform: translateX(100%); }
        }
      `}</style>
    </div>
  );
}

export default App;
