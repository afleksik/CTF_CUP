import { Flame, Wine, Beer, Droplet, Leaf } from 'lucide-react';
import type { AssetType } from '../types';

interface AssetTypeBadgeProps {
  type: AssetType;
}

export function AssetTypeBadge({ type }: AssetTypeBadgeProps) {
  const config = {
    spirits: { icon: Flame, label: 'Spirits', color: 'bg-red-100 text-red-800' },
    wine: { icon: Wine, label: 'Wine', color: 'bg-purple-100 text-purple-800' },
    beer: { icon: Beer, label: 'Beer', color: 'bg-amber-100 text-amber-800' },
    mixers: { icon: Droplet, label: 'Mixers', color: 'bg-blue-100 text-blue-800' },
    garnishes: { icon: Leaf, label: 'Garnishes', color: 'bg-green-100 text-green-800' },
  };

  const { icon: Icon, label, color } = config[type];

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${color}`}>
      <Icon className="h-3 w-3" />
      {label}
    </span>
  );
}