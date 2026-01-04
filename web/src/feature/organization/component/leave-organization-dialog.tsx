import { useState, useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { AlertTriangle, Loader2, LogOut } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useLeaveOrganization } from '../hook/use-leave-organization'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/gen/shadcn/ui/dialog'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

// LeaveOrganizationDialogState represents the dialog's internal state.
type LeaveOrganizationDialogState =
  | { status: 'idle' }
  | { status: 'confirming' }
  | { status: 'error'; message: string }

interface LeaveOrganizationDialogProps {
  // organizationId is the ID of the organization to leave
  organizationId: string
  // organizationName is the name of the organization (for display)
  organizationName: string
  // isLastOrganization indicates if this is the user's last organization
  isLastOrganization: boolean
  // isSoleMember indicates if the user is the only member
  isSoleMember: boolean
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
}

export const LeaveOrganizationDialog = ({
  organizationId,
  organizationName,
  isLastOrganization,
  isSoleMember,
  trigger,
}: LeaveOrganizationDialogProps) => {
  const [isOpen, setIsOpen] = useState(false)
  const [confirmationInput, setConfirmationInput] = useState('')
  const [state, setState] = useState<LeaveOrganizationDialogState>({
    status: 'idle',
  })

  const mutation = useLeaveOrganization()
  const { organizations, setOrganizations, setSelectedOrganizationId } = useUserStore()
  const navigate = useNavigate()

  const handleLeave = useCallback(async () => {
    setState({ status: 'confirming' })
    try {
      const result = await mutation.mutateAsync({ organizationId })

      if (result.isLastOrganization) {
        // Clear user store organizations
        setOrganizations([])
        setSelectedOrganizationId(null)
        // Navigate to home or create organization page
        navigate({ to: '/' })
      } else {
        // Update organizations in store (remove this org)
        const newOrganizations = organizations.filter((org) => org.id !== organizationId)
        setOrganizations(newOrganizations)
        // Select first remaining organization
        if (newOrganizations.length > 0) {
          setSelectedOrganizationId(newOrganizations[0].id)
        }
      }

      setIsOpen(false)
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'An error occurred'

      if (connectError.code === Code.FailedPrecondition) {
        errorMessage = 'You cannot leave as the sole admin with other members'
      } else if (connectError.code === Code.PermissionDenied) {
        errorMessage = 'You are not a member of this organization'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setState({ status: 'error', message: errorMessage })
    }
  }, [mutation, organizationId, organizations, setOrganizations, setSelectedOrganizationId, navigate])

  const isConfirmationRequired = isLastOrganization && isSoleMember
  const isConfirmationValid = isConfirmationRequired ? confirmationInput === 'LEAVE' : true

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-6 w-6 text-red-400" />
            <DialogTitle>Leave Organization</DialogTitle>
          </div>
          <DialogDescription>
            {isLastOrganization && isSoleMember
              ? 'This is your last organization and you are the only member. Leaving will permanently delete all data.'
              : 'Are you sure you want to leave this organization?'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {isLastOrganization && isSoleMember && (
            <Alert className="border-red-900/50 bg-red-950/20">
              <AlertDescription className="text-sm text-red-200">
                <p className="font-semibold mb-2">This will permanently delete:</p>
                <ul className="list-disc list-inside space-y-1 ml-2">
                  <li>All projects in this organization</li>
                  <li>All session records and data</li>
                  <li>The organization itself</li>
                </ul>
                <p className="mt-3">
                  Type <strong>LEAVE</strong> to confirm this action.
                </p>
              </AlertDescription>
            </Alert>
          )}

          {!isLastOrganization && (
            <Alert className="border-yellow-900/50 bg-yellow-950/20">
              <AlertDescription className="text-sm text-yellow-200">
                You will lose access to all projects and data in{' '}
                <strong>{organizationName}</strong>.
              </AlertDescription>
            </Alert>
          )}

          {isConfirmationRequired && (
            <div className="space-y-2">
              <Input
                value={confirmationInput}
                onChange={(e) => setConfirmationInput(e.target.value)}
                placeholder="Type LEAVE to confirm"
                disabled={state.status === 'confirming'}
              />
            </div>
          )}

          {state.status === 'error' && (
            <Alert className="border-red-900/50 bg-red-950/20">
              <AlertDescription className="text-sm text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => setIsOpen(false)}
            disabled={state.status === 'confirming'}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleLeave}
            disabled={!isConfirmationValid || state.status === 'confirming'}
          >
            {state.status === 'confirming' && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            <LogOut className="mr-2 h-4 w-4" />
            Leave Organization
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
