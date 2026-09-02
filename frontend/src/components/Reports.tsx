import React, { useState } from 'react';
import { Download, FileText, ChevronDown } from 'lucide-react';

const Reports: React.FC = () => {
  const [timeRange, setTimeRange] = useState('24h');
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownload = async () => {
    try {
      setIsDownloading(true);
      const response = await fetch(`http://localhost:80/v1/admin/reports/settlements?range=${timeRange}`);
      if (!response.ok) {
        throw new Error('Failed to fetch report');
      }
      
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `settlements_report_${timeRange}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Error downloading report:', error);
      alert('Failed to download report. Please try again.');
    } finally {
      setIsDownloading(false);
    }
  };

  return (
    <div className="p-8 max-w-6xl mx-auto w-full">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900 mb-1">Reports</h1>
          <p className="text-sm text-gray-500">Generate and download financial reports for your account.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Settlements Report Card */}
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden shadow-sm hover:shadow-md transition-shadow">
          <div className="p-6 border-b border-gray-100 flex items-start gap-4">
            <div className="w-12 h-12 rounded-lg bg-blue-50 flex items-center justify-center flex-shrink-0">
              <FileText className="text-blue-600" size={24} />
            </div>
            <div>
              <h3 className="font-semibold text-gray-900 mb-1">Settlements Report</h3>
              <p className="text-sm text-gray-500">Detailed breakdown of all your settlements including amounts, status, and timestamps.</p>
            </div>
          </div>
          
          <div className="p-6 bg-gray-50/50">
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">Time Range</label>
              <div className="relative">
                <select 
                  value={timeRange}
                  onChange={(e) => setTimeRange(e.target.value)}
                  className="w-full appearance-none bg-white border border-gray-300 text-gray-700 py-2.5 px-4 pr-10 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                >
                  <option value="24h">Past 24 Hours</option>
                  <option value="48h">Past 48 Hours</option>
                  <option value="7d">Past 7 Days</option>
                  <option value="30d">Past 30 Days</option>
                </select>
                <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-gray-500">
                  <ChevronDown size={16} />
                </div>
              </div>
            </div>

            <button
              onClick={handleDownload}
              disabled={isDownloading}
              className={`w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-lg font-medium text-sm transition-colors
                ${isDownloading 
                  ? 'bg-blue-400 cursor-not-allowed text-white' 
                  : 'bg-blue-600 hover:bg-blue-700 text-white shadow-sm'}`}
            >
              <Download size={18} />
              {isDownloading ? 'Generating...' : 'Download CSV Report'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Reports;
