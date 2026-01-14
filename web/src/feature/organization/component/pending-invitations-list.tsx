import { useCallback, useState } from 'react'
import { Loader2, Mail, Trash2, X } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useListInvitations } from '../hook/use-list-invitations'
import { useRevokeInvitation } from '../hook/use-revoke-invitation'
import { Button } from '@/gen/shadcn/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/gen/shadcn/ui/alert-dialog'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

interface PendingInvitationsListProps {
  // organizationId is the ID of the organization
  organizationId: string
  // isAdmin indicates if the current user is an admin
  isAdmin: boolean
}

// formatDate formats a date string for display.
const formatDate = (dateString: string): string => {
  try {
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
  } catch {
    return dateString
  }
}

export const PendingInvitationsList = ({
  organizationId,
  isAdmin,
}: PendingInvitationsListProps) => {
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false)
  const [selectedInvitationId, setSelectedInvitationId] = useState<string | null>(null)
  const [selectedEmail, setSelectedEmail] = useState<string>('')
  const [revokeError, setRevokeError] = useState<string | null>(null)

  const { data, isLoading, error, refetch } = useListInvitations({ organizationId })
  const revokeMutation = useRevokeInvitation()

  const handleRevokeClick = useCallback((invitationId: string, email: string) => {
    setSelectedInvitationId(invitationId)
    setSelectedEmail(email)
    setRevokeError(null)
    setRevokeDialogOpen(true)
  }, [])

  const handleRevokeConfirm = useCallback(async () => {
    if (!selectedInvitationId) return

    try {
      await revokeMutation.mutateAsync({ invitationId: selectedInvitationId })
      setRevokeDialogOpen(false)
      setSelectedInvitationId(null)
      setSelectedEmail('')
      refetch()
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'Failed to revoke invitation'

      if (connectError.code === Code.PermissionDenied) {
        errorMessage = 'You must be an admin to revoke invitations'
      } else if (connectError.code === Code.NotFound) {
        errorMessage = 'Invitation not found'
      } else if (connectError.code === Code.FailedPrecondition) {
        errorMessage = 'This invitation has already been processed'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setRevokeError(errorMessage)
    }
  }, [selectedInvitationId, revokeMutation, refetch])

  // Only show for admins
  if (!isAdmin) {
    return null
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="h-5 w-5 animate-spin text-zinc-500" />
        <span className="ml-2 text-sm text-zinc-500">Loading invitations...</span>
      </div>
    )
  }

  if (error) {
    return (
      <Alert className="border-red-900/50 bg-red-950/20">
        <AlertDescription className="text-sm text-red-200">
          Failed to load invitations
        </AlertDescription>
      </Alert>
    )
  }

  const invitations = data?.invitations ?? []

  if (invitations.length === 0) {
    return (
      <div className="flex items-center gap-2 py-3 text-sm text-zinc-500">
        <Mail className="h-4 w-4" />
        <span>No pending invitations</span>
      </div>
    )
  }

  return (
    <>
      <div className="space-y-2">
        {invitations.map((invitation) => (
          <div
            key={invitation.id}
            className="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-3"
          >
            <div className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-violet-500/20">
                <Mail className="h-4 w-4 text-violet-400" />
              </div>
              <div>
                <p className="text-sm font-medium">{invitation.email}</p>
                <p className="text-xs text-zinc-500">
                  Invited {formatDate(invitation.createdAt)}
                </p>
              </div>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleRevokeClick(invitation.id, invitation.email)}
              disabled={revokeMutation.isPending}
              className="text-zinc-400 hover:text-red-400"
            >
              {revokeMutation.isPending &&
              selectedInvitationId === invitation.id ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <X className="h-4 w-4" />
              )}
              <span className="ml-1 sr-only">Revoke</span>
            </Button>
          </div>
        ))}
      </div>

      <AlertDialog open={revokeDialogOpen} onOpenChange={setRevokeDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke Invitation</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to revoke the invitation sent to{' '}
              <strong>{selectedEmail}</strong>? They will no longer be able to
              join this organization using this invitation.
            </AlertDialogDescription>
          </AlertDialogHeader>

          {revokeError && (
            <Alert className="border-red-900/50 bg-red-950/20">
              <AlertDescription className="text-sm text-red-200">
                {revokeError}
              </AlertDescription>
            </Alert>
          )}

          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokeMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleRevokeConfirm}
              disabled={revokeMutation.isPending}
              className="bg-red-600 hover:bg-red-700"
            >
              {revokeMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              <Trash2 className="mr-2 h-4 w-4" />
              Revoke
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
