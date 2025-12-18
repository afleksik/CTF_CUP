import { AlertCircle, Globe, Hash, Link as LinkIcon, FileText } from 'lucide-react';
import type { Indicator } from '../types';

interface IOCListProps {
  iocs: Indicator[];
}

const severityColors = {
  low: 'bg-blue-100 text-blue-800 border-blue-200',
  medium: 'bg-yellow-100 text-yellow-800 border-yellow-200',
  high: 'bg-orange-100 text-orange-800 border-orange-200',
  critical: 'bg-red-100 text-red-800 border-red-200',
};

const typeIcons = {
  ip: Globe,
  domain: Globe,
  hash: Hash,
  url: LinkIcon,
  default: FileText,
};

export function IOCList({ iocs }: IOCListProps) {
  if (iocs.length === 0) {
    return (
      <div className="text-center py-12 bg-gray-50 rounded-lg">
        <AlertCircle className="h-12 w-12 text-gray-400 mx-auto mb-4" />
        <p className="text-gray-600">No IOCs found in this feed</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {iocs.map((ioc) => {
        const Icon = typeIcons[ioc.type as keyof typeof typeIcons] || typeIcons.default;

        return (
          <div
            key={ioc.id}
            className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow"
          >
            <div className="flex items-start gap-4">
              <div className="p-2 bg-gray-100 rounded-lg">
                <Icon className="h-5 w-5 text-gray-600" />
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div className="flex-1 min-w-0">
                    <code className="text-sm font-mono text-gray-900 break-all">
                      {ioc.value}
                    </code>
                  </div>
                  <span
                    className={`px-3 py-1 text-xs font-semibold rounded-full border flex-shrink-0 ${
                      severityColors[ioc.severity]
                    }`}
                  >
                    {ioc.severity.toUpperCase()}
                  </span>
                </div>

                {ioc.description && (
                  <p className="text-sm text-gray-600 mb-2">{ioc.description}</p>
                )}

                <div className="flex items-center gap-4 text-xs text-gray-500">
                  <span className="px-2 py-1 bg-gray-100 rounded">
                    Type: {ioc.type}
                  </span>
                  <span>
                    Added {new Date(ioc.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
