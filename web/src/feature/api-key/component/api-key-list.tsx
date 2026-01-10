import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/gen/shadcn/ui/table'
import { Button } from '@/gen/shadcn/ui/button'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useRevokeAPIKey } from '../hook/use-revoke-api-key'
import type { APIKeyInfo } from '@/gen/grpcstub/apikey/v1/apikey_pb'

// APIKeyListProps defines the component props.
interface APIKeyListProps {
  // keys is the list of API keys to display
  keys: APIKeyInfo[]
  // isLoading indicates whether keys are being fetched
  isLoading: boolean
  // onKeyRevoked callback when a key is successfully revoked
  onKeyRevoked?: () => void
}

// formatDate formats a Unix timestamp (bigint) to a readable date string.
const formatDate = (timestamp: bigint): string => {
  if (timestamp === BigInt(0)) {
    return '-'
  }
  return new Date(Number(timestamp) * 1000).toLocaleDateString()
}

// APIKeyList displays a table of user's API keys with revoke functionality.
export const APIKeyList = ({
  keys,
  isLoading,
  onKeyRevoked,
}: APIKeyListProps) => {
  const revokeMutation = useRevokeAPIKey()

  const handleRevoke = (keyId: string) => {
    revokeMutation.mutate(
      { keyId },
      {
        onSuccess: () => {
          onKeyRevoked?.()
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  }

  if (keys.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <p className="text-muted-foreground">No API keys</p>
        <p className="mt-1 text-sm text-muted-foreground">
          Create an API key to get started
        </p>
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Key Prefix</TableHead>
          <TableHead>Created At</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {keys.map((key) => {
          const isRevoked = key.revokedAt !== BigInt(0)
          return (
            <TableRow key={key.id}>
              <TableCell className="font-medium">{key.name}</TableCell>
              <TableCell className="font-mono text-sm">
                {key.keyPrefix}...
              </TableCell>
              <TableCell>{formatDate(key.createdAt)}</TableCell>
              <TableCell>
                {isRevoked ? (
                  <Badge variant="secondary">Revoked</Badge>
                ) : (
                  <Badge variant="default">Active</Badge>
                )}
              </TableCell>
              <TableCell>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={isRevoked || revokeMutation.isPending}
                  onClick={() => handleRevoke(key.id)}
                >
                  Revoke
                </Button>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
