import { Key } from 'lucide-react'
import { useListAPIKeys } from '../hook/use-list-api-keys'
import { APIKeyList } from './api-key-list'
import { IssueAPIKeyDialog } from './issue-api-key-dialog'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'

// APIKeySettings provides a complete API key management section.
export const APIKeySettings = () => {
  const { data, isLoading, refetch } = useListAPIKeys()

  const handleKeyIssued = async () => {
    await refetch()
  }

  const handleKeyRevoked = async () => {
    await refetch()
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Key className="h-6 w-6 text-violet-400" />
            <div>
              <CardTitle>API Keys</CardTitle>
              <CardDescription>
                Manage API keys for programmatic access
              </CardDescription>
            </div>
          </div>
          <IssueAPIKeyDialog onKeyIssued={handleKeyIssued} />
        </div>
      </CardHeader>
      <CardContent>
        <APIKeyList
          keys={data?.keys ?? []}
          isLoading={isLoading}
          onKeyRevoked={handleKeyRevoked}
        />
      </CardContent>
    </Card>
  )
}
