import type { Timestamp } from '@bufbuild/protobuf/wkt'

/**
 * Formats a protobuf Timestamp to relative time string (e.g., "5m ago", "2d ago")
 */
export const formatRelativeTime = (
  timestamp: Timestamp | undefined,
): string => {
  if (!timestamp) return '-'
  const date = new Date(Number(timestamp.seconds) * 1000)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return date.toLocaleDateString()
}

/**
 * Formats a token count to abbreviated form (e.g., "1.2M", "45K")
 */
export const formatTokenCount = (
  value: bigint | number | undefined,
): string => {
  if (value === undefined || value === null) return '0'
  const num = typeof value === 'bigint' ? Number(value) : value
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(0)}K`
  return num.toLocaleString()
}

/**
 * Truncates a string ID with ellipsis (e.g., "abc123...xyz9")
 */
export const truncateId = (id: string, maxLength = 12): string => {
  if (id.length <= maxLength) return id
  const half = Math.floor((maxLength - 3) / 2)
  return `${id.slice(0, half)}...${id.slice(-half)}`
}

/**
 * Truncates a file path, keeping the last N segments
 */
export const truncatePath = (path: string, maxLength = 40): string => {
  if (path.length <= maxLength) return path
  const parts = path.split('/')
  if (parts.length <= 3) return path
  return `.../${parts.slice(-2).join('/')}`
}
