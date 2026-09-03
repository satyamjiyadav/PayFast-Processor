import React, { useState } from 'react';
import axios from 'axios';
import { CreditCard, Smartphone, Building2, Loader2 } from 'lucide-react';

export default function CheckoutForm() {
  const [activeTab, setActiveTab] = useState<'card' | 'upi' | 'netbanking'>('card');
  const [pan, setPan] = useState('');
  const [expMonth, setExpMonth] = useState('12');
  const [expYear, setExpYear] = useState('2025');
  const [upiVpa, setUpiVpa] = useState('');
  const [bankCode, setBankCode] = useState('HDFC');
  
  const [amount, setAmount] = useState('2500'); // $25.00
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [isSubscription, setIsSubscription] = useState(false);
  
  // Stable Idempotency Key that only regenerates when form inputs change
  const [idemKey, setIdemKey] = useState(`idem_${Date.now()}`);

  React.useEffect(() => {
    setIdemKey(`idem_${Date.now()}`);
  }, [activeTab, amount, pan, upiVpa, bankCode, isSubscription]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setMessage('');

    try {
      // Step 1: Tokenize based on active tab
      let tokenizePayload: any = {
        merchant_id: 'merch_01test',
        type: activeTab
      };

      if (activeTab === 'card') {
        tokenizePayload.pan = pan;
        tokenizePayload.exp_month = parseInt(expMonth);
        tokenizePayload.exp_year = parseInt(expYear);
      } else if (activeTab === 'upi') {
        tokenizePayload.upi_vpa = upiVpa;
      } else if (activeTab === 'netbanking') {
        tokenizePayload.bank_code = bankCode;
      }

      const tokenRes = await axios.post('/v1/tokens', tokenizePayload);
      const tokenId = tokenRes.data.token_id;

      // Step 2: Pay or Subscribe
      if (isSubscription) {
        await axios.post('/v1/subscriptions', {
          merchant_id: 'merch_01test',
          customer_id: 'cust_01',
          payment_method_id: tokenId,
          plan_id: 'plan_pro',
          amount: parseInt(amount),
          currency: 'usd',
          interval: 'month'
        }, {
          headers: {
            'Authorization': 'Bearer sk_test_123',
            'Idempotency-Key': `sub_${idemKey}`
          }
        });
        setMessage('Success! Subscription started.');
      } else {
        const payRes = await axios.post('/v1/payments', {
          amount: parseInt(amount),
          currency: 'usd',
          token_id: tokenId
        }, {
          headers: {
            'Authorization': 'Bearer sk_test_123',
            'Idempotency-Key': `pay_${idemKey}`
          }
        });
        setMessage(`Success! Payment ${payRes.data.status} (ID: ${payRes.data.payment_id})`);
      }
    } catch (err: any) {
      setMessage(`Error: ${err.response?.data || err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white p-8 rounded-2xl shadow-xl max-w-md w-full border border-gray-100">
      <div className="flex items-center gap-3 mb-6">
        <div className="p-3 bg-blue-50 text-blue-600 rounded-xl">
          <CreditCard size={24} />
        </div>
        <h2 className="text-2xl font-semibold text-gray-800">Checkout</h2>
      </div>

      <div className="flex gap-2 mb-6">
        <button
          type="button"
          onClick={() => setActiveTab('card')}
          className={`flex-1 py-2 px-3 rounded-lg flex items-center justify-center gap-2 text-sm font-medium transition ${activeTab === 'card' ? 'bg-blue-600 text-white shadow-md' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}
        >
          <CreditCard size={16} /> Card
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('upi')}
          className={`flex-1 py-2 px-3 rounded-lg flex items-center justify-center gap-2 text-sm font-medium transition ${activeTab === 'upi' ? 'bg-blue-600 text-white shadow-md' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}
        >
          <Smartphone size={16} /> UPI
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('netbanking')}
          className={`flex-1 py-2 px-3 rounded-lg flex items-center justify-center gap-2 text-sm font-medium transition ${activeTab === 'netbanking' ? 'bg-blue-600 text-white shadow-md' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}
        >
          <Building2 size={16} /> Netbanking
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-5">
        
        {activeTab === 'card' && (
          <>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Card Number</label>
              <input
                type="text"
                required={activeTab === 'card'}
                className="w-full px-4 py-3 rounded-lg border border-gray-200 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition"
                placeholder="4111 1111 1111 1111"
                value={pan}
                onChange={e => setPan(e.target.value)}
              />
            </div>

            <div className="flex gap-4">
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-600 mb-1">Exp Month</label>
                <input
                  type="number"
                  min="1" max="12"
                  className="w-full px-4 py-3 rounded-lg border border-gray-200 focus:ring-2 focus:ring-blue-500 outline-none transition"
                  value={expMonth}
                  onChange={e => setExpMonth(e.target.value)}
                />
              </div>
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-600 mb-1">Exp Year</label>
                <input
                  type="number"
                  className="w-full px-4 py-3 rounded-lg border border-gray-200 focus:ring-2 focus:ring-blue-500 outline-none transition"
                  value={expYear}
                  onChange={e => setExpYear(e.target.value)}
                />
              </div>
            </div>
          </>
        )}

        {activeTab === 'upi' && (
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1">UPI ID (VPA)</label>
            <input
              type="text"
              required={activeTab === 'upi'}
              className="w-full px-4 py-3 rounded-lg border border-gray-200 focus:ring-2 focus:ring-blue-500 outline-none transition"
              placeholder="user@upi"
              value={upiVpa}
              onChange={e => setUpiVpa(e.target.value)}
            />
          </div>
        )}

        {activeTab === 'netbanking' && (
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1">Select Bank</label>
            <select
              required={activeTab === 'netbanking'}
              className="w-full px-4 py-3 rounded-lg border border-gray-200 focus:ring-2 focus:ring-blue-500 outline-none transition bg-white"
              value={bankCode}
              onChange={e => setBankCode(e.target.value)}
            >
              <option value="HDFC">HDFC Bank</option>
              <option value="SBI">State Bank of India</option>
              <option value="ICICI">ICICI Bank</option>
              <option value="AXIS">Axis Bank</option>
            </select>
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-600 mb-1">Amount (Cents)</label>
          <input
            type="number"
            className="w-full px-4 py-3 rounded-lg border border-gray-200 focus:ring-2 focus:ring-blue-500 outline-none transition"
            value={amount}
            onChange={e => setAmount(e.target.value)}
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="sub"
            checked={isSubscription}
            onChange={e => setIsSubscription(e.target.checked)}
            className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
          />
          <label htmlFor="sub" className="text-sm font-medium text-gray-600">
            Subscribe Monthly
          </label>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full py-3.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg shadow-md shadow-blue-500/20 transition flex items-center justify-center gap-2 disabled:opacity-70"
        >
          {loading ? <Loader2 className="animate-spin" size={20} /> : 'Pay Now'}
        </button>

        {message && (
          <div className={`p-4 rounded-lg text-sm font-medium ${message.includes('Error') ? 'bg-red-50 text-red-600' : 'bg-green-50 text-green-600'}`}>
            {message}
          </div>
        )}
      </form>
    </div>
  );
}
